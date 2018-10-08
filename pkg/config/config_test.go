package config_test

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziAws "github.com/chanzuckerberg/go-misc/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	yaml "gopkg.in/yaml.v2"
)

type TestSuite struct {
	suite.Suite

	ctx context.Context

	// aws
	awsClient *cziAws.Client
	mockIAM   *cziAws.MockIAMSvc

	// cleanup
	server *httptest.Server
}

func (ts *TestSuite) TearDownTest() {
	ts.server.Close()
}

func (ts *TestSuite) SetupTest() {
	// t := ts.T()
	// a := assert.New(t)
	ts.ctx = context.Background()

	sess, server := cziAws.NewMockSession()
	ts.server = server

	ts.awsClient = cziAws.New(sess)
	_, ts.mockIAM = ts.awsClient.WithMockIAM()
}

func (ts *TestSuite) TestFromFile() {
	t := ts.T()
	a := assert.New(t)

	tmpFile, err := ioutil.TempFile("", "tmpConfig")
	a.Nil(err)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	c1, err := config.DefaultConfig()
	a.Nil(err)

	bytes, err := yaml.Marshal(c1)
	a.Nil(err)
	_, err = tmpFile.Write(bytes)
	a.Nil(err)

	c2, err := config.FromFile(tmpFile.Name())
	a.Nil(err)

	a.Equal(c1, c2)
}

func (ts *TestSuite) TestPersist() {
	t := ts.T()
	a := assert.New(t)

	tmpFile, err := ioutil.TempFile("", "tmpConfig")
	a.Nil(err)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	c1, err := config.DefaultConfig()
	a.Nil(err)
	c1.ClientConfig.ConfigFile = tmpFile.Name()
	err = c1.Persist()
	a.Nil(err)

	c2, err := config.FromFile(tmpFile.Name())
	a.Nil(err)
	a.Equal(c1, c2)
}

func (ts *TestSuite) TestUpdateAWSUsername() {
	t := ts.T()
	a := assert.New(t)

	output := &iam.GetUserOutput{
		User: &iam.User{UserName: aws.String("testo")},
	}
	ts.mockIAM.On("GetUserWithContext", mock.Anything).Return(output, nil)
	c, err := config.DefaultConfig()
	a.Nil(err)

	err = c.SetAWSUsernameIfMissing(ts.ctx, ts.awsClient)
	a.Nil(err)
	ts.mockIAM.Mock.AssertNumberOfCalls(t, "GetUserWithContext", 1)

	err = c.SetAWSUsernameIfMissing(ts.ctx, ts.awsClient)
	a.Nil(err)
	// Should read the username from the config
	ts.mockIAM.Mock.AssertNumberOfCalls(t, "GetUserWithContext", 1)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
