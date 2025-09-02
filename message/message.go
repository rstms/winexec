package message

import (
	"io/fs"
	"time"
)

type FailResponse struct {
	Success bool
	Message string
}

type PingResponse struct {
	Success bool
	Message string
}

type ExecRequest struct {
	Command string
	Args    []string
}

type ExecResponse struct {
	Success  bool
	Message  string
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
}

type SpawnResponse struct {
	Success  bool
	Message  string
	Command  string
	ExitCode int
}

type FileGetRequest struct {
	Pathname string
	URL      string
	CA       []byte
	Cert     []byte
	Key      []byte
}

type FileGetResponse struct {
	Success  bool
	Message  string
	Pathname string
	Bytes    int64
}

type FileDownloadRequest struct {
	Pathname string
}

type FileDownloadResponse struct {
	Success   bool
	Message   string
	Pathname  string
	Content   []byte
	Timestamp time.Time
	Mode      fs.FileMode
}

type FileUploadRequest struct {
	Pathname  string
	Content   []byte
	Timestamp time.Time
	Mode      fs.FileMode
	Force     bool
}

type FileUploadResponse struct {
	Success  bool
	Message  string
	Pathname string
}

type DirectoryRequest struct {
	Pathname string
}

type DirectoryCreateRequest struct {
	Pathname string
	Mode     fs.FileMode
}

type DirectoryDestroyRequest struct {
	Pathname string
}

type DirectoryEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
	Mode    fs.FileMode
}

type DirectoryResponse struct {
	Success  bool
	Message  string
	Pathname string
	Entries  map[string]DirectoryEntry
}
