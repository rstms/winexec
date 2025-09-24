package server

import (
	"encoding/json"
	"fmt"
	"github.com/rstms/winexec/message"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func handleSpawn(w http.ResponseWriter, r *http.Request) {
	if Verbose {
		log.Printf("%s -> winexec %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)
	}
	var request message.SpawnRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		Warning("%v", Fatal(err))
		fail(w, r, "failed decoding request", http.StatusBadRequest)
		return
	}
	if Verbose {
		log.Printf("%+v\n", request)
	}

	spawned, exitCode, err := spawn(request.Env, request.Command, request.Args)
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

func spawn(env []string, command string, args []string) (string, int, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		commandWords := append([]string{command}, args...)
		commandLine := strings.Join(commandWords, " ")
		cmd = exec.Command("cmd", "/c", "start "+commandLine)
	default:
		cmd = exec.Command(command, args...)
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	var exitCode int
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	if Debug {
		log.Printf("Spawn: %v\n", cmd)
	}
	err := cmd.Start()
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
