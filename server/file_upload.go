package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"os"
	"time"
)

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.FileUploadRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "failed decoding request", http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	if IsFile(request.Pathname) {
		if !request.Force {
			Warning("file exists: '%s'", request.Pathname)
			fail(w, r, "file exists", http.StatusBadRequest)
			return
		}
	}

	err = os.WriteFile(request.Pathname, request.Content, request.Mode)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "write failed", http.StatusBadRequest)
		return
	}
	err = os.Chtimes(request.Pathname, time.Time{}, request.Timestamp)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "time update failed", http.StatusBadRequest)
		return

	}
	response := message.FileDownloadResponse{
		Success:  true,
		Message:  "uploaded",
		Pathname: request.Pathname,
	}
	succeed(w, r, &response)
}
