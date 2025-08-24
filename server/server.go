package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const Version = "1.0.8"

const DEFAULT_BIND_ADDRESS = "127.0.0.1"
const DEFAULT_HTTPS_PORT = 10080
const DEFAULT_SHUTDOWN_TIMEOUT_SECONDS = 5

var Verbose bool
var Debug bool

type Daemon struct {
	Name                   string
	Address                string
	Version                string
	Port                   int
	started                chan struct{}
	shutdownRequest        chan struct{}
	shutdownComplete       chan struct{}
	menu                   *Menu
	ca                     string
	cert                   string
	key                    string
	shutdownTimeoutSeconds int
	debug                  bool
	verbose                bool
	viperPrefix            string
}

func NewDaemon(viperKey string) (*Daemon, error) {
	if viperKey == "" {
		viperKey = "winexec"
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	viper.SetDefault(viperKey+".bind_address", DEFAULT_BIND_ADDRESS)
	viper.SetDefault(viperKey+".https_port", DEFAULT_HTTPS_PORT)
	viper.SetDefault(viperKey+".ca", filepath.Join(configDir, "winexec", "ca.pem"))
	viper.SetDefault(viperKey+".cert", filepath.Join(configDir, "winexec", "cert.pem"))
	viper.SetDefault(viperKey+".key", filepath.Join(configDir, "winexec", "key.pem"))
	viper.SetDefault(viperKey+".shutdown_timeout_seconds", DEFAULT_SHUTDOWN_TIMEOUT_SECONDS)
	viper.SetDefault(viperKey+".debug", false)
	viper.SetDefault(viperKey+".verbose", false)

	d := Daemon{
		Name:                   "winexec",
		Address:                viper.GetString(viperKey + ".bind_address"),
		Port:                   viper.GetInt(viperKey + ".https_port"),
		Version:                Version,
		started:                make(chan struct{}),
		shutdownRequest:        make(chan struct{}),
		shutdownComplete:       make(chan struct{}),
		debug:                  viper.GetBool(viperKey + ".debug"),
		verbose:                viper.GetBool(viperKey + ".verbose"),
		ca:                     viper.GetString(viperKey + ".ca"),
		cert:                   viper.GetString(viperKey + ".cert"),
		key:                    viper.GetString(viperKey + ".key"),
		shutdownTimeoutSeconds: viper.GetInt(viperKey + ".shutdown_timeout_seconds"),
		viperPrefix:            viperKey,
	}
	Verbose = d.verbose
	Debug = d.debug
	return &d, nil
}

func (d *Daemon) GetConfig() map[string]any {
	cfg := make(map[string]any)
	for _, key := range viper.AllKeys() {
		if strings.HasPrefix(key, d.viperPrefix+".") {
			cfg[key] = viper.Get(key)
		}
	}
	return cfg
}

func (d *Daemon) Start() error {
	go runServer(d)
	title := fmt.Sprintf("%s v%s", d.Name, d.Version)
	menu, err := NewMenu(title, d.shutdownRequest, d.shutdownComplete)
	if err != nil {
		return err
	}
	d.menu = menu
	<-d.started
	return nil
}

func (d *Daemon) Stop() error {
	d.shutdownRequest <- struct{}{}
	<-d.shutdownComplete
	return nil
}

func (d *Daemon) Run(message string) error {
	err := d.Start()
	if err != nil {
		return err
	}
	sigint := make(chan os.Signal)
	signal.Notify(sigint, syscall.SIGINT)
	if message != "" {
		fmt.Println(message)
	}
	<-sigint
	err = d.Stop()
	if err != nil {
		return err
	}
	return nil
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

func fail(w http.ResponseWriter, r *http.Request, message string) {
	status := http.StatusBadRequest
	response := FailResponse{false, message}
	if Verbose {
		log.Printf("%s <- [%d] %s\n", r.RemoteAddr, status, message)
	}
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func succeed(w http.ResponseWriter, r *http.Request, response interface{}) {
	if Verbose {
		log.Printf("%s <- [200] %+v\n", r.RemoteAddr, response)
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		fail(w, r, err.Error())
	}
}

func handleExec(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request ExecRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	exit, stdout, stderr, err := run(request.Command, request.Args...)
	if err != nil {
		fail(w, r, err.Error())
		return
	}
	response := ExecResponse{
		true,
		request.Command,
		exit,
		stdout,
		stderr,
	}
	succeed(w, r, &response)
}

func handleSpawn(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request ExecRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	command := append([]string{request.Command}, request.Args...)
	exitCode, err := spawn(strings.Join(command, " "))
	if err != nil {
		fail(w, r, err.Error())
		return
	}
	response := SpawnResponse{
		true,
		request.Command,
		exitCode,
	}
	succeed(w, r, &response)
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	response := PingResponse{true, "pong"}
	succeed(w, r, &response)
}

func readPEM(filename string) ([]byte, error) {
	filename = filepath.Clean(filename)
	if strings.HasPrefix(filename, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return []byte{}, err
		}
		filename = filepath.Join(homeDir, filename[1:])
	}
	return os.ReadFile(filename)
}

func runServer(d *Daemon) {

	serverCertPEM, err := readPEM(d.cert)
	if err != nil {
		log.Fatalf("Failed reading server certificate: %v", err)
	}

	serverKeyPEM, err := readPEM(d.key)
	if err != nil {
		log.Fatalf("Failed reading server certificate key: %v", err)
	}

	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		log.Fatalf("Failed creating X509 keypair: %v", err)
	}

	caCertPEM, err := readPEM(d.ca)
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

	listen := fmt.Sprintf("%s:%d", d.Address, d.Port)
	server := http.Server{
		Addr:      listen,
		TLSConfig: tlsConfig,
	}

	http.HandleFunc("GET /ping/", handlePing)
	http.HandleFunc("POST /exec/", handleExec)
	http.HandleFunc("POST /spawn/", handleSpawn)

	log.Printf("%s v%s server listening on %s in TLS mode\n", d.Name, d.Version, server.Addr)
	go func() {
		err := server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("ServeTLS failed: %v", err)
		}
	}()

	d.started <- struct{}{}

	<-d.shutdownRequest
	log.Println("received shutdown request")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.shutdownTimeoutSeconds)*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatalln("Server Shutdown failed: ", err)
	}
	log.Println("server shutdown complete")
	d.shutdownComplete <- struct{}{}
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
