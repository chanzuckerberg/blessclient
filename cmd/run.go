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
		a, err := cziSSH.GetSSHAgent(os.Getenv("SSH_AUTH_SOCK"))
		if err != nil {
			return err
		}
		defer a.Close()
		manager := cziSSH.NewAgentKeyManager(a)

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
				// TODO(el): split these into separate prs
				// AWSRoleARN:    "",
				// OIDCClientID:  "",
				// OIDCIssuerURL: "",
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

		client := bless.NewOIDC(
			awsClient, &config.LambdaConfig{
				FunctionName: "TODO", // TODO
			})

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

		return manager.WriteKey(priv, cert)
	},
}
