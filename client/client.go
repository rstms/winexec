package client

import (
	"fmt"
	"github.com/rstms/winexec/message"
	"github.com/rstms/winexec/server"
	"github.com/spf13/viper"
	"io/fs"
	"log"
	"net/url"
	"os"
	"slices"
	"strings"
)

const Version = "1.2.7"

const DEFAULT_AUTO_DELETE_SECONDS = 300

type WinexecClient struct {
	url               string
	debug             bool
	AutoDeleteSeconds int
	certSubject       string
	certDuration      string
	api               APIClient
	server            *server.WinexecServer
}

func viperPrefix() string {
	prefix := "winexec.client."
	if ProgramName() == "winexec" {
		prefix = "client."
	}
	return prefix
}

func NewWinexecClient(caFile, certFile, keyFile string) (*WinexecClient, error) {

	prefix := viperPrefix()
	defaultURL := &url.URL{
		Scheme: "https",
		Host:   fmt.Sprintf("%s:%d", server.DEFAULT_BIND_ADDRESS, server.DEFAULT_HTTPS_PORT),
	}
	var err error
	if ViperGetString(prefix+"url") != "" {
		defaultURL, err = url.Parse(ViperGetString(prefix + "url"))
		if err != nil {
			return nil, Fatal(err)
		}
	}
	ViperSetDefault(prefix+"scheme", defaultURL.Scheme)
	ViperSetDefault(prefix+"hostname", defaultURL.Hostname())
	ViperSetDefault(prefix+"https_port", defaultURL.Port())
	ViperSetDefault(prefix+"path", defaultURL.Path)

	winexecURL := url.URL{
		Scheme: ViperGetString(prefix + "scheme"),
		Host:   ViperGetString(prefix + "hostname"),
		Path:   ViperGetString(prefix + "path"),
	}
	switch ViperGetString(prefix + "https_port") {
	case "", "443":
	default:
		winexecURL.Host += ":" + ViperGetString(prefix+"https_port")
	}
	client := WinexecClient{
		url:               winexecURL.String(),
		debug:             ViperGetBool(prefix + "debug"),
		AutoDeleteSeconds: ViperGetInt(prefix + "auto_delete_seconds"),
	}

	client.api, err = NewAPIClient("winexec", client.url, certFile, keyFile, caFile, nil)
	if err != nil {
		return nil, Fatal(err)
	}

	if client.debug {
		log.Printf("NewWinexecClient: %+v\n", client)
	}

	if ViperGetBool(prefix + "enable_winexec_server") {
		client.server, err = server.NewWinexecServer()
		if err != nil {
			return nil, Fatal(err)
		}
	}

	return &client, nil
}

func (c *WinexecClient) Close() error {
	if c.server != nil {
		err := c.server.Stop()
		if err != nil {
			return Fatal(err)
		}
	}
	return nil
}

func (c *WinexecClient) GetConfig() map[string]any {
	prefix := ViperKey(viperPrefix()) + "."
	cfg := make(map[string]any)
	for _, key := range viper.AllKeys() {
		if strings.HasPrefix(key, prefix) {
			cfg[key] = viper.Get(key)
		}
	}
	return cfg
}

func (c *WinexecClient) Spawn(command string, args, env []string, exitCode *int) error {
	if c.debug {
		log.Printf("winexec Spawn(%s)\n", command)
	}
	request := message.SpawnRequest{Command: command, Args: args, Env: env}
	var response message.SpawnResponse
	if c.debug {
		log.Printf("winexec spawn request: %+v\n", request)
	}

	_, err := c.api.Post("/spawn/", &request, &response, nil)
	if err != nil {
		return Fatal(err)
	}
	if c.debug {
		log.Printf("winexec spawn response: %+v\n", response)
	}
	if !response.Success {
		return Fatalf("WinExec: spawn failed: %v", response)
	}
	if exitCode != nil {
		*exitCode = response.ExitCode
	} else if response.ExitCode != 0 {
		return Fatalf("Spawning process '%s' exited %d", command, response.ExitCode)
	}
	return nil
}

func (c *WinexecClient) Exec(command string, args, env []string, exitCode *int) (string, string, error) {
	if c.debug {
		log.Printf("winexec Exec(%s %v)\n", command, args)
	}
	request := message.ExecRequest{Command: command, Args: args, Env: env}
	var response message.ExecResponse
	if c.debug {
		log.Printf("winexec exec request: %+v\n", request)
	}
	_, err := c.api.Post("/exec/", &request, &response, nil)
	if err != nil {
		return "", "", Fatal(err)
	}
	if c.debug {
		log.Printf("winexec exec response: %+v\n", response)
	}
	if !response.Success {
		return "", "", Fatalf("WinExec: exec failed: %v", response)
	}
	if exitCode != nil {
		*exitCode = response.ExitCode
	} else if response.ExitCode != 0 {
		return "", "", Fatalf("Process '%s' exited %d\n%s", command, response.ExitCode, response.Stderr)
	}
	return response.Stdout, response.Stderr, nil
}

func (c *WinexecClient) Upload(dst, src string, force bool) error {
	if c.debug {
		log.Printf("winexec Upload(%s %s)\n", dst, src)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return Fatal(err)
	}

	fileinfo, err := os.Stat(src)
	if err != nil {
		return Fatal(err)
	}
	request := message.FileUploadRequest{
		Pathname:  dst,
		Content:   data,
		Timestamp: fileinfo.ModTime(),
		Mode:      fileinfo.Mode(),
		Force:     force,
	}
	var response message.FileDownloadResponse
	if c.debug {
		log.Printf("winexec upload request: %+v\n", request)
	}
	_, err = c.api.Post("/upload/", &request, &response, nil)
	if err != nil {
		return Fatal(err)
	}
	if c.debug {
		log.Printf("winexec upload response: %+v\n", response)
	}
	if !response.Success {
		return Fatalf("WinExec: Upload failed: %v", response)
	}
	return nil
}

func (c *WinexecClient) Download(dst, src string) error {
	if c.debug {
		log.Printf("winexec Download(%s %s)\n", dst, src)
	}
	request := message.FileDownloadRequest{
		Pathname: src,
	}
	if c.debug {
		log.Printf("winexec download request: %+v\n", request)
	}
	var response message.FileDownloadResponse
	_, err := c.api.Post("/download/", &request, &response, nil)
	if err != nil {
		return Fatal(err)
	}
	if c.debug {
		log.Printf("winexec download response: %+v\n", response)
	}
	if !response.Success {
		return Fatalf("WinExec: Download failed: %v", response)
	}
	err = os.WriteFile(dst, response.Content, 0600)
	if err != nil {
		return Fatal(err)
	}
	return nil
}

func (c *WinexecClient) GetISO(dst, url, ca, cert, key string, autoDeleteSeconds *int) error {
	var seconds int
	seconds = c.AutoDeleteSeconds
	if autoDeleteSeconds != nil {
		seconds = *autoDeleteSeconds
	}
	if c.debug {
		log.Printf("winexec GetISO(dst=%s, url=%s, ca=%s, cert=%s, key=%s, seconds=%d)\n", dst, url, ca, cert, key, seconds)
	}
	var err error
	var caData []byte
	if ca != "" {
		caData, err = os.ReadFile(ca)
		if err != nil {
			return Fatal(err)
		}
	}
	var certData []byte
	if cert != "" {
		certData, err = os.ReadFile(cert)
		if err != nil {
			return Fatal(err)
		}
	}
	var keyData []byte
	if key != "" {
		keyData, err = os.ReadFile(key)
		if err != nil {
			return Fatal(err)
		}
	}
	request := message.FileGetRequest{
		Pathname:          dst,
		URL:               url,
		CA:                caData,
		Cert:              certData,
		Key:               keyData,
		AutoDeleteSeconds: seconds,
	}

	if c.debug {
		log.Printf("winexec get request: %+v\n", request)
	}
	var response message.FileGetResponse
	_, err = c.api.Post("/get/", &request, &response, nil)
	if err != nil {
		return Fatal(err)
	}
	if c.debug {
		log.Printf("winexec get response: %+v\n", response)
	}
	if !response.Success {
		return Fatalf("WinExec: GetISO failed: %v", response)
	}
	return nil
}

func (c *WinexecClient) DirFiles(pathname string) ([]string, error) {
	if c.debug {
		log.Printf("winexec DirFiles(%s)\n", pathname)
	}
	entries, err := c.DirEntries(pathname)
	if err != nil {
		return []string{}, Fatal(err)
	}
	files := []string{}
	for name, entry := range entries {
		if entry.Mode.IsRegular() {
			files = append(files, name)
		}
	}
	slices.Sort(files)
	return files, nil
}

func (c *WinexecClient) DirSubs(pathname string) ([]string, error) {
	if c.debug {
		log.Printf("winexec DirSubdirs(%s)\n", pathname)
	}
	entries, err := c.DirEntries(pathname)
	if err != nil {
		return []string{}, Fatal(err)
	}
	subs := []string{}
	for name, entry := range entries {
		if entry.Mode.IsDir() {
			subs = append(subs, name)
		}
	}
	slices.Sort(subs)
	return subs, nil
}

func (c *WinexecClient) DirEntries(pathname string) (map[string]message.DirectoryEntry, error) {
	if c.debug {
		log.Printf("winexec DirEntries(%s)\n", pathname)
	}
	entries := make(map[string]message.DirectoryEntry)
	request := message.DirectoryRequest{
		Pathname: pathname,
	}
	if c.debug {
		log.Printf("winexec directory request: %+v\n", request)
	}
	var response message.DirectoryResponse
	_, err := c.api.Post("/dir/", &request, &response, nil)
	if err != nil {
		return entries, Fatal(err)
	}
	if c.debug {
		log.Printf("winexec directory response: %+v\n", response)
	}
	if !response.Success {
		return entries, Fatalf("WinExec: directory operation failed: %v", response)
	}
	for name, entry := range response.Entries {
		entries[name] = entry
	}
	return entries, nil
}

func (c *WinexecClient) MkdirAll(pathname string, mode fs.FileMode) error {
	if c.debug {
		log.Printf("winexec MkdirAll(%s, %v)\n", pathname, mode)
	}
	request := message.DirectoryCreateRequest{
		Pathname: pathname,
		Mode:     mode,
	}
	if c.debug {
		log.Printf("winexec directory request: %+v\n", request)
	}
	var response message.DirectoryResponse
	_, err := c.api.Post("/mkdir/", &request, &response, nil)
	if err != nil {
		return Fatal(err)
	}
	if c.debug {
		log.Printf("winexec directory response: %+v\n", response)
	}
	if !response.Success {
		return Fatalf("WinExec: directory operation failed: %v", response)
	}
	return nil
}

func (c *WinexecClient) RemoveAll(pathname string) error {
	if c.debug {
		log.Printf("winexec RemoveAll(%s)\n", pathname)
	}
	request := message.DirectoryDestroyRequest{
		Pathname: pathname,
	}
	if c.debug {
		log.Printf("winexec directory request: %+v\n", request)
	}
	var response message.DirectoryResponse
	_, err := c.api.Post("/rmdir/", &request, &response, nil)
	if err != nil {
		return Fatal(err)
	}
	if c.debug {
		log.Printf("winexec directory response: %+v\n", response)
	}
	if !response.Success {
		return Fatalf("WinExec: directory destroy failed: %v", response)
	}
	return nil
}

func (c *WinexecClient) IsFile(pathname string) (bool, error) {
	if c.debug {
		log.Printf("winexec IsFile(%s)\n", pathname)
	}
	request := message.IsRequest{
		Pathname: pathname,
	}
	if c.debug {
		log.Printf("winexec isfile request: %+v\n", request)
	}
	var response message.IsResponse
	_, err := c.api.Post("/isfile/", &request, &response, nil)
	if err != nil {
		return false, Fatal(err)
	}
	if c.debug {
		log.Printf("winexec isfile response: %+v\n", response)
	}
	if !response.Success {
		return false, Fatalf("WinExec: isfile failed: %v", response)
	}
	return response.Result, nil
}

func (c *WinexecClient) IsDir(pathname string) (bool, error) {
	if c.debug {
		log.Printf("winexec IsDir(%s)\n", pathname)
	}
	request := message.IsRequest{
		Pathname: pathname,
	}
	if c.debug {
		log.Printf("winexec isdir request: %+v\n", request)
	}
	var response message.IsResponse
	_, err := c.api.Post("/isdir/", &request, &response, nil)
	if err != nil {
		return false, Fatal(err)
	}
	if c.debug {
		log.Printf("winexec isdir response: %+v\n", response)
	}
	if !response.Success {
		return false, Fatalf("WinExec: isdir failed: %v", response)
	}
	return response.Result, nil
}

func (c *WinexecClient) DeleteFile(pathname string) error {
	if c.debug {
		log.Printf("winexec DeleteFile(%s)\n", pathname)
	}
	request := message.FileDeleteRequest{
		Pathname: pathname,
	}
	if c.debug {
		log.Printf("winexec delete file request: %+v\n", request)
	}
	var response message.FileResponse
	_, err := c.api.Post("/delete/", &request, &response, nil)
	if err != nil {
		return Fatal(err)
	}
	if c.debug {
		log.Printf("winexec file response: %+v\n", response)
	}
	if !response.Success {
		return Fatalf("WinExec: DeleteFile failed: %v", response)
	}
	return nil
}

func (c *WinexecClient) GetOS() (string, error) {
	if c.debug {
		log.Println("winexec GetOS()")
	}
	var response message.GetOSResponse
	_, err := c.api.Get("/os/", &response)
	if err != nil {
		return "", Fatal(err)
	}
	if c.debug {
		log.Printf("winexec getos response: %+v\n", response)
	}
	if !response.Success {
		return "", Fatalf("WinExec: getos failed: %v", response)
	}
	return response.OS, nil
}
