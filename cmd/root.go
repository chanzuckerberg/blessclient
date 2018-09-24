package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	debug bool
	quiet bool
)

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Use this to enable verbose mode")
}

var rootCmd = &cobra.Command{
	Use:   "blessclient",
	Short: "",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return fmt.Errorf("Missing verbose flag")
		}
		if verbose {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	},
}

// Execute executes the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
