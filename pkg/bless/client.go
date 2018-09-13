package client

import (
	"encoding/json"
	"path"
	"time"

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
func New(conf *config.Config, sess *session.Session) (*Client, error) {
	awsClient := cziAWS.NewClient(sess)
	username, err := awsClient.IAM.GetUsername()
	if err != nil {
		// TODO this could have a more informative user error
		return nil, err
	}
	// TODO: just using the first region for now...
	authContext := &kmsauth.AuthContextV2{
		From:     username,
		To:       conf.Regions[0].Name,
		UserType: "user",
	}
	cacheFile := path.Join(conf.ClientConfig.CacheDir, conf.ClientConfig.KMSAuthCacheFile)
	tokenGenerator := kmsauth.NewTokenGenerator(
		conf.Regions[0].KMSAuthKeyID,
		kmsauth.TokenVersion2,
		sess,
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
