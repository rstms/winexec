package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/getlantern/systray"
	"log"
	"net/http"
	"os"
	"runtime"
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

var ShutdownRequest chan struct{}
var ShutdownComplete chan struct{}

func main() {
	addr := flag.String("addr", "0.0.0.0", "listen address")
	port := flag.Int("port", defaultPort, "listen port")
	debugFlag := flag.Bool("debug", false, "run in foreground mode")
	verboseFlag := flag.Bool("verbose", false, "verbose mode")
	versionFlag := flag.Bool("version", false, "output version")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s v%s\n", os.Args[0], Version)
		os.Exit(0)
	}

	Verbose = *verboseFlag
	Debug = *debugFlag

	ShutdownRequest = make(chan struct{})
	ShutdownComplete = make(chan struct{})

	go runServer(addr, port)

	// Ensure the program is run with a Windows GUI context
	runtime.LockOSThread()
	systray.Run(onReady, onExit)
}

func onReady() {
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

func onExit() {
	// Clean up here
	ShutdownRequest <- struct{}{}
	<-ShutdownComplete
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

type SpawnRequest struct {
	Command string
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

	exit, stdout, stderr, err := Run(request.Command, request.Args...)
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
	var request SpawnRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	exitCode, err := Spawn(request.Command)
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

func runServer(addr *string, port *int) {

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

	listen := fmt.Sprintf("%s:%d", *addr, *port)
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
