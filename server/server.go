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

const Version = "1.1.14"

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

func NewWinexecServer() (*Daemon, error) {
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
	if Debug {
		log.Printf("winexec server config: %s\n", FormatJSON(d.GetConfig()))
	}
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
	log.Println("callingRunServer")
	go runServer(d)
	log.Println("awaiting 'started' message...")
	<-d.started
	log.Println("received 'started' message")
	title := fmt.Sprintf("%s v%s", d.Name, d.Version)
	menu, err := NewMenu(title, d.shutdownRequest, d.shutdownComplete)
	if err != nil {
		return err
	}
	d.menu = menu
	return nil
}

func (d *Daemon) Stop() error {
	log.Println("Stop: sending 'shutdownRequest' message")
	d.shutdownRequest <- struct{}{}
	log.Println("Stop: awaiting 'shutdownComplete' message...")
	<-d.shutdownComplete
	log.Println("Stop: received 'shutdownComplete' message")
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
	select {
	case <-sigint:
		log.Println("Run: received SIGINT")
		err = d.Stop()
		if err != nil {
			return err
		}
	case <-d.shutdownComplete:
		log.Println("Run: received shutdownComplete")
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
	http.HandleFunc("POST /dir/", handleDirectoryEntries)
	http.HandleFunc("POST /mkdir/", handleDirectoryCreate)
	http.HandleFunc("POST /rmdir/", handleDirectoryDestroy)

	log.Printf("%s v%s server listening on %s in TLS mode\n", d.Name, d.Version, server.Addr)
	go func() {
		err := server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("ServeTLS failed: %v", err)
		}
	}()

	log.Println("runServer: sending 'started'")
	d.started <- struct{}{}
	log.Println("runServer: sent 'started'")

	defer func() {
		log.Println("runServer.exit: sending 'shutdownComplete'")
		d.shutdownComplete <- struct{}{}
		log.Println("runServer.exit: sent 'shutdownComplete'")
	}()

	log.Println("runServer: awaiting 'shutdownRequest'")
	<-d.shutdownRequest
	log.Println("runServer: received 'shutdownRequest'")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.shutdownTimeoutSeconds)*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatalln("Server Shutdown failed: ", err)
	}
}
