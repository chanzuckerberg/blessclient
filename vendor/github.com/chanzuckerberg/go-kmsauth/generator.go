package kmsauth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/chanzuckerberg/go-kmsauth/kmsauth/aws"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TokenGenerator generates a token
type TokenGenerator struct {
	// AuthKey the key_arn or alias to use for authentication
	AuthKey string
	// TokenVersion is a kmsauth token version
	TokenVersion TokenVersion
	// The token lifetime
	TokenLifetime time.Duration
	// A file to use as a cache
	TokenCacheFile *string
	// An auth context
	AuthContext AuthContext

	// AwsClient for kms encryption
	AwsClient *aws.Client
	// rw mutex
	mutex sync.RWMutex
}

// NewTokenGenerator returns a new token generator
func NewTokenGenerator(
	authKey string,
	tokenVersion TokenVersion,
	sess *session.Session,
	tokenLifetime time.Duration,
	tokenCacheFile *string,
	authContext AuthContext,
) *TokenGenerator {
	return &TokenGenerator{
		AuthKey:        authKey,
		TokenVersion:   tokenVersion,
		TokenLifetime:  tokenLifetime,
		TokenCacheFile: tokenCacheFile,
		AuthContext:    authContext,

		AwsClient: aws.NewClient(sess),
	}
}

// Validate validates the TokenGenerator
func (tg *TokenGenerator) Validate() error {
	if tg == nil {
		return errors.New("Nil token generator")
	}
	return tg.AuthContext.Validate()
}

// getCachedToken tries to fetch a token from the cache
func (tg *TokenGenerator) getCachedToken() (*Token, error) {
	if tg.TokenCacheFile == nil {
		log.Warn("No TokenCacheFile specified")
		return nil, nil
	}
	// lock for reading
	tg.mutex.RLock()
	defer tg.mutex.RUnlock()

	_, err := os.Stat(*tg.TokenCacheFile)
	if os.IsNotExist(err) {
		// token cache file does not exist
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "Error os.Stat token cache at %s", *tg.TokenCacheFile)
	}
	cacheBytes, err := ioutil.ReadFile(*tg.TokenCacheFile)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not open token cache file %s", *tg.TokenCacheFile))
	}

	tokenCache := &TokenCache{}
	err = json.Unmarshal(cacheBytes, tokenCache)
	if err != nil {
		return nil, errors.Wrap(err, "Could not unmarshal token cache")
	}
	// Compare token cache with current params
	ok := reflect.DeepEqual(tokenCache.AuthContext, tg.AuthContext.GetKMSContext())
	if !ok {
		log.Debug("Cached token invalid")
		return nil, nil
	}
	now := time.Now()
	// subtract timeSkew to account for clock skew
	notAfter := tokenCache.Token.NotAfter.Add(-1 * timeSkew)
	if now.After(notAfter) { // expired, need new token
		return nil, nil
	}
	return &tokenCache.Token, nil
}

// cacheToken caches a token
func (tg *TokenGenerator) cacheToken(tokenCache *TokenCache) error {
	if tg.TokenCacheFile == nil {
		log.Debug("No TokenCacheFile specified")
		return nil
	}
	// lock for writing
	tg.mutex.Lock()
	defer tg.mutex.Unlock()

	dir := path.Dir(*tg.TokenCacheFile)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "Could not create cache directories %s", dir)
	}

	data, err := json.Marshal(tokenCache)
	if err != nil {
		return errors.Wrap(err, "Could not marshal token cache")
	}

	err = ioutil.WriteFile(*tg.TokenCacheFile, data, 0644)
	return errors.Wrap(err, "Could not write token to cache")
}

// GetToken gets a token
func (tg *TokenGenerator) GetToken() (*Token, error) {
	token, err := tg.getCachedToken()
	if err != nil {
		return nil, err
	}
	// If we could not find a token then return a new one
	if token != nil {
		return token, err
	}
	return NewToken(tg.TokenLifetime), nil
}

// GetEncryptedToken returns the encrypted kmsauth token
func (tg *TokenGenerator) GetEncryptedToken() (*EncryptedToken, error) {
	token, err := tg.GetToken()
	if err != nil {
		return nil, err
	}

	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return nil, errors.Wrap(err, "Could not marshal token")
	}

	encryptedStr, err := tg.AwsClient.KMS.EncryptBytes(
		tg.AuthKey,
		tokenBytes,
		tg.AuthContext.GetKMSContext())

	encryptedToken := EncryptedToken(encryptedStr)
	return &encryptedToken, err
}
