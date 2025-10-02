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
	"sync"
	"syscall"
	"time"
)

const Version = "1.2.10"

const DEFAULT_BIND_ADDRESS = "127.0.0.1"
const DEFAULT_HTTPS_PORT = 10080
const DEFAULT_SHUTDOWN_TIMEOUT_SECONDS = 5
const DEFAULT_AUTODELETE_INTERVAL_SECONDS = 60

var Verbose bool
var Debug bool

type WinexecServer struct {
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
	enableMenu             bool

	autoDeleteFiles           map[string]time.Time
	autoDeleteWaiter          sync.WaitGroup
	autoDeleteIntervalSeconds int
	autoDeleteStopRequest     chan struct{}

	startupCommand      string
	startupCommandArgs  []string
	shutdownCommand     string
	shutdownCommandArgs []string
}

func viperPrefix() string {
	prefix := "winexec.server."
	if ProgramName() == "winexec" {
		prefix = "server."
	}
	return prefix
}

func NewWinexecServer() (*WinexecServer, error) {
	prefix := viperPrefix()
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	configDir := filepath.Join(userConfigDir, ProgramName())
	ViperSetDefault(prefix+"bind_address", DEFAULT_BIND_ADDRESS)
	ViperSetDefault(prefix+"https_port", DEFAULT_HTTPS_PORT)
	ViperSetDefault(prefix+"ca", filepath.Join(configDir, "keymaster.pem"))
	ViperSetDefault(prefix+"cert", filepath.Join(configDir, "winexec-server-cert.pem"))
	ViperSetDefault(prefix+"key", filepath.Join(configDir, "winexec-server-key.pem"))
	ViperSetDefault(prefix+"shutdown_timeout_seconds", DEFAULT_SHUTDOWN_TIMEOUT_SECONDS)
	ViperSetDefault(prefix+"autodelete_interval_seconds", DEFAULT_AUTODELETE_INTERVAL_SECONDS)

	s := WinexecServer{
		Name:                      "winexec",
		Address:                   ViperGetString(prefix + "bind_address"),
		Port:                      ViperGetInt(prefix + "https_port"),
		Version:                   Version,
		started:                   make(chan struct{}),
		shutdownRequest:           make(chan struct{}),
		shutdownComplete:          make(chan struct{}),
		debug:                     ViperGetBool(prefix + "debug"),
		verbose:                   ViperGetBool(prefix + "verbose"),
		ca:                        ViperGetString(prefix + "ca"),
		cert:                      ViperGetString(prefix + "cert"),
		key:                       ViperGetString(prefix + "key"),
		shutdownTimeoutSeconds:    ViperGetInt(prefix + "shutdown_timeout_seconds"),
		autoDeleteIntervalSeconds: ViperGetInt(prefix + "autodelete_interval_seconds"),
		autoDeleteFiles:           make(map[string]time.Time),
		autoDeleteStopRequest:     make(chan struct{}),
		enableMenu:                ViperGetBool(prefix + "menu"),
		startupCommand:            ViperGetString(prefix + "startup_command"),
		startupCommandArgs:        ViperGetStringSlice(prefix + "startup_command_args"),
		shutdownCommand:           ViperGetString(prefix + "shutdown_command"),
		shutdownCommandArgs:       ViperGetStringSlice(prefix + "shutdown_command_args"),
	}
	Verbose = s.verbose
	Debug = s.debug
	if Debug {
		log.Printf("winexec server config: %s\n", FormatJSON(s.GetConfig()))
	}
	return &s, nil
}

func (s *WinexecServer) GetConfig() map[string]any {
	prefix := ViperKey(viperPrefix())
	cfg := make(map[string]any)
	for _, key := range viper.AllKeys() {
		if strings.HasPrefix(key, prefix) {
			cfg[key] = viper.Get(key)
		}
	}
	return cfg
}

func (s *WinexecServer) Start() error {
	log.Println("callingRunServer")
	go runServer(s)
	log.Println("awaiting 'started' message...")
	<-s.started
	log.Println("received 'started' message")
	if s.enableMenu {
		title := fmt.Sprintf("%s v%s", s.Name, s.Version)
		menu, err := NewMenu(title, s.shutdownRequest, s.shutdownComplete)
		if err != nil {
			return err
		}
		s.menu = menu
	}
	return nil
}

func (s *WinexecServer) Stop() error {
	log.Println("Stop: sending 'shutdownRequest' message")
	s.shutdownRequest <- struct{}{}
	log.Println("Stop: awaiting 'shutdownComplete' message...")
	<-s.shutdownComplete
	log.Println("Stop: received 'shutdownComplete' message")
	return nil
}

func (s *WinexecServer) Run() error {
	err := s.Start()
	if err != nil {
		return err
	}
	if ViperGetBool("verbose") {
		log.Println("CTRL-C to exit")
	}
	sigint := make(chan os.Signal)
	signal.Notify(sigint, syscall.SIGINT)
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, syscall.SIGTERM)
	select {
	case <-sigint:
		log.Println("\nreceived SIGINT")
		err = s.Stop()
		if err != nil {
			return err
		}
	case <-sigterm:
		log.Println("\nreceived SIGTERM")
		err = s.Stop()
		if err != nil {
			return err
		}
	case <-s.shutdownComplete:
		log.Println("\nreceived shutdownComplete")
	}
	return nil
}

func fail(w http.ResponseWriter, r *http.Request, failMessage string, status int) {
	response := message.FailResponse{
		Success: false,
		Message: failMessage,
	}
	if Verbose {
		log.Printf("%s <- winexec fail [%d] %s\n", r.RemoteAddr, status, failMessage)
	}
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		fail(w, r, "failed encoding response", http.StatusInternalServerError)
	}
}

func succeed(w http.ResponseWriter, r *http.Request, response interface{}) {
	if Verbose {
		log.Printf("%s <- winexec response [200] %+v\n", r.RemoteAddr, response)
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		fail(w, r, "failed encoding response", http.StatusInternalServerError)
	}
}

func runServer(s *WinexecServer) {

	if s.startupCommand != "" {
		err := s.runCommand("startup", s.startupCommand, s.startupCommandArgs...)
		if err != nil {
			Warning("startup command failed")
		}
	}

	serverCertPEM, err := os.ReadFile(s.cert)
	if err != nil {
		log.Fatalf("Failed reading server certificate: %v", err)
	}

	serverKeyPEM, err := os.ReadFile(s.key)
	if err != nil {
		log.Fatalf("Failed reading server certificate key: %v", err)
	}

	serverCert, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		log.Fatalf("Failed creating X509 keypair: %v", err)
	}

	caCertPEM, err := os.ReadFile(s.ca)
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

	listen := fmt.Sprintf("%s:%d", s.Address, s.Port)
	server := http.Server{
		Addr:      listen,
		TLSConfig: tlsConfig,
	}

	http.HandleFunc("GET /ping/", handlePing)
	http.HandleFunc("GET /os/", handleGetOS)
	http.HandleFunc("POST /exec/", handleExec)
	http.HandleFunc("POST /spawn/", handleSpawn)
	http.HandleFunc("POST /download/", handleFileDownload)
	http.HandleFunc("POST /upload/", handleFileUpload)
	http.HandleFunc("POST /delete/", handleFileDelete)
	http.HandleFunc("POST /dir/", handleDirectoryEntries)
	http.HandleFunc("POST /mkdir/", handleDirectoryCreate)
	http.HandleFunc("POST /rmdir/", handleDirectoryDestroy)
	http.HandleFunc("POST /get/", s.handleFileGet)
	http.HandleFunc("POST /isfile/", s.handleIsFile)
	http.HandleFunc("POST /isdir/", s.handleIsDir)

	log.Printf("%s v%s server listening on %s in TLS mode\n", s.Name, s.Version, server.Addr)
	go func() {
		err := server.ListenAndServeTLS("", "")
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("ServeTLS failed: %v", err)
		}
	}()

	go s.runAutoDelete()

	if s.verbose {
		log.Println("runServer: sending 'started'")
	}
	s.started <- struct{}{}
	if s.verbose {
		log.Println("runServer: sent 'started'")
	}

	defer func() {
		if s.verbose {
			log.Println("runServer.exit: sending 'shutdownComplete'")
		}
		s.shutdownComplete <- struct{}{}
		if s.verbose {
			log.Println("runServer.exit: sent 'shutdownComplete'")
		}
	}()

	if s.verbose {
		log.Println("runServer: awaiting 'shutdownRequest'")
	}
	<-s.shutdownRequest
	if s.verbose {
		log.Println("runServer: received 'shutdownRequest'")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.shutdownTimeoutSeconds)*time.Second)
	defer cancel()

	err = server.Shutdown(ctx)
	if err != nil {
		log.Fatalln("Server Shutdown failed: ", err)
	}

	s.stopAutoDelete()

	if s.shutdownCommand != "" {
		err := s.runCommand("shutdown", s.shutdownCommand, s.shutdownCommandArgs...)
		if err != nil {
			Warning("shutdown command failed")
		}
	}
}

func (s *WinexecServer) stopAutoDelete() {
	if s.verbose {
		log.Println("sending autoDeleteStopRequest")
	}
	s.autoDeleteStopRequest <- struct{}{}
	if s.verbose {
		log.Printf("awaiting autoDelete shutdown...")
	}
	s.autoDeleteWaiter.Wait()
	if s.verbose {
		log.Printf("autoDelete shutdown complete")
	}
}

func (s *WinexecServer) setAutoDelete(pathname string, seconds int) {
	if s.verbose {
		log.Printf("setAutoDelete(%s, %d)\n", pathname, seconds)
	}
	if seconds != 0 {
		s.autoDeleteFiles[pathname] = time.Now().Add(time.Duration(int64(seconds)) * time.Second)
	}
}

func (s *WinexecServer) checkAutoDelete(shutdown bool) {
	if s.debug {
		log.Printf("checkAutoDelete(shutdown=%v)\n", shutdown)
	}
	expiredFiles := []string{}
	for filename, expireTime := range s.autoDeleteFiles {
		if shutdown || time.Now().After(expireTime) {
			expiredFiles = append(expiredFiles, filename)
			log.Printf("autoDeleting: %s\n", filename)
			err := os.Remove(filename)
			if err != nil {
				Warning("autodelete failed: %v", Fatal(err))
			}
		}
	}
	for _, filename := range expiredFiles {
		delete(s.autoDeleteFiles, filename)
	}
}

func (s *WinexecServer) runAutoDelete() {
	s.autoDeleteWaiter.Add(1)
	defer s.autoDeleteWaiter.Done()
	if s.verbose {
		defer log.Println("runAutoDelete: exiting")
		log.Println("runAutoDelete: started")
	}
	ticker := time.NewTicker(time.Duration(s.autoDeleteIntervalSeconds) * time.Second)
	for {
		select {
		case <-s.autoDeleteStopRequest:
			ticker.Stop()
			if s.verbose {
				log.Printf("runAutoDelete: received autoDeleteStopRequest")
			}
			s.checkAutoDelete(true)
			return
		case <-ticker.C:
			s.checkAutoDelete(false)
		}
	}
}
