package bless

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type LambdaResponse struct {
	Certificate  *string `json:"certificate,omitempty"`
	ErrorType    *string `json:"errorType"`
	ErrorMessage *string `json:"errorMessage"`
}

// OIDC is an oidc client
type OIDC struct {
	awsClient *cziAWS.Client

	lambdaConfig *config.LambdaConfig
}

// NewOIDC returns a new OIDC client
func NewOIDC(
	awsClient *cziAWS.Client,
	lambdaConfig *config.LambdaConfig,
) *OIDC {
	return &OIDC{
		awsClient:    awsClient,
		lambdaConfig: lambdaConfig,
	}
}

// RequestCert requests a new certificate
func (o *OIDC) RequestCert(
	ctx context.Context,
	awsClient *cziAWS.Client,
	signingRequest *SigningRequest,
) (*ssh.Certificate, error) {
	payload, err := json.Marshal(signingRequest)
	if err != nil {
		return nil, errors.Wrap(err, "could not json marshal payload")
	}
	return o.getCert(ctx, payload)
}

func (o *OIDC) getCert(ctx context.Context, payload []byte) (*ssh.Certificate, error) {
	responseBytes, err := o.awsClient.Lambda.ExecuteWithQualifier(
		ctx,
		o.lambdaConfig.FunctionName,
		o.lambdaConfig.FunctionVersion,
		payload,
	)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Raw lambda response %s", string(responseBytes))
	lambdaResponse := &LambdaResponse{}
	err = json.Unmarshal(responseBytes, lambdaResponse)
	if err != nil {
		return nil, errors.Wrap(err, "Could not deserialize lambda reponse")
	}

	logrus.Debugf("Parsed lambda response %s", spew.Sdump(lambdaResponse))
	if lambdaResponse.ErrorType != nil {
		if lambdaResponse.ErrorMessage != nil {
			return nil, errors.Errorf("bless error: %s: %s", *lambdaResponse.ErrorType, *lambdaResponse.ErrorMessage)
		}
		return nil, errors.Errorf("bless error: %s", *lambdaResponse.ErrorType)
	}

	if lambdaResponse.Certificate == nil {
		return nil, errors.New("No certificate in response")
	}

	certBytes, err := base64.StdEncoding.DecodeString(*lambdaResponse.Certificate)
	if err != nil {
		return nil, errors.Wrap(err, "could not b64 decode certificate")
	}

	pubKey, err := ssh.ParsePublicKey(certBytes)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse public key from lambda response")
	}

	cert, ok := pubKey.(*ssh.Certificate)
	if !ok {
		return nil, errors.Errorf("lambda return type not *ssh.Certificate")
	}

	return cert, nil
}

// SigningRequest is a request for a certificate
// TODO(el): copy/paste from ssh-ca-lambda. Use that once open source
type SigningRequest struct {
	RemoteUsernames RemoteUsernames  `json:"remote_usernames,omitempty"`
	PublicKeyToSign *PublicKeyToSign `json:"public_key_to_sign,omitempty"`

	// IdentityAssertion used to verify the caller
	Identity Identity `json:"identity,omitempty"`
}

// Identity represents different types of identity assertions
// that we can use
type Identity struct {
	OktaAccessToken *OktaAccessTokenInput `json:"okta_identity,omitempty"`
}

type OktaAccessTokenInput struct {
	AccessToken string
}

type RemoteUsernames []string

// String returns the string representation of RemoteUsernames
func (ru RemoteUsernames) String() string {
	return strings.Join(ru, ",")
}

// List returns the []string representation of RemoteUsernames
func (ru RemoteUsernames) List() []string {
	return ru
}

func (ru RemoteUsernames) MarshalJSON() ([]byte, error) {
	return json.Marshal(ru.String())
}

func (ru RemoteUsernames) UnmarshalJSON(data []byte) error {
	var remoteUsernames string
	err := json.Unmarshal(data, &remoteUsernames)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling remote usernames")
	}
	ru = RemoteUsernames(strings.Split(remoteUsernames, ",")) // nolint: go-lint
	return nil
}

type PublicKeyToSign struct {
	key crypto.PublicKey
}

func NewPublicKeyToSign(key crypto.PublicKey) *PublicKeyToSign {
	return &PublicKeyToSign{key: key}
}

func (p *PublicKeyToSign) MarshalJSON() ([]byte, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(p.key)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal public key")
	}

	data := base64.StdEncoding.EncodeToString(pubBytes)
	return json.Marshal(map[string]string{"publicKey": data})
}

func (p *PublicKeyToSign) UnmarshalJSON(data []byte) error {
	// first unmarshal the intermediate
	intermediate := map[string]string{}
	err := json.Unmarshal(data, &intermediate)
	if err != nil {
		return errors.Wrap(err, "could not json unmarshal public key")
	}

	derB64, ok := intermediate["publicKey"]
	if !ok {
		return errors.New("unknown serialization format")
	}

	derBytes, err := base64.StdEncoding.DecodeString(derB64)
	if err != nil {
		return errors.Wrap(err, "could not b64 decode public key")
	}

	pub, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return errors.Wrap(err, "could not parse DER bytes")
	}

	p.key = pub
	return nil
}
