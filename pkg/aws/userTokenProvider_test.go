package aws_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	cziAws "github.com/chanzuckerberg/blessclient/pkg/aws"
	"github.com/stretchr/testify/assert"
)

func TestUserTokenProviderCached(t *testing.T) {
	a := assert.New(t)
	isLogin := true
	f, err := ioutil.TempFile("", "cache")
	a.Nil(err)
	defer f.Close()
	defer os.Remove(f.Name())
	mockIAM := mockIAMSvc{}
	provider := cziAws.NewUserTokenProvider(sess, f.Name(), isLogin, stscreds.StdinTokenProvider)
	provider.Client.IAM.Svc = mockIAM
	cached := &cziAws.UserTokenProviderCache{
		Expiration:      aws.Time(time.Now().Add(time.Hour)),
		AccessKeyID:     aws.String("access_key_id"),
		SecretAccessKey: aws.String("secret_access_key"),
		SessionToken:    aws.String("session_token"),
	}
	b, err := json.Marshal(cached)
	a.Nil(err)
	err = ioutil.WriteFile(f.Name(), b, 0644)
	a.Nil(err)
	creds, err := provider.Retrieve()
	a.Nil(err)
	a.Equal(*cached.AccessKeyID, creds.AccessKeyID)
	a.Equal(*cached.SecretAccessKey, creds.SecretAccessKey)
	a.Equal(*cached.SessionToken, creds.SessionToken)
}

func TestUserTokenProviderCachedExpired(t *testing.T) {
	a := assert.New(t)
	isLogin := true
	f, err := ioutil.TempFile("", "cache")
	a.Nil(err)
	defer f.Close()
	defer os.Remove(f.Name())

	mockIAM := mockIAMSvc{}
	mockIAM.ResponsesGetUser = append(
		mockIAM.ResponsesGetUser,
		&iam.GetUserOutput{
			User: &iam.User{
				Arn:      aws.String("Some arn"),
				UserName: aws.String("username"),
			},
		},
	)

	mockIAM.ResponsesListMFADevicesPages = &iam.ListMFADevicesOutput{
		MFADevices: []*iam.MFADevice{
			&iam.MFADevice{
				UserName:     aws.String("username"),
				SerialNumber: aws.String("serial number"),
			},
		},
	}

	mockSTS := mockSTSSvc{}
	mockSTS.ResponsesGetSessionToken = &sts.GetSessionTokenOutput{
		Credentials: &sts.Credentials{
			AccessKeyId:     aws.String("sts id"),
			SecretAccessKey: aws.String("sts secret"),
			SessionToken:    aws.String("sts token"),
			Expiration:      aws.Time(time.Now().Add(time.Hour)),
		},
	}

	tokenProvider := func() (string, error) {
		return "tokentoken", nil
	}
	provider := cziAws.NewUserTokenProvider(sess, f.Name(), isLogin, tokenProvider)

	provider.Client.IAM.Svc = mockIAM
	provider.Client.STS.Svc = mockSTS

	cached := &cziAws.UserTokenProviderCache{
		Expiration:      aws.Time(time.Now().Add(-1 * time.Hour)),
		AccessKeyID:     aws.String("access_key_id"),
		SecretAccessKey: aws.String("secret_access_key"),
		SessionToken:    aws.String("session_token"),
	}

	b, err := json.Marshal(cached)
	a.Nil(err)
	err = ioutil.WriteFile(f.Name(), b, 0644)
	a.Nil(err)
	creds, err := provider.Retrieve()
	a.Nil(err)
	a.Equal(*mockSTS.ResponsesGetSessionToken.Credentials.AccessKeyId, creds.AccessKeyID)
	a.Equal(*mockSTS.ResponsesGetSessionToken.Credentials.SecretAccessKey, creds.SecretAccessKey)
	a.Equal(*mockSTS.ResponsesGetSessionToken.Credentials.SessionToken, creds.SessionToken)

	cachedB, err := ioutil.ReadFile(f.Name())
	a.Nil(err)
	err = json.Unmarshal(cachedB, cached)
	a.Nil(err)

	a.Equal(*cached.AccessKeyID, creds.AccessKeyID)
	a.Equal(*cached.SecretAccessKey, creds.SecretAccessKey)
}

// MOCKS
// ---------------------------------------------------
// Returns an empty client.Config
var sess = func() *session.Session {
	// server is the mock server that simply writes a 200 status back to the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return session.Must(session.NewSession(&aws.Config{
		DisableSSL: aws.Bool(true),
		Endpoint:   aws.String(server.URL),
		Region:     aws.String("mock"),
	}))
}()

type mockConfigProvider struct{}

func (cp mockConfigProvider) ClientConfig(serviceName string, cfgs ...*aws.Config) client.Config {
	return client.Config{}
}

type mockIAMSvc struct {
	iamiface.IAMAPI
	ResponsesGetUser             []*iam.GetUserOutput
	ResponsesListMFADevicesPages *iam.ListMFADevicesOutput
}

func (m mockIAMSvc) ListMFADevicesPages(in *iam.ListMFADevicesInput, f func(o *iam.ListMFADevicesOutput, lastPage bool) bool) error {
	f(m.ResponsesListMFADevicesPages, true)
	return nil
}

func (m mockIAMSvc) GetUser(in *iam.GetUserInput) (*iam.GetUserOutput, error) {
	if len(m.ResponsesGetUser) < 1 {
		return nil, fmt.Errorf("Mock no response queued up")
	}
	ret := m.ResponsesGetUser[0]
	m.ResponsesGetUser = m.ResponsesGetUser[1:]
	return ret, nil
}

type mockSTSSvc struct {
	stsiface.STSAPI

	ResponsesGetSessionToken *sts.GetSessionTokenOutput
}

func (m mockSTSSvc) GetSessionToken(in *sts.GetSessionTokenInput) (*sts.GetSessionTokenOutput, error) {
	return m.ResponsesGetSessionToken, nil
}
