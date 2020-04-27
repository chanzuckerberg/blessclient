package ssh

import (
	"os/exec"
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/config"
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
	// expired ed25519PrivateKeyPath
	// ssh-keygen -s ca -I testID -n testUser -V +1s -z 1 expired_id_ed25519.pub
	expiredED25519PrivateKeyPath = "testdata/expired_id_ed25519"
)

// HACK we're mocking out the ssh command
func resetSSHCommand() {
	sshVersionCmd = exec.Command("ssh", "-V")
}

// cleanup
func (ts *TestSuite) TearDownTest() {
	resetSSHCommand()
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

func (ts *TestSuite) TestNoCertPresent() {
	t := ts.T()
	a := assert.New(t)
	s, err := NewSSH(rsaPrivateKeyPath) // no cert for this key
	a.Nil(err)
	a.NotNil(s)
	// no error no cert
	cert, err := s.ReadAndParseCert()
	a.Nil(err)
	a.Nil(cert)
}

func (ts *TestSuite) TestExpiredCert() {
	t := ts.T()
	a := assert.New(t)
	resetSSHCommand()

	s, err := NewSSH(expiredED25519PrivateKeyPath) // no cert for this key
	a.Nil(err)
	a.NotNil(s)
	// no error no cert
	cert, err := s.ReadAndParseCert()
	a.Nil(err)
	a.NotNil(cert)
	a.Len(cert.ValidPrincipals, 1)
	a.Equal(cert.ValidPrincipals[0], "testUser")
	a.Equal(cert.Serial, uint64(1))
}

func (ts *TestSuite) TestIsCertFreshExpiredCert() {
	t := ts.T()
	a := assert.New(t)
	sshVersionCmd = exec.Command("echo", "OpenSSH_7.6")
	defer resetSSHCommand()
	s, err := NewSSH(expiredED25519PrivateKeyPath)
	a.Nil(err)
	a.NotNil(s)
	conf := &config.Config{
		ClientConfig: config.ClientConfig{
			BastionIPS:  []string{},
			RemoteUsers: []string{"testUser"},
		},
	}
	// no error no cert
	fresh, err := s.IsCertFresh(conf)
	a.Nil(err)
	a.False(fresh)
}

func (ts *TestSuite) TestIsCertFreshNoCert() {
	t := ts.T()
	a := assert.New(t)
	sshVersionCmd = exec.Command("echo", "OpenSSH_7.6")
	defer resetSSHCommand()
	s, err := NewSSH(rsaPrivateKeyPath) // no cert for this key
	a.Nil(err)
	a.NotNil(s)
	conf := &config.Config{
		ClientConfig: config.ClientConfig{
			BastionIPS:  []string{},
			RemoteUsers: []string{"testUser"},
		},
	}
	// no error no cert
	fresh, err := s.IsCertFresh(conf)
	a.Nil(err)
	a.False(fresh)
}

func (ts *TestSuite) TestKeyNotFound() {
	t := ts.T()
	a := assert.New(t)

	s, err := NewSSH("somekeythatdoesnotexist")
	a.NotNil(err)
	a.Nil(s)
	a.Contains(err.Error(), "Key somekeythatdoesnotexist not found")
}

func TestSSHSuite(t *testing.T) {
	ts := &TestSuite{
		loggerHook: test.NewGlobal(),
	}
	suite.Run(t, ts)
}
