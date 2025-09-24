package message

import (
	"io/fs"
	"time"
)

type FailResponse struct {
	Success bool
	Message string
}

type SuccessResponse struct {
	Success bool
	Message string
}

type ExecRequest struct {
	Command string
	Args    []string
	Env     []string
}

type ExecResponse struct {
	Success  bool
	Message  string
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
}

type SpawnRequest struct {
	Command string
	Args    []string
	Env     []string
}

type SpawnResponse struct {
	Success  bool
	Message  string
	Command  string
	ExitCode int
}

type FileGetRequest struct {
	Pathname          string
	URL               string
	CA                []byte
	Cert              []byte
	Key               []byte
	AutoDeleteSeconds int
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

type FileResponse struct {
	Success  bool
	Message  string
	Pathname string
}

type FileDeleteRequest struct {
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

type GetOSResponse struct {
	Success bool
	Message string
	OS      string
}

type IsRequest struct {
	Pathname string
}

type IsResponse struct {
	Success  bool
	Message  string
	Pathname string
	Result   bool
}

type NetbootConfig struct {
	Address           string `json:"address"`
	OS                string `json:"os"`
	Version           string `json:"version"`
	Arch              string `json:"arch"`
	Serial            string `json:"serial"`
	Mirror            string `json:"mirror"`
	Response          string `json:"response"`
	DisklabelTemplate string `json:"disklabel_template"`
	KernelParams      string `json:"kernel_params"`
	Debug             bool   `json:"debug"`
	Quiet             bool   `json:"quiet"`
}
