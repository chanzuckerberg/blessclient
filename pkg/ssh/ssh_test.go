package ssh

import (
	"os/exec"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite

	loggerHook *test.Hook
}

const (
	// ssh-keygen -t rsa -b 4096 -C ""
	rsaPrivateKeyPath = "testdata/id_rsa"
	// ssh-keygen -t ecdsa -b 521 -C ""
	ecdsaPrivateKeyPath = "testdata/id_ecdsa"
	// ssh-keygen -t ed25519 -C ""
	ed25519PrivateKeyPath = "testdata/id_ed25519"
)

// cleanup
func (ts *TestSuite) TearDownTest() {
	sshVersionCmd = exec.Command("ssh", "-V")
	ts.loggerHook.Reset()
}

// setup
func (ts *TestSuite) SetupTest() {
}

// tests
// -----
func (ts *TestSuite) TestRSAKey() {
	t := ts.T()
	a := assert.New(t)

	s, err := NewSSH(rsaPrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)

	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)
}

func (ts *TestSuite) TestECDSAKey() {
	t := ts.T()
	a := assert.New(t)
	s, err := NewSSH(ecdsaPrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)
	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)
}

func (ts *TestSuite) TestED25519AKey() {
	t := ts.T()
	a := assert.New(t)
	s, err := NewSSH(ed25519PrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)
	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)
}

func (ts *TestSuite) TestEmptySSHPathError() {
	t := ts.T()
	a := assert.New(t)
	s, err := NewSSH("")
	a.NotNil(err)
	a.Equal("Must provide a non-empty path to the ssh private key", err.Error())
	a.Nil(s)
}

func (ts *TestSuite) TestCheckKeyTypeAndClientVersionDoesNotError() {
	t := ts.T()
	a := assert.New(t)

	s, err := NewSSH(rsaPrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)

	s.CheckKeyTypeAndClientVersion()
}

func (ts *TestSuite) TestSSHVersion() {
	t := ts.T()
	a := assert.New(t)
	sshVersionCmd = exec.Command("echo", "OpenSSH_7.6")
	v, err := GetSSHVersion()
	a.Nil(err)
	a.NotEmpty(v)
	a.Equal("OpenSSH_7.6\n", v)
}

func (ts *TestSuite) TestSSHVersionError() {
	t := ts.T()
	a := assert.New(t)
	sshVersionCmd = exec.Command("notfoundnotfoundnotfound")
	v, err := GetSSHVersion()
	a.NotNil(err)
	a.Empty(v)
}

func (ts *TestSuite) TestCheckVersionErrorLogError() {
	t := ts.T()
	a := assert.New(t)
	s, err := NewSSH(rsaPrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)
	sshVersionCmd = exec.Command("notfoundnotfoundnotfound")
	s.CheckKeyTypeAndClientVersion()
}

func TestSSHSuite(t *testing.T) {
	ts := &TestSuite{
		loggerHook: test.NewGlobal(),
	}
	suite.Run(t, ts)
}
