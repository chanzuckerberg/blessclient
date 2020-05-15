package config_test

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziAws "github.com/chanzuckerberg/go-misc/aws"
	cziAWSMocks "github.com/chanzuckerberg/go-misc/aws/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
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
	r := require.New(t)

	tmpFile, err := ioutil.TempFile("", "tmpConfig")
	r.Nil(err)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	c1 := config.DefaultConfig()
	r.Nil(err)

	bytes, err := yaml.Marshal(c1)
	r.Nil(err)
	_, err = tmpFile.Write(bytes)
	r.Nil(err)

	c2, err := config.FromFile(tmpFile.Name())
	r.Nil(err)

	r.Equal(c1, c2)
}

func (ts *TestSuite) TestPersist() {
	t := ts.T()
	r := require.New(t)

	tmpFile, err := ioutil.TempFile("", "tmpConfig")
	r.Nil(err)
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	c1 := config.DefaultConfig()
	r.Nil(err)
	err = c1.Persist(tmpFile.Name())
	r.Nil(err)
}

func TestFromFileMissing(t *testing.T) {
	t.Parallel()
	r := require.New(t)
	c, err := config.FromFile("notfoundnotfoundnotfound")
	r.NotNil(err)
	r.Contains(err.Error(), "could not read config at notfoundnotfoundnotfound")
	r.Nil(c)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
