package ssh_test

import (
	"testing"

	czissh "github.com/chanzuckerberg/blessclient/pkg/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
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
}

// setup
func (ts *TestSuite) SetupTest() {
}

// tests
// -----
func (ts *TestSuite) TestRSAKey() {
	t := ts.T()
	a := assert.New(t)

	s, err := czissh.NewSSH(rsaPrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)

	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)
}

func (ts *TestSuite) TestECDSAKey() {
	t := ts.T()
	a := assert.New(t)
	s, err := czissh.NewSSH(ecdsaPrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)
	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)
}

func (ts *TestSuite) TestED25519AKey() {
	t := ts.T()
	a := assert.New(t)
	s, err := czissh.NewSSH(ed25519PrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)
	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)
}

func (ts *TestSuite) TestSSHVersion() {
	t := ts.T()
	a := assert.New(t)
	s, err := czissh.GetSSHVersion()
	a.Nil(err)
	a.NotEmpty(s)
}

func TestSSHSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
