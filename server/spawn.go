package server

import (
	"encoding/json"
	"fmt"
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func handleSpawn(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.ExecRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "failed decoding request", http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	command := append([]string{request.Command}, request.Args...)
	spawned, exitCode, err := spawn(strings.Join(command, " "))
	if err != nil {
		Warning("spawn: %v", Fatal(err))
		fail(w, r, "spawn failed", http.StatusBadRequest)
		return
	}
	response := message.SpawnResponse{
		Success:  true,
		Message:  "spawned",
		Command:  spawned,
		ExitCode: exitCode,
	}
	succeed(w, r, &response)
}

func spawn(command string) (string, int, error) {
	cmd := exec.Command("cmd", "/c", "start "+command)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	var exitCode int
	if Debug {
		log.Printf("Spawn: %v\n", cmd)
	}
	err := cmd.Run()
	if err != nil {
		switch e := err.(type) {
		case *exec.ExitError:
			exitCode = e.ExitCode()
			err = nil
		}
	}
	if Debug {
		log.Printf("exitCode=%d\n", exitCode)
		log.Printf("err=%v\n", err)
	}
	return fmt.Sprintf("%v", cmd), exitCode, err
}
