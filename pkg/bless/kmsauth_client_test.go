package bless

import (
	"context"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
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
	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziAws "github.com/chanzuckerberg/go-misc/aws"
	cziAWSMocks "github.com/chanzuckerberg/go-misc/aws/mocks"
	"github.com/chanzuckerberg/go-misc/kmsauth"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type TestSuite struct {
	suite.Suite

	ctrl       *gomock.Controller
	mockKMS    *cziAWSMocks.MockKMSAPI
	mockLambda *cziAWSMocks.MockLambdaAPI
	client     *KMSAuthClient

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
	// assert mocks
	ts.ctrl.Finish()

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

	ts.ctrl = gomock.NewController(t)
	awsClient := cziAws.New(sess)
	awsClient, ts.mockKMS = awsClient.WithMockKMS(ts.ctrl)
	awsClient, ts.mockLambda = awsClient.WithMockLambda(ts.ctrl)

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
	lambdaResponse := &LambdaResponse{
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

	ts.client = NewKMSAuthClient(conf).WithUsername(username).WithAwsClient(awsClient).WithTokenGenerator(tg)
}

func (ts *TestSuite) TestEverythingOk() {
	t := ts.T()
	a := assert.New(t)

	ts.mockKMS.EXPECT().EncryptWithContext(gomock.Any(), gomock.Any()).Return(ts.encryptOut, nil)
	ts.mockLambda.EXPECT().InvokeWithContext(gomock.Any(), gomock.Any()).Return(ts.lambdaExecuteOut, nil)

	err := ts.client.RequestCert(ts.ctx)
	a.Nil(err)
}

type mockExtendedAgent struct {
	agent.ExtendedAgent
	Err error
}

func (m *mockExtendedAgent) Remove(ssh.PublicKey) error {
	return m.Err
}

func (ts *TestSuite) TestRemoveCertFromAgent() {
	t := ts.T()
	a := assert.New(t)

	mock := &mockExtendedAgent{}
	c := KMSAuthClient{}

	// rsa
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	a.NoError(err)

	// dsa
	params := &dsa.Parameters{}
	err = dsa.GenerateParameters(params, rand.Reader, dsa.L1024N160)
	a.NoError(err)
	dsaPrivatekey := &dsa.PrivateKey{}
	dsaPrivatekey.PublicKey.Parameters = *params
	err = dsa.GenerateKey(dsaPrivatekey, rand.Reader)
	a.NoError(err)

	// ecdsa
	pubkeyCurve := elliptic.P256()                                      //see http://golang.org/pkg/crypto/elliptic/#P256
	ecdsaPrivateKey, err := ecdsa.GenerateKey(pubkeyCurve, rand.Reader) // this generates a public & private key pair
	a.NoError(err)

	// ed25519
	_, ed25519PrivKey, err := ed25519.GenerateKey(rand.Reader)
	a.NoError(err)

	// no remove errors
	a.NoError(c.baseClient.removeKeyFromAgent(mock, rsaKey))
	a.NoError(c.baseClient.removeKeyFromAgent(mock, dsaPrivatekey))
	a.NoError(c.baseClient.removeKeyFromAgent(mock, ecdsaPrivateKey))
	a.NoError(c.baseClient.removeKeyFromAgent(mock, ed25519PrivKey))

	// errors
	err = c.baseClient.removeKeyFromAgent(mock, "not a key")
	a.Error(err)
}

func (ts *TestSuite) TestBadPrincipalsCert() {
	t := ts.T()
	a := assert.New(t)

	// cert generated as follows:
	// ssh-keygen -t rsa -f test_key
	// ssh-keygen -s test_key -I test-cert  -O critical:source-address:0.0.0.0/0 -n test-principal -V -520w:-510w test_key.pub
	ts.mockKMS.EXPECT().EncryptWithContext(gomock.Any(), gomock.Any()).Return(ts.encryptOut, nil)
	ts.mockLambda.EXPECT().InvokeWithContext(gomock.Any(), gomock.Any()).Return(ts.lambdaExecuteOut, nil)

	certPath := fmt.Sprintf("%s-cert.pub", ts.conf.ClientConfig.SSHPrivateKey)
	cert, err := ioutil.ReadFile("testdata/bad-principal")
	a.Nil(err)
	err = ioutil.WriteFile(certPath, cert, 0644)
	a.Nil(err)
	defer os.RemoveAll(certPath)
	err = ts.client.RequestCert(ts.ctx)
	a.Nil(err)
}

func (ts *TestSuite) TestBadCriticalOptionsCert() {
	t := ts.T()
	a := assert.New(t)
	// cert generated as follows:
	// ssh-keygen -t rsa -f test_key
	// ssh-keygen -s test_key -I test-cert  -O critical:source-address:0.0.0.0/0 -n test-principal -V -520w:-510w test_key.pub
	ts.mockKMS.EXPECT().EncryptWithContext(gomock.Any(), gomock.Any()).Return(ts.encryptOut, nil)
	ts.mockLambda.EXPECT().InvokeWithContext(gomock.Any(), gomock.Any()).Return(ts.lambdaExecuteOut, nil)
	certPath := fmt.Sprintf("%s-cert.pub", ts.conf.ClientConfig.SSHPrivateKey)
	cert, err := ioutil.ReadFile("testdata/bad-critical-options")
	a.Nil(err)
	err = ioutil.WriteFile(certPath, cert, 0644)
	a.Nil(err)
	defer os.RemoveAll(certPath)
	err = ts.client.RequestCert(ts.ctx)
	a.Nil(err)
}

func (ts *TestSuite) TestReportsLambdaErrors() {
	t := ts.T()
	a := assert.New(t)

	lambdaResponse := &LambdaResponse{
		Certificate:  aws.String("my new cert"),
		ErrorType:    aws.String("rando error"),
		ErrorMessage: aws.String("rando error message"),
	}
	lambdaBytes, err := json.Marshal(lambdaResponse)
	a.Nil(err)
	ts.lambdaExecuteOut = &lambda.InvokeOutput{
		Payload: lambdaBytes,
	}

	ts.mockKMS.EXPECT().EncryptWithContext(gomock.Any(), gomock.Any()).Return(ts.encryptOut, nil)
	ts.mockLambda.EXPECT().InvokeWithContext(gomock.Any(), gomock.Any()).Return(ts.lambdaExecuteOut, nil)

	err = ts.client.RequestCert(ts.ctx)
	a.NotNil(err)
	a.Contains(err.Error(), "bless error")
	a.Contains(err.Error(), *lambdaResponse.ErrorMessage)
	a.Contains(err.Error(), *lambdaResponse.ErrorType)
}

func (ts *TestSuite) TestNoCertificateInResponse() {
	t := ts.T()
	a := assert.New(t)

	lambdaResponse := &LambdaResponse{
		Certificate:  nil,
		ErrorType:    nil,
		ErrorMessage: nil,
	}
	lambdaBytes, err := json.Marshal(lambdaResponse)
	a.Nil(err)
	ts.lambdaExecuteOut = &lambda.InvokeOutput{
		Payload: lambdaBytes,
	}

	ts.mockKMS.EXPECT().EncryptWithContext(gomock.Any(), gomock.Any()).Return(ts.encryptOut, nil)
	ts.mockLambda.EXPECT().InvokeWithContext(gomock.Any(), gomock.Any()).Return(ts.lambdaExecuteOut, nil)

	err = ts.client.RequestCert(ts.ctx)
	a.NotNil(err)
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