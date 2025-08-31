package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/rstms/winexec/message"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const Version = "1.1.4"

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
}

func viperPrefix() string {
	prefix := "winexec.server."
	if ProgramName() == "winexec" {
		prefix = "server."
	}
	return prefix
}

func NewDaemon() (*Daemon, error) {
	prefix := viperPrefix()
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(userConfigDir, "winexec")
	ViperSetDefault(prefix+"bind_address", DEFAULT_BIND_ADDRESS)
	ViperSetDefault(prefix+"https_port", DEFAULT_HTTPS_PORT)
	ViperSetDefault(prefix+"ca", filepath.Join(configDir, "ca.pem"))
	ViperSetDefault(prefix+"cert", filepath.Join(configDir, "cert.pem"))
	ViperSetDefault(prefix+"key", filepath.Join(configDir, "key.pem"))
	ViperSetDefault(prefix+"shutdown_timeout_seconds", DEFAULT_SHUTDOWN_TIMEOUT_SECONDS)

	d := Daemon{
		Name:                   "winexec",
		Address:                ViperGetString(prefix + "bind_address"),
		Port:                   ViperGetInt(prefix + "https_port"),
		Version:                Version,
		started:                make(chan struct{}),
		shutdownRequest:        make(chan struct{}),
		shutdownComplete:       make(chan struct{}),
		debug:                  ViperGetBool("debug"),
		verbose:                ViperGetBool("verbose"),
		ca:                     ViperGetString(prefix + "ca"),
		cert:                   ViperGetString(prefix + "cert"),
		key:                    ViperGetString(prefix + "key"),
		shutdownTimeoutSeconds: ViperGetInt(prefix + "shutdown_timeout_seconds"),
	}
	Verbose = d.verbose
	Debug = d.debug
	return &d, nil
}

func (d *Daemon) GetConfig() map[string]any {
	prefix := ViperKey(viperPrefix())
	cfg := make(map[string]any)
	for _, key := range viper.AllKeys() {
		if strings.HasPrefix(key, prefix) {
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

func fail(w http.ResponseWriter, r *http.Request, failMessage string, status int) {
	response := message.FailResponse{
		Success: false,
		Message: failMessage,
	}
	if Verbose {
		log.Printf("%s <- [%d] %s\n", r.RemoteAddr, status, failMessage)
	}
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		fail(w, r, "failed encoding response", http.StatusInternalServerError)
	}
}

func succeed(w http.ResponseWriter, r *http.Request, response interface{}) {
	if Verbose {
		log.Printf("%s <- [200] %+v\n", r.RemoteAddr, response)
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		fail(w, r, "failed encoding response", http.StatusInternalServerError)
	}
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
	http.HandleFunc("POST /download/", handleFileDownload)
	http.HandleFunc("POST /upload/", handleFileUpload)
	http.HandleFunc("POST /get/", handleFileGet)

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
