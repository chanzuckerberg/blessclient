package bless

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziSSH "github.com/chanzuckerberg/blessclient/pkg/ssh"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/chanzuckerberg/go-misc/kmsauth"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// KMSAuthClient is a client that works with kmsauth identity assertions
type KMSAuthClient struct {
	baseClient *baseClient

	aws      *cziAWS.Client
	tg       *kmsauth.TokenGenerator
	conf     *config.Config
	username string
}

// KMSAuthLambdaPayload is the payload for the bless lambda
type KMSAuthLambdaPayload struct {
	RemoteUsernames string `json:"remote_usernames,omitempty"`
	BastionIPs      string `json:"bastion_ips,omitempty"`
	BastionUser     string `json:"bastion_user,omitempty"`
	BastionUserIP   string `json:"bastion_user_ip,omitempty"`
	Command         string `json:"command,omitempty"`
	PublicKeyToSign string `json:"public_key_to_sign,omitempty"`
	KMSAuthToken    string `json:"kmsauth_token"`
}

func (k *KMSAuthLambdaPayload) Marshal() ([]byte, error) {
	payloadB, err := json.Marshal(k)
	return payloadB, errors.Wrap(err, "could not serialize payload")
}

// New returns a new client
func NewKMSAuthClient(conf *config.Config) *KMSAuthClient {
	return &KMSAuthClient{
		baseClient: newBaseClient(conf),
		conf:       conf,
	}
}

// WithAwsClient configures an aws client
func (c *KMSAuthClient) WithAwsClient(client *cziAWS.Client) *KMSAuthClient {
	c.aws = client
	return c
}

// WithUsername configures the username
func (c *KMSAuthClient) WithUsername(username string) *KMSAuthClient {
	c.username = username
	return c
}

// WithTokenGenerator configures a token generator
func (c *KMSAuthClient) WithTokenGenerator(tg *kmsauth.TokenGenerator) *KMSAuthClient {
	c.tg = tg
	return c
}

// RequestKMSAuthToken requests a new kmsauth token
func (c *KMSAuthClient) RequestKMSAuthToken(ctx context.Context) (*kmsauth.EncryptedToken, error) {
	token, err := c.tg.GetEncryptedToken(ctx)
	return token, errors.Wrap(err, "Error requesting kmsauth token")
}

func (c *KMSAuthClient) RequestCert(ctx context.Context) error {
	logrus.Debugf("Requesting certificate")

	payload := &KMSAuthLambdaPayload{
		BastionUser:     c.username,
		RemoteUsernames: strings.Join(c.conf.GetRemoteUsers(c.username), ","),
		BastionIPs:      strings.Join(c.conf.ClientConfig.BastionIPS, ","),
		BastionUserIP:   "0.0.0.0/0",
		Command:         "*",
	}

	s, err := cziSSH.NewSSH(c.conf.ClientConfig.SSHPrivateKey)
	if err != nil {
		return err
	}

	logrus.Debug("Requesting new cert")
	pubKey, err := s.ReadPublicKey()
	if err != nil {
		return err
	}
	logrus.Debugf("Using public key: %s", string(pubKey))

	token, err := c.RequestKMSAuthToken(ctx)
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("Missing KMSAuth Token")
	}
	logrus.Debugf("With KMSAuthToken %s", token.String())

	payload.KMSAuthToken = token.String()
	payload.PublicKeyToSign = string(pubKey)
	logrus.Debugf("Requesting cert with lambda payload %s", spew.Sdump(payload))

	payloadBytes, err := payload.Marshal()
	if err != nil {
		return nil
	}

	lambdaResponse, err := c.baseClient.getCert(ctx, c.aws, payloadBytes)
	if err != nil {
		return err
	}

	err = s.WriteCert([]byte(*lambdaResponse.Certificate))
	if err != nil {
		return errors.Wrap(err, "Error writing cert to disk")
	}

	err = c.baseClient.updateSSHAgent()
	if err != nil {
		// Not a fatal error so just printing a warning
		logrus.WithError(err).Warn("Error adding certificate to ssh-agent, is your ssh-agent running?")
	}
	return nil
}
