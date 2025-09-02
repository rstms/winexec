/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"os"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "query winexec task",
	Long: `
Issue a task scheduler QUERY for the winexec task.
Exit 0 if task is running, otherwise exit nonzero.
`,
	Run: func(cmd *cobra.Command, args []string) {
		running, err := Task.Query()
		if !ViperGetBool("quiet") {
			cobra.CheckErr(err)
		}
		if running {
			os.Exit(0)
		}
		os.Exit(1)
	},
}

func init() {
	CobraAddCommand(rootCmd, taskCmd, queryCmd)
}
