package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	debug bool
	quiet bool
)

func init() {
}

var rootCmd = &cobra.Command{
	Use:   "blessclient",
	Short: "",
}

// Execute executes the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
