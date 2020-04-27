package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/ssh"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	kmsauth "github.com/chanzuckerberg/go-misc/kmsauth"
	"github.com/davecgh/go-spew/spew"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "run requests a certificate",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Debugf("Running blessclient v%s", util.VersionCacheKey())
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

		ssh, err := ssh.NewSSH(conf.ClientConfig.SSHPrivateKey)
		if err != nil {
			return err
		}

		isFresh, err := ssh.IsCertFresh(conf)
		if err != nil {
			return err
		}
		if isFresh {
			log.Debug("Cert is already fresh - using it")
			return nil
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
	awsClient, err := getAWSClient(ctx, conf, sess, region)
	if err != nil {
		return err
	}
	username, err := conf.GetAWSUsername(ctx, awsClient)
	if err != nil {
		return err
	}

	return getCert(ctx, conf, awsClient, username, region)
}

// getAWSClient configures an aws client
func getAWSClient(ctx context.Context, conf *config.Config, sess *session.Session, region config.Region) (*cziAWS.Client, error) {
	// for things meant to be run as a user
	userConf := &aws.Config{
		Region: aws.String(region.AWSRegion),
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
	return awsClient, nil
}

// getCert requests a cert and persists it to disk
func getCert(ctx context.Context, conf *config.Config, awsClient *cziAWS.Client, username string, region config.Region) error {
	kmsauthContext := &kmsauth.AuthContextV2{
		From:     username,
		To:       conf.LambdaConfig.FunctionName,
		UserType: "user",
	}
	kmsAuthCachePath, err := conf.GetKMSAuthCachePath(region.AWSRegion)
	if err != nil {
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
	client := bless.NewKMSAuthClient(conf).WithAwsClient(awsClient).WithTokenGenerator(tg).WithUsername(username)
	err = client.RequestCert(ctx)
	if err != nil {
		return err
	}
	return nil
}
