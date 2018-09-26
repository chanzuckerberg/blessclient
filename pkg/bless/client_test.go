package bless_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/go-kmsauth"
	cziAws "github.com/chanzuckerberg/go-misc/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite

	mockKMS    *cziAws.MockKMSSvc
	mockLambda *cziAws.MockLambdaSvc
	client     *bless.Client

	// some default vals
	encryptOut       *kms.EncryptOutput
	lambdaExecuteOut *lambda.InvokeOutput
	conf             *config.Config
	ctx              context.Context
	// cleanup
	pathsToRemove []string
	server        *httptest.Server
}

// cleanup
func (ts *TestSuite) TearDownTest() {
	rmPaths(ts.pathsToRemove)
	ts.server.Close()
}

func (ts *TestSuite) SetupTest() {
	t := ts.T()
	a := assert.New(t)
	ts.ctx = context.Background()

	ts.ctx = context.Background()
	conf, pathsToRemove := testConfig(t)
	ts.pathsToRemove = pathsToRemove
	sess, server := cziAws.NewMockSession()

	ts.server = server
	ts.conf = conf

	f, err := ioutil.TempFile("", "tokencache")
	a.Nil(err)
	defer f.Close()

	ts.pathsToRemove = append(ts.pathsToRemove, f.Name())

	username := "test_username"
	authKey := "my auth key"
	ttl := time.Hour

	awsClient := cziAws.New(sess)
	awsClient, ts.mockKMS = awsClient.WithMockKMS()
	awsClient, ts.mockLambda = awsClient.WithMockLambda()

	authContext := &kmsauth.AuthContextV2{
		To:       "a",
		From:     "b",
		UserType: "user",
	}

	tg := kmsauth.NewTokenGenerator(
		authKey,
		kmsauth.TokenVersion2,
		ttl,
		nil,
		authContext,
		awsClient,
	)

	// some default values
	encryptedStr := "super duper secret"
	keyID := "someKEUasfd"
	encryptOut := &kms.EncryptOutput{
		CiphertextBlob: []byte(encryptedStr),
		KeyId:          &keyID,
	}
	lambdaResponse := &bless.LambdaResponse{
		Certificate:  aws.String("my new cert"),
		ErrorType:    nil,
		ErrorMessage: nil,
	}
	lambdaBytes, err := json.Marshal(lambdaResponse)
	a.Nil(err)
	lambdaExecuteOut := &lambda.InvokeOutput{
		Payload: lambdaBytes,
	}

	ts.encryptOut = encryptOut
	ts.lambdaExecuteOut = lambdaExecuteOut

	ts.client = bless.New(conf).WithUsername(username).WithAwsClient(awsClient).WithTokenGenerator(tg)
}

func (ts *TestSuite) TestEverythingOk() {
	t := ts.T()
	a := assert.New(t)

	ts.mockKMS.On("EncryptWithContext", mock.Anything).Return(ts.encryptOut, nil)
	ts.mockLambda.On("InvokeWithContext", mock.Anything).Return(ts.lambdaExecuteOut, nil)

	err := ts.client.RequestCert(ts.ctx)
	a.Nil(err)
}

// If we can't parse the cert on disk - error
func (ts *TestSuite) TestErrOnMalformedCert() {
	t := ts.T()
	a := assert.New(t)

	certPath := fmt.Sprintf("%s-cert.pub", ts.conf.ClientConfig.SSHPrivateKey)
	err := ioutil.WriteFile(certPath, []byte("bad cert"), 0644)
	a.Nil(err)
	defer os.RemoveAll(certPath)

	err = ts.client.RequestCert(ts.ctx)
	a.NotNil(err)
	a.Contains(err.Error(), "Could not parse cert")
}

// If we already have a fresh cert don't request one
func (ts *TestSuite) TestFreshCert() {
	t := ts.T()
	a := assert.New(t)
	// cert generated as follows:
	// ssh-keygen -t rsa -f test_key
	// ssh-keygen -s test_key -I test-cert  -O critical:source-address:0.0.0.0/0 -n test-principal -V -520w:-510w test_key.pub
	ts.mockKMS.On("EncryptWithContext", mock.Anything).Return(ts.encryptOut, nil)
	ts.mockLambda.On("InvokeWithContext", mock.Anything).Return(ts.lambdaExecuteOut, nil)
	certPath := fmt.Sprintf("%s-cert.pub", ts.conf.ClientConfig.SSHPrivateKey)
	cert, err := ioutil.ReadFile("testdata/cert")
	a.Nil(err)
	err = ioutil.WriteFile(certPath, cert, 0644)
	a.Nil(err)
	defer os.RemoveAll(certPath)
	err = ts.client.RequestCert(ts.ctx)
	a.Nil(err)
	a.True(ts.mockLambda.Mock.AssertNotCalled(t, "InvokeWithContext"))
}

func (ts *TestSuite) TestBadPrincipalsCert() {
	t := ts.T()
	a := assert.New(t)
	// cert generated as follows:
	// ssh-keygen -t rsa -f test_key
	// ssh-keygen -s test_key -I test-cert  -O critical:source-address:0.0.0.0/0 -n test-principal -V -520w:-510w test_key.pub
	ts.mockKMS.On("EncryptWithContext", mock.Anything).Return(ts.encryptOut, nil)
	ts.mockLambda.On("InvokeWithContext", mock.Anything).Return(ts.lambdaExecuteOut, nil)
	certPath := fmt.Sprintf("%s-cert.pub", ts.conf.ClientConfig.SSHPrivateKey)
	cert, err := ioutil.ReadFile("testdata/bad-principal")
	a.Nil(err)
	err = ioutil.WriteFile(certPath, cert, 0644)
	a.Nil(err)
	defer os.RemoveAll(certPath)
	err = ts.client.RequestCert(ts.ctx)
	a.Nil(err)
	a.True(ts.mockLambda.Mock.AssertCalled(t, "InvokeWithContext", mock.Anything))
}

func (ts *TestSuite) TestBadCriticalOptionsCert() {
	t := ts.T()
	a := assert.New(t)
	// cert generated as follows:
	// ssh-keygen -t rsa -f test_key
	// ssh-keygen -s test_key -I test-cert  -O critical:source-address:0.0.0.0/0 -n test-principal -V -520w:-510w test_key.pub
	ts.mockKMS.On("EncryptWithContext", mock.Anything).Return(ts.encryptOut, nil)
	ts.mockLambda.On("InvokeWithContext", mock.Anything).Return(ts.lambdaExecuteOut, nil)
	certPath := fmt.Sprintf("%s-cert.pub", ts.conf.ClientConfig.SSHPrivateKey)
	cert, err := ioutil.ReadFile("testdata/bad-critical-options")
	a.Nil(err)
	err = ioutil.WriteFile(certPath, cert, 0644)
	a.Nil(err)
	defer os.RemoveAll(certPath)
	err = ts.client.RequestCert(ts.ctx)
	a.Nil(err)
	a.True(ts.mockLambda.Mock.AssertCalled(t, "InvokeWithContext", mock.Anything))
}

func (ts *TestSuite) TestReportsLambdaErrors() {
	t := ts.T()
	a := assert.New(t)

	lambdaResponse := &bless.LambdaResponse{
		Certificate:  aws.String("my new cert"),
		ErrorType:    aws.String("rando error"),
		ErrorMessage: aws.String("rando error message"),
	}
	lambdaBytes, err := json.Marshal(lambdaResponse)
	a.Nil(err)
	ts.lambdaExecuteOut = &lambda.InvokeOutput{
		Payload: lambdaBytes,
	}

	ts.mockKMS.On("EncryptWithContext", mock.Anything).Return(ts.encryptOut, nil)
	ts.mockLambda.On("InvokeWithContext", mock.Anything).Return(ts.lambdaExecuteOut, nil)

	err = ts.client.RequestCert(ts.ctx)
	a.NotNil(err)
	a.Contains(err.Error(), "bless error")
	a.Contains(err.Error(), *lambdaResponse.ErrorMessage)
	a.Contains(err.Error(), *lambdaResponse.ErrorType)
}

func (ts *TestSuite) TestNoCertificateInResponse() {
	t := ts.T()
	a := assert.New(t)

	lambdaResponse := &bless.LambdaResponse{
		Certificate:  nil,
		ErrorType:    nil,
		ErrorMessage: nil,
	}
	lambdaBytes, err := json.Marshal(lambdaResponse)
	a.Nil(err)
	ts.lambdaExecuteOut = &lambda.InvokeOutput{
		Payload: lambdaBytes,
	}

	ts.mockKMS.On("EncryptWithContext", mock.Anything).Return(ts.encryptOut, nil)
	ts.mockLambda.On("InvokeWithContext", mock.Anything).Return(ts.lambdaExecuteOut, nil)

	err = ts.client.RequestCert(ts.ctx)
	a.NotNil(err)
	a.Equal(err, errs.ErrNoCertificateInResponse)
}
func TestBlessClientSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// helpers---------
func rmPaths(paths []string) {
	for _, path := range paths {
		// best-effort
		os.RemoveAll(path)
	}
}

func testConfig(t *testing.T) (*config.Config, []string) {
	a := assert.New(t)

	pathsToRemove := []string{}

	// Create a temporary dir to write stuff to
	dirName, err := ioutil.TempDir("", "blessclient-test")
	a.Nil(err)
	pathsToRemove = append(pathsToRemove, dirName)

	// Create dummy ssh key
	f, err := ioutil.TempFile("", "blessclient-dummy-key")
	a.Nil(err)
	defer f.Close()

	// Create the public key
	pubKeyPath := fmt.Sprintf("%s.pub", f.Name())
	certPath := fmt.Sprintf("%s-cert.pub", f.Name())
	err = ioutil.WriteFile(pubKeyPath, []byte("public key"), 0644)
	a.Nil(err)

	conf := &config.Config{
		ClientConfig: config.ClientConfig{
			ConfigFile: path.Join(dirName, "config.yml"),
		},
		LambdaConfig: config.LambdaConfig{},
	}
	conf.ClientConfig.SSHPrivateKey = f.Name()
	conf.ClientConfig.RemoteUsers = []string{"test-principal"}

	pathsToRemove = append(pathsToRemove, f.Name(), pubKeyPath, certPath)
	return conf, pathsToRemove
}
