package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
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

	localized, err := filepath.Localize(request.Pathname)
	if err != nil {
		Warning("failed to localize path '%s': %v", request.Pathname, err)
		fail(w, r, "path localization failed", http.StatusBadRequest)
		return
	}
	err = os.WriteFile(localized, request.Content, request.Mode)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "write failed", http.StatusBadRequest)
		return
	}
	err = os.Chtimes(localized, time.Time{}, request.Timestamp)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "time update failed", http.StatusBadRequest)
		return

	}
	response := message.FileDownloadResponse{
		Success:  true,
		Message:  "uploaded",
		Pathname: localized,
	}
	succeed(w, r, &response)
}
