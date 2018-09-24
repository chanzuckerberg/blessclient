package bless

import (
	"encoding/json"
	"strings"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/blessclient/pkg/ssh"
	"github.com/chanzuckerberg/go-kmsauth"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Client is a bless client
type Client struct {
	Aws      *cziAWS.Client
	tg       *kmsauth.TokenGenerator
	conf     *config.Config
	username string
}

// New returns a new client
func New(conf *config.Config) *Client {
	return &Client{
		conf: conf,
	}
}

// WithAwsClient configures an aws client
func (c *Client) WithAwsClient(client *cziAWS.Client) *Client {
	c.Aws = client
	return c
}

// WithUsername configures the username
func (c *Client) WithUsername(username string) *Client {
	c.username = username
	return c
}

// WithTokenGenerator configures a token generator
func (c *Client) WithTokenGenerator(tg *kmsauth.TokenGenerator) *Client {
	c.tg = tg
	return c
}

// LambdaPayload is the payload for the bless lambda
type LambdaPayload struct {
	RemoteUsernames string `json:"remote_usernames,omitempty"`
	BastionIPs      string `json:"bastion_ips,omitempty"`
	BastionUser     string `json:"bastion_user,omitempty"`
	BastionUserIP   string `json:"bastion_user_ip,omitempty"`
	Command         string `json:"command,omitempty"`
	PublicKeyToSign string `json:"public_key_to_sign,omitempty"`
	KMSAuthToken    string `json:"kmsauth_token"`
}

// LambdaResponse is a lambda response
type LambdaResponse struct {
	Certificate  *string `json:"certificate,omitempty"`
	ErrorType    *string `json:"errorType"`
	ErrorMessage *string `json:"errorMessage"`
}

// RequestKMSAuthToken requests a new kmsauth token
func (c *Client) RequestKMSAuthToken() (*kmsauth.EncryptedToken, error) {
	token, err := c.tg.GetEncryptedToken()
	return token, errors.Wrap(err, "Error requesting kmsauth token")
}

// RequestCert requests a cert
func (c *Client) RequestCert() error {
	log.Debugf("Requesting certificate")
	payload := &LambdaPayload{
		BastionUser:     c.username,
		RemoteUsernames: strings.Join(c.conf.ClientConfig.RemoteUsers, ","),
		BastionIPs:      strings.Join(c.conf.ClientConfig.BastionIPS, ","),
		BastionUserIP:   "0.0.0.0/0",
		Command:         "*",
	}

	s, err := ssh.NewSSH(c.conf.ClientConfig.SSHPrivateKey)
	if err != nil {
		return err
	}

	isFresh, err := s.IsCertFresh(c.conf)
	if err != nil {
		return err
	}
	if isFresh {
		log.Debug("Cert is already fresh - using it")
		return nil
	}
	log.Debug("Requesting new cert")

	pubKey, err := s.ReadPublicKey()
	if err != nil {
		return err
	}
	log.Debugf("Using public key: %s", string(pubKey))

	token, err := c.RequestKMSAuthToken()
	if err != nil {
		return err
	}
	if token == nil {
		return errs.ErrMissingKMSAuthToken
	}
	log.Debugf("With KMSAuthToken %s", token.String())

	payload.KMSAuthToken = token.String()
	payload.PublicKeyToSign = string(pubKey)
	log.Debugf("Requesting cert with lambda payload %s", spew.Sdump(payload))

	payloadB, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "Could not serialize lambda payload")
	}
	responseBytes, err := c.Aws.Lambda.Execute(c.conf.LambdaConfig.FunctionName, payloadB)
	if err != nil {
		return err
	}
	log.Debugf("Raw lambda response %s", string(responseBytes))
	lambdaReponse := &LambdaResponse{}
	err = json.Unmarshal(responseBytes, lambdaReponse)
	if err != nil {
		return errors.Wrap(err, "Could not deserialize lambda reponse")
	}
	log.Debugf("Parsed lambda response %s", spew.Sdump(lambdaReponse))

	if lambdaReponse.ErrorType != nil {
		if lambdaReponse.ErrorMessage != nil {
			return errors.Errorf("bless error: %s: %s", *lambdaReponse.ErrorType, *lambdaReponse.ErrorMessage)
		}
		return errors.Errorf("bless error: %s", *lambdaReponse.ErrorType)
	}

	if lambdaReponse.Certificate == nil {
		return errs.ErrNoCertificateInResponse
	}
	return s.WriteCert([]byte(*lambdaReponse.Certificate))
}
