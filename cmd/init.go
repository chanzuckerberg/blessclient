package cmd

import (
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	prompt "github.com/segmentio/go-prompt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.Flags().StringP("config", "c", "~/blessclient/config.yml", "Use this to override the bless config file.")
	rootCmd.AddCommand(loginCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize your bless config",
	Long:  "This command asks for input and generates your blessclient config",
	Run: func(cmd *cobra.Command, args []string) {
		err := initFunc(cmd, args)
		// TODO better error handling
		if err != nil {
			log.Fatal(err)
		}
	},
}

var initFunc = func(cmd *cobra.Command, args []string) error {
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return errs.ErrMissingConfig
	}

	prompt.StringRequired()

	return nil
}
