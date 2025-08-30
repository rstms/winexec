package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"os/exec"
)

func handleExec(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.ExecRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	command, exit, stdout, stderr, err := run(request.Command, request.Args...)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "exec failed", http.StatusBadRequest)
		return
	}
	response := message.ExecResponse{
		Success:  true,
		Message:  "executed",
		Command:  command,
		ExitCode: exit,
		Stdout:   stdout,
		Stderr:   stderr,
	}
	succeed(w, r, &response)
}

func run(command string, args ...string) (string, int, string, string, error) {
	if Debug {
		log.Printf("Run: %s %v\n", command, args)
	}
	cmd := exec.Command(command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	exitCode := 0
	err := cmd.Run()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			exitCode = e.ExitCode()
			err = nil
		default:
			return "", -1, "", "", err
		}
	}
	if Debug {
		log.Printf("exitCode=%d\n", exitCode)
		log.Printf("stdout=%s\n", stdout.String())
		log.Printf("stderr=%s\n", stderr.String())
		log.Printf("err=%v\n", err)
	}
	return fmt.Sprintf("%v", cmd), exitCode, stdout.String(), stderr.String(), err
}
