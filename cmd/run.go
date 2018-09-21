package cmd

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	kmsauth "github.com/chanzuckerberg/go-kmsauth"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	multierror "github.com/hashicorp/go-multierror"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	runCmd.Flags().StringP("config", "c", config.DefaultConfigFile, "Use this to override the bless config file.")
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "run requests a certificate",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Infof("Running blessclient %s", util.VersionCacheKey())
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
				Profile:                 conf.ClientConfig.AWSUserProfile,
			},
		)
		if err != nil {
			return errors.Wrap(err, "Could not create aws session")
		}

		mfaTokenProvider := util.TokenProvider("AWS MFA token:")
		var regionErrors error
		for _, region := range conf.LambdaConfig.Regions {
			awsUserSessionProviderConf := &aws.Config{
				Region: aws.String(region.AWSRegion),
			}
			awsSessionProviderClient := cziAWS.New(sess).WithAllServices(awsUserSessionProviderConf)

			awsSessionTokenProvider := cziAWS.NewUserTokenProvider(conf.GetAWSSessionCachePath(), awsSessionProviderClient, mfaTokenProvider)
			userConf := &aws.Config{
				Region:      aws.String(region.AWSRegion),
				Credentials: credentials.NewCredentials(awsSessionTokenProvider),
			}
			// for things meant to be run as an assumed role
			roleConf := &aws.Config{
				Region: aws.String(region.AWSRegion),
				Credentials: stscreds.NewCredentials(
					sess,
					conf.LambdaConfig.RoleARN, func(p *stscreds.AssumeRoleProvider) {
						p.TokenProvider = stscreds.StdinTokenProvider
					},
				),
			}

			awsClient := cziAWS.New(sess).
				WithIAM(userConf).
				WithKMS(userConf).
				WithSTS(userConf).
				WithLambda(roleConf)

			user, err := awsClient.IAM.GetCurrentUser()
			if err != nil {
				return err
			}
			if user == nil || user.UserName == nil {
				return errors.New("AWS returned nil user")
			}

			kmsauthContext := &kmsauth.AuthContextV2{
				From:     *user.UserName,
				To:       conf.LambdaConfig.FunctionName,
				UserType: "user",
			}

			tg := kmsauth.NewTokenGenerator(
				region.KMSAuthKeyID,
				kmsauth.TokenVersion2,
				conf.ClientConfig.CertLifetime.AsDuration(),
				aws.String(conf.GetKMSAuthCachePath(region.AWSRegion)),
				kmsauthContext,
				awsClient,
			)

			client := bless.New(conf).WithAwsClient(awsClient).WithTokenGenerator(tg).WithUsername(*user.UserName)
			err = client.RequestCert()
			if err != nil {
				log.Errorf("Error in region %s: %s. Attempting other regions is available.", region.AWSRegion, err.Error())
				regionErrors = multierror.Append(regionErrors, err)
			} else {
				return nil
			}
		}
		return regionErrors
	},
}
