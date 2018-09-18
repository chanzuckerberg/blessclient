package aws

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/pkg/errors"
)

const (
	// UserTokenProviderName is the name of this provider
	UserTokenProviderName = "UserTokenProvider"
)

// UserTokenProviderCache caches mfa tokens
// Need this to json serialize/deserialize
type UserTokenProviderCache struct {
	Expiration *time.Time `json:"expiration"`

	AccessKeyID     *string `json:"access_key_id"`
	SecretAccessKey *string `json:"secret_access_key"`
	SessionToken    *string `json:"session_token"`
}

// UserTokenProvider is a token provider that gets sts tokens for a user
type UserTokenProvider struct {
	credentials.Expiry

	Client   *Client
	Duration time.Duration

	cacheFile string
	m         sync.RWMutex

	expireWindow  time.Duration
	isLogin       bool
	tokenProvider func() (string, error)
}

// NewUserTokenProvider returns a new user token provider
func NewUserTokenProvider(
	c client.ConfigProvider,
	cacheFile string,
	isLogin bool,
	tokenProvider func() (string, error)) *UserTokenProvider {
	p := &UserTokenProvider{
		Client:   NewClient(c, nil, nil),
		Duration: stscreds.DefaultDuration,

		cacheFile:     cacheFile,
		expireWindow:  10 * time.Second,
		isLogin:       isLogin,
		tokenProvider: tokenProvider,
	}
	return p
}

// try reading from file cache
func (p *UserTokenProvider) fromCache() (*sts.Credentials, error) {
	p.m.RLock()
	defer p.m.RUnlock()

	b, err := ioutil.ReadFile(p.cacheFile)
	if err != nil {
		// no cache - return nil credentials
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Could not open mfa token cache %s", p.cacheFile)
	}

	var tokenCache UserTokenProviderCache
	err = json.Unmarshal(b, &tokenCache)
	if err != nil {
		return nil, errors.Wrapf(err, "Cache corrupted at %s, please delete", p.cacheFile)
	}

	// expired - return nil credentials
	if time.Now().After(tokenCache.Expiration.Add(-1 * p.expireWindow)) {
		return nil, nil
	}

	// else return cached
	creds := &sts.Credentials{
		AccessKeyId:     tokenCache.AccessKeyID,
		SecretAccessKey: tokenCache.SecretAccessKey,
		SessionToken:    tokenCache.SessionToken,

		Expiration: tokenCache.Expiration,
	}
	return creds, nil
}

// toCache writes to cache
func (p *UserTokenProvider) toCache(creds *sts.Credentials) error {
	p.m.Lock()
	defer p.m.Unlock()

	tokenCache := &UserTokenProviderCache{
		AccessKeyID:     creds.AccessKeyId,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,

		Expiration: creds.Expiration,
	}

	b, err := json.Marshal(tokenCache)
	if err != nil {
		return errors.Wrap(err, "Could not marshal token to cache")
	}

	err = ioutil.WriteFile(p.cacheFile, b, 0644)
	return errors.Wrap(err, "Could not write token to cache")
}

// Retrieve generates a new set of temporary gredentials using STS.
func (p *UserTokenProvider) Retrieve() (credentials.Value, error) {
	creds := credentials.Value{}

	stsCreds, err := p.fromCache()
	if err != nil {
		return creds, err
	}

	// If we're not loging in then odds are stdin is not connected
	// and we can't proceed. Ask user to log in
	// TODO perhaps simplify this
	if !p.isLogin && (stsCreds == nil ||
		stsCreds.Expiration == nil ||
		time.Now().After(stsCreds.Expiration.Add(-30*time.Second))) {
		return creds, errors.Wrap(
			errs.ErrLoginRequired,
			"Looks like you need to log in. Please run blessclient login to do so.")
	}

	if stsCreds == nil {
		username, err := p.Client.IAM.GetUsername()
		if err != nil {
			return creds, err
		}
		mfaSerial, err := p.Client.IAM.GetMFASerial(aws.String(username))
		if err != nil {
			return creds, err
		}
		token, err := p.tokenProvider()
		if err != nil {
			return creds, errors.Wrap(err, "Could not read MFA token")
		}
		stsCreds, err = p.Client.STS.GetSTSToken(mfaSerial, token)
		if err != nil {
			return creds, err
		}
	}

	// Check that we have all of these
	if stsCreds == nil ||
		stsCreds.AccessKeyId == nil ||
		stsCreds.Expiration == nil ||
		stsCreds.SecretAccessKey == nil ||
		stsCreds.SessionToken == nil {
		return creds, errors.New("Received malformed credentials from aws.Sts.GetSTSToken")
	}

	p.SetExpiration(*stsCreds.Expiration, p.expireWindow)
	creds.AccessKeyID = *stsCreds.AccessKeyId
	creds.SecretAccessKey = *stsCreds.SecretAccessKey
	creds.SessionToken = *stsCreds.SessionToken

	return creds, p.toCache(stsCreds)
}
