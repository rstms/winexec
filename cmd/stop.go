/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop winexec task",
	Long: `
Issue a task scheduler END command.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := Task.Stop()
		cobra.CheckErr(err)
	},
}

func init() {
	CobraAddCommand(rootCmd, taskCmd, stopCmd)
}
