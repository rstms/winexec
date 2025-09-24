package server

import (
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"runtime"
)

func handlePing(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	response := message.SuccessResponse{
		Success: true,
		Message: "pong",
	}
	succeed(w, r, &response)
}

func handleGetOS(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	response := message.GetOSResponse{
		Success: true,
		Message: "os",
		OS:      runtime.GOOS,
	}
	succeed(w, r, &response)
}
