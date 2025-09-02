/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "show task scheduler config",
	Long: `
Output the task configuration XML for the winexec task.
`,
	Run: func(cmd *cobra.Command, args []string) {
		out, err := Task.GetConfig()
		cobra.CheckErr(err)
		fmt.Println(out)
	},
}

func init() {
	CobraAddCommand(rootCmd, taskCmd, showCmd)
}
