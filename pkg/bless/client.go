package client

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	cziAWS "github.com/chanzuckerberg/blessclient/pkg/aws"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/go-kmsauth"
	"github.com/pkg/errors"
)

// Client is a bless client
type Client struct {
	Aws            *cziAWS.Client
	tokenGenerator *kmsauth.TokenGenerator
	conf           *config.Config
}

// New returns a new client
func New(conf *config.Config, sess *session.Session, isLogin bool) (*Client, error) {
	mfaCache := conf.ClientConfig.MFACacheFile
	err := os.MkdirAll(path.Dir(mfaCache), 0755)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create mfa cache dir at %s", path.Dir(mfaCache))
	}
	userTokenProvider := cziAWS.NewUserTokenProvider(sess, mfaCache, isLogin)
	kmsauthCredentials := credentials.NewCredentials(userTokenProvider)
	kmsauthAWSConfig := &aws.Config{
		Credentials: kmsauthCredentials,
	}

	lambdaAWSCredentials := stscreds.NewCredentials(sess, conf.LambdaConfig.AWSProfile, func(p *stscreds.AssumeRoleProvider) {
		p.TokenProvider = stscreds.StdinTokenProvider
	})
	lambdaAWSConfig := &aws.Config{
		Credentials: lambdaAWSCredentials,
	}

	awsClient := cziAWS.NewClient(sess, kmsauthAWSConfig, lambdaAWSConfig)
	username, err := awsClient.IAM.GetUsername()
	if err != nil {
		// TODO this could have a more informative user error
		return nil, err
	}
	// TODO: just using the first region for now...
	authContext := &kmsauth.AuthContextV2{
		From:     username,
		To:       conf.LambdaConfig.FunctionName,
		UserType: "user",
	}
	cacheFile := conf.ClientConfig.KMSAuthCacheFile

	tokenGenerator := kmsauth.NewTokenGenerator(
		conf.LambdaConfig.Regions[0].KMSAuthKeyID,
		kmsauth.TokenVersion2,
		sess,
		kmsauthAWSConfig,
		time.Minute*30,
		&cacheFile,
		authContext,
	)
	return &Client{conf: conf, tokenGenerator: tokenGenerator}, nil
}

// LambdaPayload is the payload for the bless lambda
type LambdaPayload struct {
	BastionUser     string   `json:"bastion_user,omitempty"`
	RemoteUsernames []string `json:"remote_usernames,omitempty"`
	BastionIPs      []string `json:"bastion_ips,omitempty"`
	Command         string   `json:"command,omitempty"`
	PublicKeyToSign string   `json:"public_key_to_sign,omitempty"`
	KMSAuthToken    string   `json:"kmsauth_token"`
}

// LambdaResponse is a bless lambda response
type LambdaResponse struct {
	Certificate *string `json:"certificate"`
}

// RequestKMSAuthToken requests a new kmsauth token
func (c *Client) RequestKMSAuthToken() (*kmsauth.EncryptedToken, error) {
	token, err := c.tokenGenerator.GetEncryptedToken()
	return token, errors.Wrap(err, "Error requesting kmsauth token")
}

// RequestCert requests a cert
func (c *Client) RequestCert(payload *LambdaPayload) error {
	_, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "Could not json encode payload")
	}
	// responseBytes, err := c.Aws.Lambda.Execute()
	return nil
}
