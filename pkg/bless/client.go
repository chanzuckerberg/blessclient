package bless

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"strings"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/ssh"
	"github.com/chanzuckerberg/blessclient/pkg/telemetry"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/chanzuckerberg/go-misc/kmsauth"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"golang.org/x/crypto/ssh/agent"
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
func (c *Client) RequestKMSAuthToken(ctx context.Context) (*kmsauth.EncryptedToken, error) {
	ctx, span := trace.StartSpan(ctx, "request_kmsauth")
	defer span.End()
	token, err := c.tg.GetEncryptedToken(ctx)
	return token, errors.Wrap(err, "Error requesting kmsauth token")
}

// RequestCert requests a cert
func (c *Client) RequestCert(ctx context.Context) error {
	log.Debugf("Requesting certificate")
	ctx, span := trace.StartSpan(ctx, "request_cert")
	defer span.End()

	payload := &LambdaPayload{
		BastionUser:     c.username,
		RemoteUsernames: strings.Join(c.conf.GetRemoteUsers(c.username), ","),
		BastionIPs:      strings.Join(c.conf.ClientConfig.BastionIPS, ","),
		BastionUserIP:   "0.0.0.0/0",
		Command:         "*",
	}

	s, err := ssh.NewSSH(c.conf.ClientConfig.SSHPrivateKey)
	if err != nil {
		return err
	}

	// Check to see if ssh client version is compatible with the key type
	s.CheckKeyTypeAndClientVersion(ctx)

	isFresh, err := s.IsCertFresh(c.conf, c.username)
	if err != nil {
		return err
	}
	span.AddAttributes(trace.BoolAttribute(telemetry.FieldFreshCert, isFresh))
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

	token, err := c.RequestKMSAuthToken(ctx)
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("Missing KMSAuth Token")
	}
	log.Debugf("With KMSAuthToken %s", token.String())

	payload.KMSAuthToken = token.String()
	payload.PublicKeyToSign = string(pubKey)
	log.Debugf("Requesting cert with lambda payload %s", spew.Sdump(payload))
	lambdaResponse, err := c.getCert(ctx, payload)
	if err != nil {
		span.AddAttributes(trace.StringAttribute(telemetry.FieldError, err.Error()))
		return err
	}
	err = s.WriteCert([]byte(*lambdaResponse.Certificate))
	if err != nil {
		return errors.Wrap(err, "Error writing cert to disk")
	}
	err = c.updateSSHAgent(ctx)
	if err != nil {
		// Not a fatal error so just printing a warning
		log.WithError(err).Warn("Error adding certificate to ssh-agent, is your ssh-agent running?")
	}
	return nil
}

func (c *Client) updateSSHAgent(ctx context.Context) error {
	if !c.conf.ClientConfig.UpdateSSHAgent {
		log.Debug("Skipping adding to ssh-agent")
		return nil
	}
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return errors.New("SSH_AUTH_SOCK environment variable empty")
	}
	agentSock, err := net.Dial("unix", authSock)
	if err != nil {
		return errors.Wrap(err, "Could not dial SSH_AUTH_SOCK")
	}
	defer agentSock.Close()

	s, err := ssh.NewSSH(c.conf.ClientConfig.SSHPrivateKey)
	if err != nil {
		return err
	}

	privKey, err := s.ReadAndParsePrivateKey()
	if err != nil {
		return err
	}
	cert, err := s.ReadAndParseCert()
	if err != nil {
		return err
	}

	// calculate how many seconds before cert expiry
	certLifetimeSecs := uint32(time.Unix(int64(cert.ValidBefore), 0).Sub(time.Now()) / time.Second)
	log.Debugf("SSH_AUTH_SOCK: adding key to agent with %ds ttl", certLifetimeSecs)

	a := agent.NewClient(agentSock)
	key := agent.AddedKey{
		PrivateKey:   privKey,
		Certificate:  cert,
		Comment:      "Added by blessclient",
		LifetimeSecs: certLifetimeSecs,
	}

	return errors.Wrap(a.Add(key), "Could not add key/certificate to SSH_AGENT_SOCK")
}

func (c *Client) getCert(ctx context.Context, payload *LambdaPayload) (*LambdaResponse, error) {
	ctx, span := trace.StartSpan(ctx, "bless_lambda")
	defer span.End()

	payloadB, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "Could not serialize lambda payload")
	}
	responseBytes, err := c.Aws.Lambda.ExecuteWithQualifier(ctx, c.conf.LambdaConfig.FunctionName, c.conf.LambdaConfig.FunctionVersion, payloadB)
	if err != nil {
		return nil, err
	}
	log.Debugf("Raw lambda response %s", string(responseBytes))
	lambdaReponse := &LambdaResponse{}
	err = json.Unmarshal(responseBytes, lambdaReponse)
	if err != nil {
		return nil, errors.Wrap(err, "Could not deserialize lambda reponse")
	}

	log.Debugf("Parsed lambda response %s", spew.Sdump(lambdaReponse))
	if lambdaReponse.ErrorType != nil {
		if lambdaReponse.ErrorMessage != nil {
			return nil, errors.Errorf("bless error: %s: %s", *lambdaReponse.ErrorType, *lambdaReponse.ErrorMessage)
		}
		return nil, errors.Errorf("bless error: %s", *lambdaReponse.ErrorType)
	}

	if lambdaReponse.Certificate == nil {
		return nil, errors.New("No certificate in response")
	}
	return lambdaReponse, nil
}
