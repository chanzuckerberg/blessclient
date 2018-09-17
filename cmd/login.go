package cmd

import (
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	loginCmd.Flags().StringP("config", "c", config.DefaultConfigFile, "Use this to override the bless config file.")
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:           "login",
	Short:         "Login to bless using your AWS credentials, will ask for MFA",
	Long:          "This command generates a set of temporary STS tokens that are cached locally on disk for 18 hours. MFA required.",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		isLogin := true
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return errs.ErrMissingConfig
		}
		expandedConfigFile, err := homedir.Expand(configFile)
		if err != nil {
			return errors.Wrapf(err, "Could not expand %s", configFile)
		}

		conf, err := config.FromFile(expandedConfigFile)
		if err != nil {
			return err
		}

		sess, err := session.NewSessionWithOptions(
			session.Options{
				SharedConfigState:       session.SharedConfigEnable,
				AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
			},
		)
		if err != nil {
			return err
		}

		client, err := bless.New(conf, sess, isLogin)
		if err != nil {
			return err
		}
		log.Debug("Requesting kmsauth token")
		_, err = client.RequestKMSAuthToken()
		return err
	},
}
