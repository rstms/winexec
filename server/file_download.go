package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.FileDownloadRequest
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
	fileinfo, err := os.Stat(localized)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "stat failed", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(localized)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "read failed", http.StatusBadRequest)
		return
	}
	response := message.FileDownloadResponse{
		Success:   true,
		Message:   "download",
		Pathname:  localized,
		Content:   data,
		Timestamp: fileinfo.ModTime(),
		Mode:      fileinfo.Mode(),
	}
	succeed(w, r, &response)
}
