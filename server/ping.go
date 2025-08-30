package server

import (
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
)

func handlePing(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	response := message.PingResponse{
		Success: true,
		Message: "pong",
	}
	succeed(w, r, &response)
}
