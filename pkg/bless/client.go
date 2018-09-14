package client

import (
	"encoding/json"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	cziAWS "github.com/chanzuckerberg/blessclient/pkg/aws"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/blessclient/pkg/ssh"
	"github.com/chanzuckerberg/go-kmsauth"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Client is a bless client
type Client struct {
	Aws            *cziAWS.Client
	tokenGenerator *kmsauth.TokenGenerator
	conf           *config.Config
	username       string
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
	mfaAWSConfig := &aws.Config{
		Credentials: kmsauthCredentials,
	}

	lambdaAWSCredentials := stscreds.NewCredentials(
		sess,
		conf.LambdaConfig.RoleARN,
		func(p *stscreds.AssumeRoleProvider) {
			p.TokenProvider = stscreds.StdinTokenProvider
		})

	lambdaAWSConfig := &aws.Config{
		Credentials: lambdaAWSCredentials,
	}

	awsClient := cziAWS.NewClient(sess, mfaAWSConfig, lambdaAWSConfig)
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
		mfaAWSConfig,
		time.Minute*30,
		&cacheFile,
		authContext,
	)
	return &Client{conf: conf, tokenGenerator: tokenGenerator, username: username}, nil
}

// LambdaPayload is the payload for the bless lambda
type LambdaPayload struct {
	BastionUser     string      `json:"bastion_user,omitempty"`
	RemoteUsernames []string    `json:"remote_usernames,omitempty"`
	BastionIPs      []string    `json:"bastion_ips,omitempty"`
	BastionCommand  string      `json:"bastion_command,omitempty"`
	PublicKeyToSign string      `json:"public_key_to_sign,omitempty"`
	KMSAuthToken    string      `json:"kmsauth_token"`
	TTL             TTLDuration `json:"ttl"`
}

// TTLDuration is a duration
type TTLDuration struct {
	time.Duration
}

// MarshalJSON marshals to json
func (d TTLDuration) MarshalJSON() ([]byte, error) {
	// Bless expects this duration in seconds
	seconds := int64(d.Duration / time.Second)
	return json.Marshal(strconv.FormatInt(seconds, 10))
}

// LambdaResponse is a lambda response
type LambdaResponse struct {
	Certificate *string `json:"certificate,omitempty"`
}

// RequestKMSAuthToken requests a new kmsauth token
func (c *Client) RequestKMSAuthToken() (*kmsauth.EncryptedToken, error) {
	token, err := c.tokenGenerator.GetEncryptedToken()
	return token, errors.Wrap(err, "Error requesting kmsauth token")
}

// RequestCert requests a cert
func (c *Client) RequestCert() error {
	payload := &LambdaPayload{
		BastionUser:     c.username,
		RemoteUsernames: c.conf.ClientConfig.RemoteUsers,
		BastionIPs:      c.conf.ClientConfig.BastionIPS,
		BastionCommand:  "*",
		TTL:             TTLDuration(c.conf.ClientConfig.CertLifetime),
	}
	_, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "Could not json encode payload")
	}
	s, err := ssh.NewSSH(c.conf.ClientConfig.SSHPrivateKey)
	if err != nil {
		return err
	}

	isFresh, err := s.IsCertFresh()
	if err != nil {
		return err
	}
	if isFresh {
		log.Debug("Cert is already fresh")
		return nil
	}

	pubKey, err := s.ReadPublicKey()
	if err != nil {
		return err
	}

	// TODO: do I need b64 or something?
	payload.PublicKeyToSign = string(pubKey)

	payloadB, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "Could not serialize lambda payload")
	}

	responseBytes, err := c.Aws.Lambda.Execute(c.conf.LambdaConfig.FunctionName, payloadB)
	if err != nil {
		return err
	}

	lambdaReponse := &LambdaResponse{}

	err = json.Unmarshal(responseBytes, lambdaReponse)
	if err != nil {
		return errors.Wrap(err, "Could not deserialize lambda reponse")
	}
	if lambdaReponse.Certificate == nil {
		return errs.ErrNoCertificateInResponse
	}

	return nil
}
