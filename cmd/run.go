package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/blessclient/pkg/telemetry"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	kmsauth "github.com/chanzuckerberg/go-kmsauth"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	multierror "github.com/hashicorp/go-multierror"
	beeline "github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/trace"
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
		log.Debugf("Running blessclient v%s", util.VersionCacheKey())
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

		// TODO find out what happens if config is bad
		// TODO turn this off as needed
		ctx := context.Background()
		beelineConfig := beeline.Config{
			WriteKey:    conf.Telemetry.Honeycomb.WriteKey,
			Dataset:     conf.Telemetry.Honeycomb.Dataset,
			ServiceName: "blessclient",
			// STDOUT:      true, // TODO rm once done developing
		}
		beeline.Init(beelineConfig)
		defer beeline.Flush(ctx)
		beeline.AddField(ctx, telemetry.FieldBlessclientVersion, util.VersionCacheKey())

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
		ctx, tr := trace.NewTrace(ctx, "")
		span := tr.GetRootSpan()
		span.AddField(telemetry.FieldCommand, cmd.Use)

		mfaTokenProvider := util.TokenProvider("AWS MFA token:")
		var regionErrors error
		for _, region := range conf.LambdaConfig.Regions {
			ctx, span = span.CreateChild(ctx)

			// for things meant to be run as a user
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

			ctx, currentUserSpan := span.CreateChild(ctx)
			defer currentUserSpan.Send()

			user, err := awsClient.IAM.GetCurrentUser(ctx)
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
			ctx, span = beeline.StartSpan(ctx, telemetry.FieldRequestCert)
			defer span.Send()

			err = client.RequestCert(ctx)
			if err != nil {
				log.Errorf("Error in region %s: %s. Attempting next region if one is available.", region.AWSRegion, err.Error())
				regionErrors = multierror.Append(regionErrors, err)
			} else {
				return nil
			}
		}
		return regionErrors
	},
}

// runForRegion runs for a region
func runForRegion(
	ctx context.Context,
	sess *session.Session,
	mfaTokenProvider func() (string, error),
	region string,
) error {

	span := trace.GetSpanFromContext(ctx)
	if span == nil {

	}
	return nil
}
