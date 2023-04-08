package config_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziAws "github.com/chanzuckerberg/go-misc/aws"
	cziAWSMocks "github.com/chanzuckerberg/go-misc/aws/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	yaml "gopkg.in/yaml.v2"
)

type TestSuite struct {
	suite.Suite

	ctx context.Context

	ctrl *gomock.Controller

	// aws
	awsClient *cziAws.Client
	mockIAM   *cziAWSMocks.MockIAMAPI
	mockSTS   *cziAWSMocks.MockSTSAPI

	// cleanup
	server *httptest.Server
}

func (ts *TestSuite) TearDownTest() {
	ts.ctrl.Finish() // assert mocks
	ts.server.Close()
}

func (ts *TestSuite) SetupTest() {
	ts.ctx = context.Background()
	ts.ctrl = gomock.NewController(ts.T())

	sess, server := cziAws.NewMockSession()
	ts.server = server

	ts.awsClient = cziAws.New(sess)
	_, ts.mockIAM = ts.awsClient.WithMockIAM(ts.ctrl)
	_, ts.mockSTS = ts.awsClient.WithMockSTS(ts.ctrl)
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

	ts.mockIAM.EXPECT().GetUserWithContext(gomock.Any(), gomock.Any()).Return(output, nil).Times(1)
	c, err := config.DefaultConfig()
	a.Nil(err)

	err = c.SetAWSUsername(ts.ctx, ts.awsClient, nil)
	a.Nil(err)

	err = c.SetAWSUsername(ts.ctx, ts.awsClient, nil)
	a.Nil(err)
}

func (ts *TestSuite) TestUpdateAWSUsernameError() {
	t := ts.T()
	a := assert.New(t)
	e := fmt.Errorf("SOME ERROR")
	output := &iam.GetUserOutput{
		User: &iam.User{UserName: aws.String("testo")},
	}

	ts.mockIAM.EXPECT().GetUserWithContext(gomock.Any(), gomock.Any()).Return(output, e).Times(1)
	c, err := config.DefaultConfig()
	a.Nil(err)
	err = c.SetAWSUsername(ts.ctx, ts.awsClient, nil)
	a.NotNil(err)
	a.Contains(err.Error(), e.Error())
}

func (ts *TestSuite) TestUpdateAWSUsernameEmptyResponse() {
	t := ts.T()
	a := assert.New(t)
	output := &iam.GetUserOutput{
		User: &iam.User{UserName: nil},
	}
	ts.mockIAM.EXPECT().GetUserWithContext(gomock.Any(), gomock.Any()).Return(output, nil).Times(2)
	c, err := config.DefaultConfig()
	a.Nil(err)
	err = c.SetAWSUsername(ts.ctx, ts.awsClient, nil)
	a.NotNil(err)
	a.Contains(err.Error(), "AWS returned nil user")

	output.User = nil
	// does not make an aws calls
	err = c.SetAWSUsername(ts.ctx, ts.awsClient, nil)
	a.NotNil(err)
	a.Contains(err.Error(), "AWS returned nil user")
}

func (ts *TestSuite) TestGetRemoteUsers() {
	t := ts.T()
	a := assert.New(t)

	c, err := config.DefaultConfig()
	a.Nil(err)

	remoteUsers := c.GetRemoteUsers("testusername")
	a.Equal([]string{"testusername"}, remoteUsers)
}

func TestDuration(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	dur := config.Duration{Duration: time.Second}
	a.Equal(time.Second, dur.AsDuration())
}

func TestFromFileMissing(t *testing.T) {
	t.Parallel()
	a := assert.New(t)
	c, err := config.FromFile("notfoundnotfoundnotfound")
	a.NotNil(err)
	a.Contains(err.Error(), "Could not read config")
	a.Nil(c)
}

func TestGetCacheDir(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	c := &config.Config{
		ClientConfig: config.ClientConfig{
			ConfigFile: "/a/b/c.config",
		},
	}

	cachePath, err := c.GetKMSAuthCachePath("test-region")
	a.Nil(err)
	a.Equal("/a/b/cache/kmsauth/test-region.json", cachePath)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
