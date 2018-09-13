package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "login to bless using your AWS credentials",
	Run: func(cmd *cobra.Command, args []string) {

		err := login(cmd, args)
		if err != nil {
			log.Fatal(err)
		}
	},
}

var login = func(cmd *cobra.Command, args []string) error {
	return nil
}
