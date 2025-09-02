/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete task scheduler task",
	Long: `
Issue a task scheduler DELETE command for the 'winexec' task.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := Task.Delete()
		cobra.CheckErr(err)
	},
}

func init() {
	CobraAddCommand(rootCmd, taskCmd, deleteCmd)
}
