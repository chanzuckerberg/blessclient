package cmd

import (
	"os"
	"path"

	"github.com/mitchellh/go-homedir"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	cziAWS "github.com/chanzuckerberg/blessclient/pkg/aws"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	loginCmd.Flags().StringP("config", "c", config.DefaultConfigFile, "Use this to override the bless config file.")
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to bless using your AWS credentials, will ask for MFA",
	Long:  "This command generates a set of temporary STS tokens that are cached locally on disk for 18 hours. MFA required.",
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

		mfaCache := conf.ClientConfig.MFACacheFile
		err = os.MkdirAll(path.Dir(mfaCache), 0755)
		if err != nil {
			return errors.Wrapf(err, "Could not create mfa cache dir at %s", path.Dir(mfaCache))
		}

		userTokenProvider := cziAWS.NewUserTokenProvider(sess, mfaCache, isLogin)
		provider := credentials.NewCredentials(userTokenProvider)
		mfaAWSConfig := &aws.Config{
			Credentials: provider,
		}

		kmsAuthAWSClient, err := bless.New(conf, sess, mfaAWSConfig)
		if err != nil {
			return err
		}
		_, err = kmsAuthAWSClient.RequestKMSAuthToken()
		return err
	},
}
