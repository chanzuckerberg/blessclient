package kmsauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sync"
	"time"

	cziAWS "github.com/chanzuckerberg/go-misc/aws"
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
	awsClient *cziAWS.Client
	// rw mutex
	mutex sync.RWMutex
}

// NewTokenGenerator returns a new token generator
func NewTokenGenerator(
	authKey string,
	tokenVersion TokenVersion,
	tokenLifetime time.Duration,
	tokenCacheFile *string,
	authContext AuthContext,
	awsClient *cziAWS.Client,
) *TokenGenerator {
	return &TokenGenerator{
		AuthKey:        authKey,
		TokenVersion:   tokenVersion,
		TokenLifetime:  tokenLifetime,
		TokenCacheFile: tokenCacheFile,
		AuthContext:    authContext,
		awsClient:      awsClient,
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
		log.Debug("No TokenCacheFile specified")
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
	now := time.Now().UTC()
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

// getToken gets a token
func (tg *TokenGenerator) getToken() (*Token, error) {
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
func (tg *TokenGenerator) GetEncryptedToken(ctx context.Context) (*EncryptedToken, error) {
	token, err := tg.getToken()
	if err != nil {
		return nil, err
	}

	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return nil, errors.Wrap(err, "Could not marshal token")
	}

	encryptedStr, err := tg.awsClient.KMS.EncryptBytes(
		ctx,
		tg.AuthKey,
		tokenBytes,
		tg.AuthContext.GetKMSContext())

	if err != nil {
		return nil, err
	}

	encryptedToken := EncryptedToken(encryptedStr)

	tokenCache := &TokenCache{
		Token:          *token,
		EncryptedToken: encryptedToken,
		AuthContext:    tg.AuthContext.GetKMSContext(),
	}
	err = tg.cacheToken(tokenCache)
	if err != nil {
		return nil, err
	}
	return &encryptedToken, err
}
