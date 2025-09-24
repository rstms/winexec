package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
	"github.com/rstms/winexec/ospath"
	"log"
	"net/http"
	"os"
)

func failIfDir(pathname string, w http.ResponseWriter, r *http.Request) bool {
	if IsDir(pathname) {
		Warning("directory exists: %s", pathname)
		fail(w, r, "directory exists", http.StatusBadRequest)
		return true
	}
	return false
}

func failIfNotDir(pathname string, w http.ResponseWriter, r *http.Request) bool {
	if !IsDir(pathname) {
		Warning("not a directory: %s", pathname)
		fail(w, r, "not a directory", http.StatusBadRequest)
		return true
	}
	return false
}

func handleDirectoryCreate(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.DirectoryCreateRequest
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

	response := message.DirectoryResponse{
		Success:  true,
		Pathname: pathname,
	}
	if failIfDir(pathname, w, r) {
		return
	}
	err = os.MkdirAll(pathname, request.Mode)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "create failed", http.StatusBadRequest)
		return
	}
	response.Message = "created"
	succeed(w, r, &response)
}

func handleDirectoryDestroy(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.DirectoryRequest
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

	response := message.DirectoryResponse{
		Success:  true,
		Pathname: pathname,
		Message:  "destroyed",
	}
	if failIfNotDir(pathname, w, r) {
		return
	}
	err = os.RemoveAll(pathname)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "destroy failed", http.StatusBadRequest)
		return
	}
	succeed(w, r, &response)
}

func handleDirectoryEntries(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.DirectoryRequest
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

	if failIfNotDir(pathname, w, r) {
		return
	}

	response := message.DirectoryResponse{
		Success:  true,
		Pathname: pathname,
		Message:  "entries",
		Entries:  make(map[string]message.DirectoryEntry),
	}
	entries, err := os.ReadDir(pathname)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "failed reading directory", http.StatusInternalServerError)
		return
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			Warning("%v", Fatal(err))
			fail(w, r, "failed reading entry info", http.StatusInternalServerError)
			return
		}
		response.Entries[entry.Name()] = message.DirectoryEntry{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    entry.Type(),
		}
	}
	succeed(w, r, &response)
}
