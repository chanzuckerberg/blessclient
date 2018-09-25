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
	"github.com/davecgh/go-spew/spew"
	multierror "github.com/hashicorp/go-multierror"
	beeline "github.com/honeycombio/beeline-go"
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
		ctx := context.Background()
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return errs.ErrMissingConfig
		}
		expandedConfigFile, err := homedir.Expand(configFile)
		if err != nil {
			return errors.Wrapf(err, "Could not expand %s", configFile)
		}
		log.Debugf("Reading config from %s", expandedConfigFile)

		conf, err := config.FromFile(expandedConfigFile)
		if err != nil {
			return err
		}
		log.Debugf("Parsed config is: %s", spew.Sdump(conf))

		// TODO find out what happens if config is bad
		// TODO turn this off as needed
		beelineConfig := beeline.Config{
			WriteKey:    conf.Telemetry.Honeycomb.WriteKey,
			Dataset:     conf.Telemetry.Honeycomb.Dataset,
			ServiceName: "blessclient",
		}
		beeline.Init(beelineConfig)
		defer beeline.Flush(ctx)

		ctx, span := beeline.StartSpan(ctx, cmd.Use)
		span.AddTraceField(telemetry.FieldBlessclientVersion, util.VersionCacheKey())
		span.AddTraceField(telemetry.FieldBlessclientGitSha, util.GitSha)
		span.AddTraceField(telemetry.FieldBlessclientRelease, util.Release)
		span.AddTraceField(telemetry.FieldBlessclientDirty, util.Dirty)

		defer span.Send()

		sess, err := session.NewSessionWithOptions(
			session.Options{
				SharedConfigState:       session.SharedConfigEnable,
				AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
				Profile:                 conf.ClientConfig.AWSUserProfile,
			},
		)
		if err != nil {
			span.AddField(telemetry.FieldError, err.Error())
			return errors.Wrap(err, "Could not create aws session")
		}

		var regionErrors error
		for _, region := range conf.LambdaConfig.Regions {
			log.Debugf("Attempting region %s", region)
			err = processRegion(ctx, conf, sess, region)
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

func processRegion(ctx context.Context, conf *config.Config, sess *session.Session, region config.Region) error {
	ctx, span := beeline.StartSpan(ctx, "process_region")
	defer span.Send()
	span.AddField(telemetry.FieldRegion, region.AWSRegion)

	awsClient := getAWSClient(ctx, conf, sess, region)
	username, err := getAWSUsername(ctx, awsClient)
	if err != nil {
		span.AddField(telemetry.FieldError, err.Error())
		return err
	}
	return getCert(ctx, conf, awsClient, username, region)

}

// getAWSClient configures an aws client
func getAWSClient(ctx context.Context, conf *config.Config, sess *session.Session, region config.Region) *cziAWS.Client {
	ctx, span := beeline.StartSpan(ctx, "get_aws_client")
	defer span.Send()
	// for things meant to be run as a user
	mfaTokenProvider := util.TokenProvider("AWS MFA token:")
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
	return awsClient
}

// getAWSUsername gets the caller's aws username for kmsauth
func getAWSUsername(ctx context.Context, awsClient *cziAWS.Client) (string, error) {
	ctx, span := beeline.StartSpan(ctx, "get_aws_username")
	defer span.Send()
	log.Debugf("Getting current aws iam user")
	user, err := awsClient.IAM.GetCurrentUser(ctx)
	if err != nil {
		span.AddField(telemetry.FieldError, err.Error())
		return "", err
	}
	if user == nil || user.UserName == nil {
		err = errors.New("AWS returned nil user")
		span.AddField(telemetry.FieldError, err.Error())
		return "", err
	}
	beeline.AddFieldToTrace(ctx, telemetry.FieldUser, *user.UserName)
	return *user.UserName, nil
}

// getCert requests a cert and persists it to disk
func getCert(ctx context.Context, conf *config.Config, awsClient *cziAWS.Client, username string, region config.Region) error {
	ctx, span := beeline.StartSpan(ctx, "get_cert")
	defer span.Send()
	kmsauthContext := &kmsauth.AuthContextV2{
		From:     username,
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
	client := bless.New(conf).WithAwsClient(awsClient).WithTokenGenerator(tg).WithUsername(username)
	err := client.RequestCert(ctx)
	if err != nil {
		span.AddField(telemetry.FieldError, err.Error())
		return err
	}
	return nil
}
