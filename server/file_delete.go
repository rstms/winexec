package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
	"github.com/rstms/winexec/ospath"
	"log"
	"net/http"
	"os"
)

func handleFileDelete(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.FileDeleteRequest
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
	response := message.FileResponse{
		Success:  true,
		Message:  "deleted",
		Pathname: pathname,
	}
	if IsFile(pathname) {
		err := os.Remove(pathname)
		if err != nil {
			Warning("%v", Fatal(err))
			fail(w, r, "delete failed", http.StatusBadRequest)
			return
		}
	} else {
		response.Message = "file not present"
	}
	succeed(w, r, &response)
}
