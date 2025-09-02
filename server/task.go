/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/

package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

//go:embed task.xml
var xmlTemplate string

type Task struct {
	Name       string
	Username   string
	Uid        string
	Executable string
	Args       string
	Dir        string
	LogFile    string
}

func NewTask(taskName string) (*Task, error) {
	currentUser, err := user.Current()
	if err != nil {
		return nil, Fatal(err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, Fatal(err)
	}
	logDir := filepath.Join(homeDir, "logs")
	err = os.MkdirAll(logDir, 0700)
	if err != nil {
		return nil, Fatal(err)
	}
	logFile := filepath.Join(logDir, taskName+".log")
	args := "-l " + logFile + " server"
	t := Task{
		Name:       taskName,
		Username:   currentUser.Username,
		Uid:        currentUser.Uid,
		Executable: executable,
		Args:       args,
		Dir:        homeDir,
		LogFile:    logFile,
	}
	return &t, nil
}

func (t *Task) taskScheduler(cmd string, args ...string) (int, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	taskArgs := append([]string{"/" + cmd, "/TN", t.Name}, args...)
	command := exec.Command("schtasks.exe", taskArgs...)
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := command.ProcessState.ExitCode()
	estr := strings.TrimSpace(stderr.String())
	ostr := strings.TrimSpace(stdout.String())
	if ViperGetBool("verbose") {
		fmt.Printf("%s\n", ostr)
	}
	if err != nil {
		if estr != "" {
			return exitCode, "", fmt.Errorf("%v", estr)
		}
		return exitCode, "", err
	}
	return exitCode, ostr, nil
}

func (t *Task) Install() error {

	xmlData := os.Expand(xmlTemplate, func(key string) string {
		switch key {
		case "TASK_USER":
			return t.Username
		case "TASK_SID":
			return t.Uid
		case "TASK_BIN":
			return t.Executable
		case "TASK_ARGS":
			return t.Args
		case "TASK_DIR":
			return t.Dir
		}
		return "UNEXPANDED_XML_PARAM_" + key
	})

	tempDir, err := os.MkdirTemp("", "task-create-*")
	if err != nil {
		return Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	xmlFile := filepath.Join(tempDir, "task.xml")
	err = os.WriteFile(filepath.Join(tempDir, "task.xml"), []byte(xmlData), 0600)
	if err != nil {
		return Fatal(err)
	}
	createArgs := []string{
		"/XML", xmlFile,
	}
	_, _, err = t.taskScheduler("CREATE", createArgs...)
	if err != nil {
		return Fatal(err)
	}
	return nil
}

func (t *Task) Delete() error {
	_, _, err := t.taskScheduler("END")
	if err != nil {
		return Fatal(err)
	}
	_, _, err = t.taskScheduler("DELETE", "/F")
	if err != nil {
		return Fatal(err)
	}
	return nil
}

func (t *Task) Start() error {
	_, _, err := t.taskScheduler("RUN")
	if err != nil {
		return Fatal(err)
	}
	return nil
}

func (t *Task) Stop() error {
	_, _, err := t.taskScheduler("END")
	if err != nil {
		return Fatal(err)
	}
	return nil
}

func (t *Task) GetConfig() (string, error) {
	_, out, err := t.taskScheduler("QUERY", "/XML", "ONE")
	if err != nil {
		return "", Fatal(err)
	}
	return out, nil
}

func (t *Task) Query() (bool, error) {

	_, stdout, err := t.taskScheduler("QUERY", "/FO", "csv", "/NH")
	if err != nil {
		return false, Fatal(err)
	}
	fields := []string{}
	lines := strings.Split(stdout, "\n")
	if len(lines) == 1 {
		fields = strings.Split(lines[0], ",")
	}
	if len(lines) != 1 || len(fields) != 3 {
		return false, Fatalf("unexpected output: %v", stdout)
	}
	taskName := `"\` + t.Name + `"`
	if fields[0] != taskName {
		return false, Fatalf("unexpected task name: %s", fields[0])
	}
	if fields[2] == `"Running"` {
		return true, nil
	}
	return false, nil
}
