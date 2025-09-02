/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "restart winexec task",
	Long: `
Issue a task scheduler END followed by a RUN for the winexec task.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := Task.Stop()
		cobra.CheckErr(err)
		err = Task.Start()
		cobra.CheckErr(err)
	},
}

func init() {
	CobraAddCommand(rootCmd, taskCmd, restartCmd)
}
