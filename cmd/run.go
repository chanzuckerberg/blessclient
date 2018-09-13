package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run requests a certificate",
	Run: func(cmd *cobra.Command, args []string) {

		err := run(cmd, args)
		if err != nil {
			log.Fatal(err)
		}
	},
}
var run = func(cmd *cobra.Command, args []string) error {
	return nil
}
