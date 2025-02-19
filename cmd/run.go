package cmd

import (
	"context"
	"crypto"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziSSH "github.com/chanzuckerberg/blessclient/pkg/ssh"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	oidc "github.com/chanzuckerberg/go-misc/oidc_cli"
	"github.com/chanzuckerberg/go-misc/oidc_cli/oidc_impl/client"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

const (
	flagForce     = "force"
	flagPrintCert = "print-cert"
)

func init() {
	runCmd.Flags().BoolP(flagForce, "f", false, "Force certificate refresh")
	runCmd.Flags().Bool(flagPrintCert, false, "Prints the SSH Certificate for debugging purposes")

	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:           "run",
	Short:         "run requests a certificate",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, err := cmd.Flags().GetBool(flagForce)
		if err != nil {
			return errors.Wrap(err, "Missing force flag")
		}
		printCert, err := cmd.Flags().GetBool(flagPrintCert)
		if err != nil {
			return errors.Wrap(err, "Missing print-cert flag")
		}

		config, err := config.FromFile(config.DefaultConfigFile)
		if err != nil {
			return err
		}

		a, err := cziSSH.GetSSHAgent(os.Getenv("SSH_AUTH_SOCK"))
		if err != nil {
			return err
		}
		defer a.Close()
		manager := cziSSH.NewAgentKeyManager(a)

		hasCert, err := manager.HasValidCertificate()
		if err != nil {
			return err
		}
		if !force && hasCert {
			logrus.Debug("fresh cert, nothing to do")
			return nil
		}

		pub, priv, err := manager.GetKey()
		if err != nil {
			return err
		}

		sess, err := session.NewSession()
		if err != nil {
			return errors.Wrap(err, "could not initialize AWS session")
		}

		stsSvc := sts.New(sess)

		credsProvider, err := oidc.NewAwsOIDCCredsProvider(
			cmd.Context(),
			stsSvc,
			&oidc.AwsOIDCCredsProviderConfig{
				AWSRoleARN:    config.ClientConfig.RoleARN,
				OIDCClientID:  config.ClientConfig.OIDCClientID,
				OIDCIssuerURL: config.ClientConfig.OIDCIssuerURL,
			},
		)
		if err != nil {
			return err
		}

		token, err := credsProvider.FetchOIDCToken(cmd.Context())
		if err != nil {
			return err
		}

		cert, err := regionalGetCert(
			cmd.Context(),
			sess,
			credsProvider.Credentials,
			config,
			token,
			pub,
		)
		if err != nil {
			return err
		}

		if printCert {
			err = cziSSH.PrintCertificate(cert, os.Stderr)
			if err != nil {
				logrus.WithError(err).Debug("Could not print cert. Ignoring error.")
			}
		}

		err = manager.WriteKey(priv, cert)
		if err != nil {
			return err
		}

		hasCert, err = manager.HasValidCertificate()
		if err != nil {
			return err
		}

		if !hasCert {
			return errors.Errorf("wrote error to key manager, but could not fetch it back")
		}
		return nil
	},
}

func regionalGetCert(
	ctx context.Context,
	sess *session.Session,
	creds *credentials.Credentials,
	blessConfig *config.Config,
	token *client.Token,
	publicKey crypto.PublicKey,
) (*ssh.Certificate, error) {
	var errors *multierror.Error

	for _, region := range blessConfig.LambdaConfig.Regions {
		logrus.Debugf("Attempting to get cert from region %s", region.AWSRegion)

		awsConf := aws.NewConfig().WithCredentials(creds).WithRegion(region.AWSRegion)
		awsClient := cziAWS.New(sess).WithLambda(awsConf)
		client := bless.NewOIDC(awsClient, &blessConfig.LambdaConfig)
		cert, err := client.RequestCert(
			ctx,
			awsClient,
			&bless.SigningRequest{
				PublicKeyToSign: bless.NewPublicKeyToSign(publicKey),
				Identity: bless.Identity{
					OktaAccessToken: &bless.OktaAccessTokenInput{
						AccessToken: token.AccessToken,
					},
				},
			},
		)
		// if no error, done and return
		if err == nil {
			return cert, nil
		}
		// if error, accumulate it
		errors = multierror.Append(errors, err)
	}
	return nil, errors.ErrorOrNil()
}
