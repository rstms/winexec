/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"github.com/rstms/winexec/server"
	"github.com/spf13/cobra"
	"os/exec"
	"strings"
)

var Task *server.Task

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "task subcommands",
	Long: `
task subcommands
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error
		Task, err = server.NewTask(ViperGetString("task.task_name"))
		cobra.CheckErr(err)
	},
}

func init() {
	CobraAddCommand(rootCmd, rootCmd, taskCmd)
	OptionString(taskCmd, "task-name", "", "winexec", "task scheduler task name")
}

func TaskScheduler(cmd string, args ...string) (int, string, error) {
	taskName := ViperGetString("task.task_name")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	taskArgs := append([]string{"/" + cmd, "/TN", taskName}, args...)
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
