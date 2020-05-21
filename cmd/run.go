package cmd

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziSSH "github.com/chanzuckerberg/blessclient/pkg/ssh"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	oidc "github.com/chanzuckerberg/go-misc/oidc_cli"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	flagForce = "force"
)

func init() {
	runCmd.Flags().BoolP(flagForce, "f", false, "Force certificate refresh")

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

		awsConf := aws.NewConfig().WithCredentials(credsProvider.Credentials).WithRegion("us-west-2")
		awsClient := cziAWS.New(sess).WithLambda(awsConf)

		client := bless.NewOIDC(awsClient, &config.LambdaConfig)

		cert, err := client.RequestCert(
			cmd.Context(),
			awsClient,
			&bless.SigningRequest{
				PublicKeyToSign: bless.NewPublicKeyToSign(pub),
				Identity: bless.Identity{
					OktaAccessToken: &bless.OktaAccessTokenInput{
						AccessToken: token.AccessToken,
					},
				},
			},
		)
		if err != nil {
			return err
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
