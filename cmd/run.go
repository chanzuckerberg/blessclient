package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run requests a certificate",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
