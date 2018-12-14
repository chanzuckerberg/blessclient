package cmd

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/telemetry"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	kmsauth "github.com/chanzuckerberg/go-misc/kmsauth"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/honeycombio/opencensus-exporter/honeycomb"
	"github.com/pkg/errors"
	awsokta "github.com/segmentio/aws-okta/lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opencensus.io/trace"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "run requests a certificate",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := uuid.NewUUID()
		if err != nil {
			// Just for telemetry so ignore errors
			log.Debugf("Failed to generate UUID with error %s", err.Error())
		}
		log.Debugf("Running blessclient v%s", util.VersionCacheKey())
		log.Debugf("RunID: %s", id.String())
		ctx := context.Background()
		expandedConfigFile, err := util.GetConfigPath(cmd)
		if err != nil {
			return err
		}
		log.Debugf("Reading config from %s", expandedConfigFile)

		conf, err := config.FromFile(expandedConfigFile)
		if err != nil {
			return err
		}
		log.Debugf("Parsed config is: %s", spew.Sdump(conf))

		// tracing
		traceSampling := float64(1)
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(traceSampling)})
		if conf.Telemetry.Honeycomb != nil {
			honeycombExporter := honeycomb.NewExporter(conf.Telemetry.Honeycomb.WriteKey, conf.Telemetry.Honeycomb.Dataset)
			defer honeycombExporter.Close()
			honeycombExporter.ServiceName = "blessclient"
			honeycombExporter.SampleFraction = traceSampling
			trace.RegisterExporter(honeycombExporter)
		}

		ctx, span := trace.StartSpan(ctx, cmd.Use)
		span.AddAttributes(
			trace.StringAttribute(telemetry.FieldID, id.String()),
			trace.StringAttribute(telemetry.FieldBlessclientVersion, util.VersionCacheKey()),
			trace.StringAttribute(telemetry.FieldBlessclientGitSha, util.GitSha),
			trace.StringAttribute(telemetry.FieldBlessclientRelease, util.Release),
			trace.StringAttribute(telemetry.FieldBlessclientDirty, util.Dirty),
		)
		defer span.End()

		sess, err := session.NewSessionWithOptions(
			session.Options{
				SharedConfigState:       session.SharedConfigEnable,
				AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
				Profile:                 conf.ClientConfig.AWSUserProfile,
			},
		)
		if err != nil {
			span.AddAttributes(trace.StringAttribute(telemetry.FieldError, err.Error()))
			return errors.Wrap(err, "Could not create aws session")
		}

		var regionErrors error
		for _, region := range conf.LambdaConfig.Regions {
			log.Debugf("Attempting region %s", region.AWSRegion)
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
	ctx, span := trace.StartSpan(ctx, "process_region")
	defer span.End()
	span.AddAttributes(trace.StringAttribute(telemetry.FieldRegion, region.AWSRegion))

	awsClient := getAWSClient(ctx, conf, sess, region)
	username, err := conf.GetAWSUsername(ctx, awsClient)
	if err != nil {
		span.AddAttributes(trace.StringAttribute(telemetry.FieldError, err.Error()))
		return err
	}

	span.AddAttributes(trace.StringAttribute(telemetry.FieldUser, username))
	return getCert(ctx, conf, awsClient, username, region)
}

// getAWSClient configures an aws client
func getAWSClient(ctx context.Context, conf *config.Config, sess *session.Session, region config.Region) *cziAWS.Client {
	ctx, span := trace.StartSpan(ctx, "get_aws_client")
	defer span.End()
	// for things meant to be run as a user
	userConf := &aws.Config{
		Region: aws.String(region.AWSRegion),
	}
	if conf.OktaConfig != nil {
		// override user credentials with Okta credentials
		log.Debugf("Getting Okta AWS SSO credentials")
		creds, err := getAWSOktaCredentials(conf)
		if err != nil {
			log.Errorf("Error in retrieving AWS Okta session credentials: %s.", err.Error())
			return nil
		}

		userConf = &aws.Config{
			Region: aws.String(region.AWSRegion),
			Credentials: credentials.NewStaticCredentials(
				creds.AccessKeyID,
				creds.SecretAccessKey,
				creds.SessionToken,
			),
		}
	}

	lambdaConf := userConf
	if conf.LambdaConfig.RoleARN != nil {
		// for things meant to be run as an assumed role
		lambdaConf = &aws.Config{
			Region: aws.String(region.AWSRegion),
			Credentials: stscreds.NewCredentials(
				sess,
				*conf.LambdaConfig.RoleARN, func(p *stscreds.AssumeRoleProvider) {
					p.TokenProvider = stscreds.StdinTokenProvider
				},
			),
		}
	}
	awsClient := cziAWS.New(sess).
		WithIAM(userConf).
		WithKMS(userConf).
		WithSTS(userConf).
		WithLambda(lambdaConf)
	return awsClient
}

func getAWSOktaCredentials(conf *config.Config) (*credentials.Value, error) {

	awsOktaConfig, err := awsokta.NewConfigFromEnv()
	if err != nil {
		return nil, errors.Wrap(err, "Error getting aws-okta config")
	}

	profiles, err := awsOktaConfig.Parse()
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing aws-okta config")
	}

	profile := conf.OktaConfig.Profile
	if _, ok := profiles[profile]; !ok {
		return nil, errors.Errorf("Profile '%s' not found in your aws config", profile)
	}

	mfaDevice := conf.GetOktaMFADevice()
	opts := awsokta.ProviderOptions{
		MFADevice:          mfaDevice,
		Profiles:           profiles,
		SessionDuration:    time.Hour,
		AssumeRoleDuration: time.Hour,
	}

	kr, err := awsokta.OpenKeyring(nil)
	if err != nil {
		return nil, errors.Wrap(err, "Error opening keyring for credential storage")
	}

	p, err := awsokta.NewProvider(kr, profile, opts)
	if err != nil {
		return nil, errors.Wrap(err, "Error initializing aws-okta provider")
	}

	creds, err := p.Retrieve()

	return &creds, errors.Wrap(err, "Error retrieving STS credentials using aws-okta")
}

// getCert requests a cert and persists it to disk
func getCert(ctx context.Context, conf *config.Config, awsClient *cziAWS.Client, username string, region config.Region) error {
	ctx, span := trace.StartSpan(ctx, "get_cert")
	defer span.End()
	kmsauthContext := &kmsauth.AuthContextV2{
		From:     username,
		To:       conf.LambdaConfig.FunctionName,
		UserType: "user",
	}
	kmsAuthCachePath, err := conf.GetKMSAuthCachePath(region.AWSRegion)
	if err != nil {
		span.AddAttributes(trace.StringAttribute(telemetry.FieldError, err.Error()))
		return err
	}

	tg := kmsauth.NewTokenGenerator(
		region.KMSAuthKeyID,
		kmsauth.TokenVersion2,
		conf.ClientConfig.CertLifetime.AsDuration(),
		aws.String(kmsAuthCachePath),
		kmsauthContext,
		awsClient,
	)
	client := bless.New(conf).WithAwsClient(awsClient).WithTokenGenerator(tg).WithUsername(username)
	err = client.RequestCert(ctx)
	if err != nil {
		span.AddAttributes(trace.StringAttribute(telemetry.FieldError, err.Error()))
		return err
	}
	return nil
}
