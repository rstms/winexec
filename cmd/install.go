/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install winexec task",
	Long: `
Create a task scheduler task running winexec in the current user context.
Set the task to run interactively and start on login.
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := Task.Install()
		cobra.CheckErr(err)
	},
}

func init() {
	CobraAddCommand(rootCmd, taskCmd, installCmd)
}
