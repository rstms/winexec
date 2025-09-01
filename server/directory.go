package server

import (
	"encoding/json"
	"github.com/rstms/winexec/message"
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

func failNoOperation(w http.ResponseWriter, r *http.Request) {
	Warning("no directory operation")
	fail(w, r, "no operation selected", http.StatusBadRequest)
}

func failIfMultipleOps(request *message.DirectoryRequest, w http.ResponseWriter, r *http.Request) bool {
	switch {
	case request.List:
		if !request.Create && !request.Destroy {
			return false
		}
	case request.Create:
		if !request.List && !request.Destroy {
			return false
		}
	case request.Destroy:
		if !request.List && !request.Create {
			return false
		}
	default:
		failNoOperation(w, r)
		return true
	}
	Warning("directory operation conflict")
	fail(w, r, "multiple operations requested", http.StatusBadRequest)

	return true
}

func handleDirectoryOps(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
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

	if failIfMultipleOps(&request, w, r) {
		return
	}

	response := message.DirectoryResponse{
		Success:  true,
		Pathname: request.Pathname,
	}
	switch {
	case request.List:
		if failIfNotDir(request.Pathname, w, r) {
			return
		}
		err := listDirectory(request.Pathname, request.Detail, &response)
		if err != nil {
			Warning("listDirectory failed: %v", err)
			fail(w, r, "list failed", http.StatusInternalServerError)
			return
		}
		response.Message = "listed"
	case request.Create:
		if failIfDir(request.Pathname, w, r) {
			return
		}
		err := os.MkdirAll(request.Pathname, request.Mode)
		if err != nil {
			Warning("%v", Fatal(err))
			fail(w, r, "create failed", http.StatusBadRequest)
			return
		}
		response.Message = "created"
	case request.Destroy:
		if failIfNotDir(request.Pathname, w, r) {
			return
		}
		err := os.RemoveAll(request.Pathname)
		if err != nil {
			Warning("%v", Fatal(err))
			fail(w, r, "destroy failed", http.StatusBadRequest)
			return
		}
		response.Message = "destroyed"
	default:
		failNoOperation(w, r)
		return
	}
	succeed(w, r, &response)
}

func listDirectory(pathname string, detail bool, response *message.DirectoryResponse) error {
	entries, err := os.ReadDir(pathname)
	if err != nil {
		return Fatal(err)
	}
	for _, entry := range entries {
		switch {
		case entry.IsDir():
			line := entry.Name()
			if detail {
				info, err := entry.Info()
				if err != nil {
					return Fatal(err)
				}
				line = FormatJSON(info)
			}
			response.Dirs = append(response.Dirs, line)
		case entry.Type().IsRegular():
			line := entry.Name()
			if detail {
				info, err := entry.Info()
				if err != nil {
					return Fatal(err)
				}
				line = FormatJSON(info)
			}
			response.Files = append(response.Files, line)
		}
	}
	return nil
}
