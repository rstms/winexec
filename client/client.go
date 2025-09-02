package client

import (
	"github.com/rstms/winexec/message"
	"github.com/spf13/viper"
	"io/fs"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"
)

const Version = "1.1.15"

type WinexecClient struct {
	api   APIClient
	debug bool
}

func viperPrefix() string {
	prefix := "winexec.client."
	if ProgramName() == "winexec" {
		prefix = "client."
	}
	return prefix
}

func NewWinexecClient() (*WinexecClient, error) {

	prefix := viperPrefix()
	url := ViperGetString(prefix + "url")
	cert := ViperGetString(prefix + "cert")
	key := ViperGetString(prefix + "key")
	ca := ViperGetString(prefix + "ca")
	debug := ViperGetBool(prefix + "debug")

	if ViperGetBool("verbose") {
		log.Printf("NewWinexecClient: %s\n", FormatJSON(&map[string]any{
			"url":   url,
			"cert":  cert,
			"key":   key,
			"ca":    ca,
			"debug": debug,
		}))
	}

	api, err := NewAPIClient(prefix, url, cert, key, ca, nil)
	if err != nil {
		return nil, Fatal(err)
	}
	client := WinexecClient{
		api:   api,
		debug: debug,
	}

	return &client, nil

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

func (c *WinexecClient) Spawn(command string, exitCode *int) error {
	if c.debug {
		log.Printf("winexec Spawn(%s)\n", command)
	}
	request := message.ExecRequest{Command: command}
	var response message.ExecResponse
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
		return Fatalf("Spawned Process '%s' exited %d", command, response.ExitCode)
	}
	return nil
}

func (c *WinexecClient) Exec(command string, args []string, exitCode *int) (string, string, error) {
	if c.debug {
		log.Printf("winexec Exec(%s %v)\n", command, args)
	}
	request := message.ExecRequest{Command: command, Args: args}
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
		Pathname:  c.WindowsPath(dst),
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
		Pathname: c.WindowsPath(src),
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

func (c *WinexecClient) GetISO(dst, url, ca, cert, key string) error {
	if c.debug {
		log.Printf("winexec GetISO(%s %s %s %s %s)\n", dst, url, ca, cert, key)
	}
	caData, err := os.ReadFile(ca)
	if err != nil {
		return Fatal(err)
	}
	certData, err := os.ReadFile(cert)
	if err != nil {
		return Fatal(err)
	}
	keyData, err := os.ReadFile(key)
	if err != nil {
		return Fatal(err)
	}
	request := message.FileGetRequest{
		Pathname: dst,
		URL:      url,
		CA:       caData,
		Cert:     certData,
		Key:      keyData,
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

// convert a local path to a windows path
func (c *WinexecClient) WindowsPath(localPath string) string {
	if strings.Contains(localPath, `\`) {
		log.Println("has windows separators")
		localPath = strings.ReplaceAll(localPath, `\`, "/")
	}
	var drivePrefix string
	winPath := localPath
	switch {
	case regexp.MustCompile(`^/[a-zA-Z]/`).MatchString(winPath):
		//log.Println("has drive letter coded as dir")
		drivePrefix = strings.ToUpper(string(winPath[1])) + ":"
		winPath = winPath[2:]
	case regexp.MustCompile(`^[a-zA-Z]:`).MatchString(winPath):
		//log.Println("has drive letter colon prefix")
		drivePrefix = strings.ToUpper(string(winPath[0])) + ":"
		winPath = winPath[2:]
	case regexp.MustCompile(`^//[^/]+/[^/]+/[^/]+`).MatchString(winPath):
		//log.Printf("has UNC drive prefix: %s\n", winPath)
		elements := regexp.MustCompile(`^//([^/]+)/([a-zA-Z][$:]{0,1})(/.*)$`).FindStringSubmatch(winPath)
		if len(elements) == 4 {
			for i, element := range elements {
				log.Printf("%d: %s\n", i, element)
			}
			drivePrefix = strings.ToUpper(string(elements[2][0])) + ":"
			winPath = elements[3]
		}
	}
	winPath = strings.ReplaceAll(winPath, "/", `\`)
	ret := drivePrefix + winPath
	//log.Printf("localPath=%s\n", localPath)
	//log.Printf("drivePrefix=%s\n", drivePrefix)
	//log.Printf("winPath=%s\n", winPath)
	//log.Printf("ret=%s\n", ret)
	return ret
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
	request := message.DirectoryRequest{
		Pathname: c.WindowsPath(pathname),
	}
	if c.debug {
		log.Printf("winexec directory request: %+v\n", request)
	}
	entries := make(map[string]message.DirectoryEntry)
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
		Pathname: c.WindowsPath(pathname),
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
		Pathname: c.WindowsPath(pathname),
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
