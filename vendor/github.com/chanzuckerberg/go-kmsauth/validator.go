package kmsauth

import (
	"encoding/base64"
	"encoding/json"
	"time"

	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/pkg/errors"
)

// TokenValidator validates a token
type TokenValidator struct {
	// An auth context
	AuthContext AuthContext
	// TokenLifetime is the max lifetime we accept tokens to have
	TokenLifetime time.Duration
	// AuthKeys are a set of KMSKeys to accept
	AuthKeys map[string]bool
	// AwsClient for kms encryption
	AwsClient *cziAWS.Client
}

// NewTokenValidator returns a new token validator
func NewTokenValidator(
	authKeys map[string]bool,
	authContext AuthContext,
	tokenLifetime time.Duration,
	awsClient *cziAWS.Client,
) *TokenValidator {
	return &TokenValidator{
		AuthKeys:      authKeys,
		AuthContext:   authContext,
		TokenLifetime: tokenLifetime,
		AwsClient:     awsClient,
	}
}

// validate validates the token validator
func (tv *TokenValidator) validate() error {
	if tv == nil {
		return errors.New("Nil token validator")
	}
	return tv.AuthContext.Validate()
}

// ValidateToken validates a token
func (tv *TokenValidator) ValidateToken(tokenb64 string) error {
	token, err := tv.decryptToken(tokenb64)
	if err != nil {
		return err
	}
	return token.IsValid(tv.TokenLifetime)
}

// decryptToken decrypts a token
func (tv *TokenValidator) decryptToken(tokenb64 string) (*Token, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(tokenb64)
	if err != nil {
		return nil, errors.Wrap(err, "Could not base64 decode token")
	}
	plaintext, keyID, err := tv.AwsClient.KMS.Decrypt(ciphertext, tv.AuthContext.GetKMSContext())
	if err != nil {
		return nil, err
	}
	ok, _ := tv.AuthKeys[keyID]
	if !ok {
		return nil, errors.Errorf("Invalid KMS key used %s", keyID)
	}
	token := &Token{}
	err = json.Unmarshal(plaintext, token)
	return token, errors.Wrap(err, "Could not unmarshal token from plaintext")
}
