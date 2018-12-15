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
	"github.com/aws/aws-sdk-go/service/sts"
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
	mockSTS   *cziAws.MockSTSSvc

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
	_, ts.mockSTS = ts.awsClient.WithMockSTS()
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

func (ts *TestSuite) TestGetAWSUsernameOktaConfig() {
	t := ts.T()
	a := assert.New(t)

	output := &sts.GetCallerIdentityOutput{}
	output.SetUserId("role_id:test_user")
	ts.mockSTS.On("GetCallerIdentityWithContext", mock.Anything).Return(output, nil)
	c, err := config.DefaultConfig()
	mfaDevice := "phone1"
	c.OktaConfig = &config.OktaConfig{
		Profile:   "testprofile",
		MFADevice: &mfaDevice,
	}
	a.Nil(err)

	username, err := c.GetAWSUsername(ts.ctx, ts.awsClient)
	a.Nil(err)
	a.Equal(username, "test_user")
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

func (ts *TestSuite) TestUpdateAWSUsernameError() {
	t := ts.T()
	a := assert.New(t)
	e := fmt.Errorf("SOME ERROR")
	output := &iam.GetUserOutput{
		User: &iam.User{UserName: aws.String("testo")},
	}
	ts.mockIAM.On("GetUserWithContext", mock.Anything).Return(output, e)
	c, err := config.DefaultConfig()
	a.Nil(err)
	err = c.SetAWSUsernameIfMissing(ts.ctx, ts.awsClient)
	a.NotNil(err)
	a.Contains(err.Error(), e.Error())
}

func (ts *TestSuite) TestUpdateAWSUsernameEmptyResponse() {
	t := ts.T()
	a := assert.New(t)
	output := &iam.GetUserOutput{
		User: &iam.User{UserName: nil},
	}
	ts.mockIAM.On("GetUserWithContext", mock.Anything).Return(output, nil)
	c, err := config.DefaultConfig()
	a.Nil(err)
	err = c.SetAWSUsernameIfMissing(ts.ctx, ts.awsClient)
	a.NotNil(err)
	a.Contains(err.Error(), "AWS returned nil user")

	output.User = nil
	err = c.SetAWSUsernameIfMissing(ts.ctx, ts.awsClient)
	a.NotNil(err)
	a.Contains(err.Error(), "AWS returned nil user")
}

func (ts *TestSuite) TestGetRemoteUsers() {
	t := ts.T()
	a := assert.New(t)

	c, err := config.DefaultConfig()
	a.Nil(err)

	remoteUsers := c.GetRemoteUsers(ts.ctx, "testusername")
	a.Equal([]string{"testusername"}, remoteUsers)
}

func (ts *TestSuite) TestGetOktaMFADevice() {
	t := ts.T()
	a := assert.New(t)

	c, err := config.DefaultConfig()
	a.Nil(err)
	c.OktaConfig = &config.OktaConfig{
		Profile: "testprofile",
	}
	mfaDevice := c.GetOktaMFADevice()
	a.Equal(mfaDevice, "phone1")

	configMfaDevice := "u2f"
	c.OktaConfig = &config.OktaConfig{
		Profile:   "testprofile",
		MFADevice: &configMfaDevice,
	}
	mfaDevice = c.GetOktaMFADevice()
	a.Equal(mfaDevice, "u2f")
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
