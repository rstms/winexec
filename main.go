package main

import (
	"context"
	_ "embed"
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

//go:embed server.crt
var serverCert []byte

//go:embed server.key
var serverKey []byte


const serverName = "winexec"
const defaultPort = 10080
const SHUTDOWN_TIMEOUT = 5
const Version = "0.0.3"

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

	// Ensure the program is run with a Windows GUI context
	runtime.LockOSThread()

	go runServer(addr, port)

	systray.Run(onReady, onExit)
}

func onReady() {
	// Set the icon and tooltip
	systray.SetIcon(iconData)
	title := fmt.Sprintf("winexec v%s", Version)
	systray.SetTitle(title)
	systray.SetTooltip(title)

	// Add menu items
	mQuit := systray.AddMenuItem("Quit", "Shutdown and exit")

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

type Response struct {
	Success bool
	Message string
	Data    interface{}
}

func fail(w http.ResponseWriter, message string, status int) {
	log.Printf("  [%d] %s", status, message)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{false, message, struct{}{}})
}

func succeed(w http.ResponseWriter, message string, status int, result interface{}) {
	log.Printf("  [%d] %s", status, message)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{true, message, result})
}

func checkClientCert(w http.ResponseWriter, r *http.Request) bool {
	if Debug {
		return true
	}
	usernameHeader, ok := r.Header["X-Client-Cert-Dn"]
	if !ok {
		fail(w, "missing client cert DN", http.StatusBadRequest)
		return false
	}
	if Verbose {
		log.Printf("client cert dn: %s\n", usernameHeader[0])
	}
	if usernameHeader[0] != "CN=winexec" {
		fail(w, fmt.Sprintf("client cert (%s) != filterctl", usernameHeader[0]), http.StatusBadRequest)
		return false
	}
	return true
}

/*
func sendClasses(w http.ResponseWriter, config *classes.SpamClasses, address string) {
	result := config.GetClasses(address)
	message := fmt.Sprintf("%s spam classes", address)
	succeed(w, message, http.StatusOK, result)
}

func handleGetClass(w http.ResponseWriter, r *http.Request) {
	if !checkClientCert(w, r) {
		return
	}
	address := r.PathValue("address")
	scoreParam := r.PathValue("score")
	log.Printf("GET address=%s score=%s\n", address, scoreParam)
	score, err := strconv.ParseFloat(scoreParam, 32)
	if err != nil {
		fail(w, "score conversion failed", http.StatusBadRequest)
		return
	}
	config, ok := readConfig(w)
	if ok {
		class := config.GetClass([]string{address}, float32(score))
		succeed(w, class, http.StatusOK, []classes.SpamClass{{class, float32(score)}})
	}
}

func handleGetClasses(w http.ResponseWriter, r *http.Request) {
	if !checkClientCert(w, r) {
		return
	}
	address := r.PathValue("address")
	log.Printf("GET address=%s\n", address)
	config, ok := readConfig(w)
	if ok {
		sendClasses(w, config, address)
	}
}

func handlePutClassThreshold(w http.ResponseWriter, r *http.Request) {
	if !checkClientCert(w, r) {
		return
	}
	address := r.PathValue("address")
	name := r.PathValue("name")
	threshold := r.PathValue("threshold")
	log.Printf("PUT address=%s name=%s threshold=%s\n", address, name, threshold)
	score, err := strconv.ParseFloat(threshold, 32)
	if err != nil {
		fail(w, "threshold conversion failed", http.StatusBadRequest)
		return
	}
	config, ok := readConfig(w)
	if ok {
		config.SetThreshold(address, name, float32(score))
		if writeConfig(w, config) {
			sendClasses(w, config, address)
		}
	}
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !checkClientCert(w, r) {
		return
	}
	address := r.PathValue("address")
	log.Printf("DELETE (user) address=%s\n", address)
	config, ok := readConfig(w)
	if ok {
		config.DeleteClasses(address)
		if writeConfig(w, config) {
			result := []classes.SpamClass{}
			succeed(w, "deleted", http.StatusOK, result)
		}
	}
}

func handleDeleteClass(w http.ResponseWriter, r *http.Request) {
	if !checkClientCert(w, r) {
		return
	}
	address := r.PathValue("address")
	name := r.PathValue("name")
	log.Printf("DELETE (class) address=%s name=%s\n", address, name)
	config, ok := readConfig(w)
	if ok {
		config.GetClasses(address)
		config.DeleteClass(address, name)
		if writeConfig(w, config) {
			sendClasses(w, config, address)
		}
	}
}
*/

func handleHello(w http.ResponseWriter, r *http.Request) {

	data := make(map[string]string)
	data["response"] = "pong"
	succeed(w, "hello", 200, data)
}

func runServer(addr *string, port *int) {

        certFile, err := ioutil.TempFile("", "server.crt")
        if err != nil {
            log.Fatal(err)
        }
        defer os.Remove(certFile.Name())
        _, err = certFile.Write(serverCert)
        if err != nil {
            log.Fatal(err)
        }
        certFile.Close()

        keyFile, err := ioutil.TempFile("", "server.key")
        if err != nil {
            log.Fatal(err)
        }
        defer os.Remove(keyFile.Name())
        _, err = keyFile.Write(serverKey)
        if err != nil {
            log.Fatal(err)
        }
        keyFile.Close()


	listen := fmt.Sprintf("%s:%d", *addr, *port)
	server := http.Server{
		Addr: listen,
	}

	http.HandleFunc("GET /ping/", handleHello)
	/*
		http.HandleFunc("GET /filterctl/classes/{address}", handleGetClasses)
		http.HandleFunc("GET /filterctl/class/{address}/{score}", handleGetClass)
		http.HandleFunc("PUT /filterctl/classes/{address}/{name}/{threshold}", handlePutClassThreshold)
		http.HandleFunc("DELETE /filterctl/classes/{address}", handleDeleteUser)
		http.HandleFunc("DELETE /filterctl/classes/{address}/{name}", handleDeleteClass)
	*/

	go func() {
		log.Printf("%s v%s listening on %s\n", serverName, Version, listen)
                err := server.ListenAndServeTLS(certFile.Name(), keyFile.Name())
		if err != nil && err != http.ErrServerClosed {
			log.Fatalln("ListenAndServeTLS failed: ", err)
		}
	}()

	<-ShutdownRequest

	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), SHUTDOWN_TIMEOUT*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatalln("Server Shutdown failed: ", err)
	}
	log.Println("shutdown complete")
	ShutdownComplete <- struct{}{}
}
