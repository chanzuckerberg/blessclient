package util_test

import (
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite

	cmd *cobra.Command
}

func (ts *TestSuite) SetupTest() {
	ts.cmd = &cobra.Command{}
}

func (ts *TestSuite) TearDownTest() {

}

func (ts *TestSuite) TestGetConfigPathMissing() {
	t := ts.T()
	a := assert.New(t)
	configPath, err := util.GetConfigPath(ts.cmd)
	a.Empty(configPath)
	a.NotNil(err)
	a.Contains(err.Error(), "Missing config")
}

func (ts *TestSuite) TestGetConfigPathNotExpandable() {
	t := ts.T()
	a := assert.New(t)
	ts.cmd.Flags().StringP("config", "c", "~invalidpath", "Use this to override the bless config file.")
	configPath, err := util.GetConfigPath(ts.cmd)
	a.Empty(configPath)
	a.NotNil(err)
	a.Contains(err.Error(), "cannot expand user-specific home dir")
}

func (ts *TestSuite) TestGetConfigPathOK() {
	t := ts.T()
	a := assert.New(t)
	ts.cmd.Flags().StringP("config", "c", "~", "Use this to override the bless config file.")
	_, err := util.GetConfigPath(ts.cmd)
	a.Nil(err)
}

func TestCobraSuite(t *testing.T) {
	suite.Run(t, &TestSuite{})
}
