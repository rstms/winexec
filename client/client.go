package client

import (
	"github.com/rstms/winexec/message"
	"github.com/spf13/viper"
	"log"
	"os"
	"regexp"
	"strings"
)

const Version = "1.1.7"

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

func (c *WinexecClient) Upload(dst, src string) error {
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
		Pathname:  WindowsPath(dst),
		Content:   data,
		Timestamp: fileinfo.ModTime(),
		Mode:      fileinfo.Mode(),
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
		Pathname: WindowsPath(src),
	}
	var response message.FileDownloadResponse
	if c.debug {
		log.Printf("winexec download request: %+v\n", request)
	}
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
	var response message.ExecResponse
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
func WindowsPath(localPath string) string {
	if strings.Contains(localPath, `\`) {
		log.Println("has windows separators")
		localPath = strings.ReplaceAll(localPath, `\`, "/")
	}
	var drivePrefix string
	winPath := localPath
	switch {
	case regexp.MustCompile(`^/[a-zA-Z]/`).MatchString(winPath):
		log.Println("has drive coded as dir")
		drivePrefix = strings.ToUpper(string(winPath[1])) + ":"
		winPath = winPath[2:]
	case regexp.MustCompile(`^[a-zA-Z]:`).MatchString(winPath):
		log.Println("has drive letter")
		drivePrefix = strings.ToUpper(string(winPath[0])) + ":"
		winPath = winPath[2:]
	}
	winPath = strings.ReplaceAll(winPath, "/", `\`)
	ret := drivePrefix + winPath
	log.Printf("localPath=%s\n", localPath)
	log.Printf("drivePrefix=%s\n", drivePrefix)
	log.Printf("winPath=%s\n", winPath)
	log.Printf("ret=%s\n", ret)
	return ret
}
