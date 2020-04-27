package bless

import (
	"context"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/json"
	"os"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziSSH "github.com/chanzuckerberg/blessclient/pkg/ssh"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// baseClient is a client with shared functionality
type baseClient struct {
	conf *config.Config
}

func newBaseClient(conf *config.Config) *baseClient {
	return &baseClient{
		conf: conf,
	}
}

// LambdaResponse is a lambda response
type LambdaResponse struct {
	Certificate  *string `json:"certificate,omitempty"`
	ErrorType    *string `json:"errorType"`
	ErrorMessage *string `json:"errorMessage"`
}

// RequestCert requests a cert

func (c *baseClient) updateSSHAgent() error {
	if !c.conf.ClientConfig.UpdateSSHAgent {
		logrus.Debug("Skipping adding to ssh-agent")
		return nil
	}

	// TODO(el): move to fail earlier
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return errors.New("SSH_AUTH_SOCK environment variable empty")
	}

	a, err := cziSSH.GetSSHAgent(authSock)
	if err != nil {
		return err
	}
	defer a.Close()

	s, err := cziSSH.NewSSH(c.conf.ClientConfig.SSHPrivateKey)
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
	certLifetime := int64(cert.ValidBefore) - time.Now().Unix()
	logrus.Debugf("SSH_AUTH_SOCK: adding key to agent with %ds ttl", certLifetime)

	err = c.removeKeyFromAgent(a, privKey)
	if err != nil {
		// we ignore this error since duplicates don't
		// typically cause any issues
		logrus.Debugf("could not remove cert from ssh agent: %s", err.Error())
	}

	key := agent.AddedKey{
		PrivateKey:   privKey,
		Certificate:  cert,
		Comment:      "Added by blessclient",
		LifetimeSecs: uint32(certLifetime),
	}

	return errors.Wrap(a.Add(key), "Could not add key/certificate to SSH_AGENT_SOCK")
}

func (c *baseClient) removeKeyFromAgent(a agent.ExtendedAgent, privKey interface{}) error {
	var pubKey ssh.PublicKey
	var err error

	switch typedPrivKey := privKey.(type) {
	case *rsa.PrivateKey:
		pubKey, err = ssh.NewPublicKey(typedPrivKey.Public())
		if err != nil {
			return errors.Wrap(err, "could not parse public key from rsa.Private key")
		}
	case *dsa.PrivateKey:
		pubKey, err = ssh.NewPublicKey(&typedPrivKey.PublicKey)
		if err != nil {
			return errors.Wrap(err, "could not parse public key from dsa.Private key")
		}
	case *ecdsa.PrivateKey:
		pubKey, err = ssh.NewPublicKey(typedPrivKey.Public())
		if err != nil {
			return errors.Wrap(err, "could not parse public key from ecdsa.Private key")
		}
	case ed25519.PrivateKey:
		pubKey, err = ssh.NewPublicKey(typedPrivKey.Public())
		if err != nil {
			return errors.Wrap(err, "could not parse public key from ed25519.Private key")
		}
	default:
		return errors.Errorf("can't remove public key from agent since wrong type %T", privKey)
	}

	return errors.Wrap(a.Remove(pubKey), "could not remove pub key from agent")
}

func (c *baseClient) getCert(
	ctx context.Context,
	awsClient *cziAWS.Client,
	payload []byte) (*LambdaResponse, error) {

	responseBytes, err := awsClient.Lambda.ExecuteWithQualifier(
		ctx,
		c.conf.LambdaConfig.FunctionName,
		c.conf.LambdaConfig.FunctionVersion,
		payload,
	)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Raw lambda response %s", string(responseBytes))
	lambdaReponse := &LambdaResponse{}
	err = json.Unmarshal(responseBytes, lambdaReponse)
	if err != nil {
		return nil, errors.Wrap(err, "Could not deserialize lambda reponse")
	}

	logrus.Debugf("Parsed lambda response %s", spew.Sdump(lambdaReponse))
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
