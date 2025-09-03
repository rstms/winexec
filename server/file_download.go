package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"os"
)

func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
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
	srcPathname := request.Pathname
	fileinfo, err := os.Stat(srcPathname)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "stat failed", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(srcPathname)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "read failed", http.StatusBadRequest)
		return
	}
	response := message.FileDownloadResponse{
		Success:   true,
		Message:   "download",
		Pathname:  srcPathname,
		Content:   data,
		Timestamp: fileinfo.ModTime(),
		Mode:      fileinfo.Mode(),
	}
	succeed(w, r, &response)
}
