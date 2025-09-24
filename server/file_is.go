package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
	"github.com/rstms/winexec/ospath"
	"log"
	"net/http"
)

func (s *WinexecServer) handleIsFile(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.IsRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "failed decoding request", http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}
	pathname := ospath.LocalPath(request.Pathname)

	response := message.IsResponse{
		Success:  true,
		Message:  "isfile",
		Pathname: pathname,
		Result:   IsFile(pathname),
	}
	succeed(w, r, &response)
}

func (s *WinexecServer) handleIsDir(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.IsRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "failed decoding request", http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	pathname := ospath.LocalPath(request.Pathname)

	response := message.IsResponse{
		Success:  true,
		Message:  "isdir",
		Pathname: pathname,
		Result:   IsDir(pathname),
	}
	succeed(w, r, &response)
}
