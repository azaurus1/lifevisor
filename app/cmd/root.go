package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lifevisor",
	Short: "lifevisor is a cli tool for syncing local acitivitywatch data to a remote postgres db",
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Oops. An error while executing lifevisor '%s'\n", err)
		os.Exit(1)
	}
}
