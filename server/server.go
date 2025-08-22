package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/getlantern/systray"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

//go:embed icon.ico
var iconData []byte

//go:embed certs/*
var certs embed.FS

const serverName = "winexec"
const defaultPort = 10080
const SHUTDOWN_TIMEOUT = 5
const Version = "1.0.2"

var Verbose bool
var Debug bool

type Daemon struct {
	Address          string
	Port             int
	shutdownRequest  chan struct{}
	shutdownComplete chan struct{}
}

func NewDaemon(addr string, port int, debug, verbose bool) (*Daemon, error) {
	Verbose = verbose
	Debug = debug
	d := Daemon{
		Address:          addr,
		Port:             port,
		shutdownRequest:  make(chan struct{}),
		shutdownComplete: make(chan struct{}),
	}
	return &d, nil
}

func (d *Daemon) Start() error {

	go runServer(d.Address, d.Port, d.shutdownRequest, d.shutdownComplete)

	// Ensure the program is run with a Windows GUI context
	runtime.LockOSThread()
	systray.Run(d.onReady, d.onExit)

	return nil
}

func (d *Daemon) Stop() error {
	d.shutdownRequest <- struct{}{}
	<-d.shutdownComplete
	return nil
}

func (d *Daemon) onReady() {
	// Set the icon and tooltip
	title := fmt.Sprintf("winexec v%s", Version)
	systray.SetTitle(title)
	systray.SetTooltip(title)
	systray.SetIcon(iconData)

	// Add menu items
	mQuit := systray.AddMenuItem(fmt.Sprintf("Quit %v", title), "Shutdown and exit")

	// Handle menu item clicks
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

}

func (d *Daemon) onExit() {
	// Clean up here
	d.shutdownRequest <- struct{}{}
	<-d.shutdownComplete
}

type FailResponse struct {
	Success bool
	Error   string
}

type PingResponse struct {
	Success bool
	Message string
}

type ExecRequest struct {
	Command string
	Args    []string
}

type ExecResponse struct {
	Success  bool
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
}

type SpawnResponse struct {
	Success  bool
	Command  string
	ExitCode int
}

func fail(w http.ResponseWriter, message string) {
	status := http.StatusBadRequest
	response := FailResponse{false, message}
	if Verbose {
		log.Printf(" fail: [%d] %s\n", status, message)
	}
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func succeed(w http.ResponseWriter, response interface{}) {
	if Verbose {
		log.Printf("  response: [200] %+v\n", response)
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		fail(w, err.Error())
	}
}

func handleExec(w http.ResponseWriter, r *http.Request) {
	var request ExecRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	exit, stdout, stderr, err := run(request.Command, request.Args...)
	if err != nil {
		fail(w, err.Error())
		return
	}
	response := ExecResponse{
		true,
		request.Command,
		exit,
		stdout,
		stderr,
	}
	succeed(w, &response)
}

func handleSpawn(w http.ResponseWriter, r *http.Request) {
	var request ExecRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	command := append([]string{request.Command}, request.Args...)
	exitCode, err := spawn(strings.Join(command, " "))
	if err != nil {
		fail(w, err.Error())
		return
	}
	response := SpawnResponse{
		true,
		request.Command,
		exitCode,
	}
	succeed(w, &response)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	response := PingResponse{true, "pong"}
	succeed(w, &response)
}

func runServer(addr string, port int, ShutdownRequest, ShutdownComplete chan struct{}) {

	serverCertPEM, err := certs.ReadFile("certs/cert.pem")
	if err != nil {
		log.Fatalf("Failed reading server certificate: %v", err)
	}

	serverKeyPEM, err := certs.ReadFile("certs/key.pem")
	if err != nil {
		log.Fatalf("Failed reading server certificate key: %v", err)
	}

	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		log.Fatalf("Failed creating X509 keypair: %v", err)
	}

	caCertPEM, err := certs.ReadFile("certs/ca.pem")
	if err != nil {
		log.Fatalf("Failed reading CA file: %v", err)
	}

	caCertPool := x509.NewCertPool()
	ok := caCertPool.AppendCertsFromPEM(caCertPEM)
	if !ok {
		log.Fatalf("failed appending CA cert to pool")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
	}

	listen := fmt.Sprintf("%s:%d", addr, port)
	server := http.Server{
		Addr:      listen,
		TLSConfig: tlsConfig,
	}

	http.HandleFunc("GET /ping/", handlePing)
	http.HandleFunc("POST /exec/", handleExec)
	http.HandleFunc("POST /spawn/", handleSpawn)

	go func() {
		log.Printf("%s v%s listening on %s\n", serverName, Version, listen)
		err := server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln("ServeTLS failed: ", err)
		}
	}()

	<-ShutdownRequest

	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), SHUTDOWN_TIMEOUT*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatalln("Server Shutdown failed: ", err)
	}
	log.Println("shutdown complete")
	ShutdownComplete <- struct{}{}
}

func spawn(command string) (int, error) {
	cmd := exec.Command("cmd", "/c", "start "+command)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	var exitCode int
	if Debug {
		log.Printf("Spawn: %v\n", cmd)
	}
	err := cmd.Run()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			exitCode = e.ExitCode()
			err = nil
		}
	}
	if Debug {
		log.Printf("exitCode=%d\n", exitCode)
		log.Printf("err=%v\n", err)
	}
	return exitCode, err
}

func run(command string, args ...string) (int, string, string, error) {
	if Debug {
		log.Printf("Run: %s %v\n", command, args)
	}
	cmd := exec.Command(command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	exitCode := 0
	err := cmd.Run()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			exitCode = e.ExitCode()
			err = nil
		default:
			return -1, "", "", err
		}
	}
	if Debug {
		log.Printf("exitCode=%d\n", exitCode)
		log.Printf("stdout=%s\n", stdout.String())
		log.Printf("stderr=%s\n", stderr.String())
		log.Printf("err=%v\n", err)
	}
	return exitCode, stdout.String(), stderr.String(), err
}
