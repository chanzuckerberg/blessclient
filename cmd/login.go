package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	loginCmd.Flags().StringP("config", "c", "~/blessclient/config.yml", "Use this to override the bless config file.")
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to bless using your AWS credentials, will ask for MFA",
	Long:  "This command generates a set of temporary STS tokens that are cached locally on disk for 18 hours. MFA required.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
