/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "start the winexec task",
	Long: `
Issue a task scheduler RUN command for the winexec task.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := Task.Start()
		cobra.CheckErr(err)
	},
}

func init() {
	CobraAddCommand(rootCmd, taskCmd, startCmd)
}
