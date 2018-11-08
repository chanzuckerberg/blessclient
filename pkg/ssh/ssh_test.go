package ssh_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	czissh "github.com/chanzuckerberg/blessclient/pkg/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ssh"
)

type TestSuite struct {
	suite.Suite

	rsaPubKeyPath  string
	rsaPrivKeyPath string

	ecdsaPubKeyPath  string
	ecdsaPrivKeyPath string
}

// cleanup
func (ts *TestSuite) TearDownTest() {
	os.Remove(ts.rsaPrivKeyPath)
	os.Remove(ts.rsaPubKeyPath)

	os.Remove(ts.ecdsaPrivKeyPath)
	os.Remove(ts.ecdsaPubKeyPath)
}

// setup
func (ts *TestSuite) SetupTest() {
	ts.generateRSAKey()
	ts.generateECDSAKey()
}

// tests
// -----
func (ts *TestSuite) TestRSAKey() {
	t := ts.T()
	a := assert.New(t)

	s, err := czissh.NewSSH(ts.rsaPrivKeyPath)
	a.Nil(err)
	a.NotNil(s)

	_, err = s.ReadAndParsePublicKey()
	a.Nil(err)
	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)

}

func (ts *TestSuite) TestECDSAKey() {
	t := ts.T()
	a := assert.New(t)

	s, err := czissh.NewSSH(ts.ecdsaPrivKeyPath)
	a.Nil(err)
	a.NotNil(s)

	_, err = s.ReadAndParsePublicKey()
	a.Nil(err)
	_, err = s.ReadAndParsePrivateKey()
	a.Nil(err)

}

func TestSSHSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// helpers
// -------

func (ts *TestSuite) generateRSAKey() {
	t := ts.T()
	a := assert.New(t)
	// RSA
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	a.Nil(err)

	rsaPrivFile, err := ioutil.TempFile("", "rsa")
	a.Nil(err)
	ts.rsaPrivKeyPath = rsaPrivFile.Name()
	defer rsaPrivFile.Close()

	privateKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	err = pem.Encode(rsaPrivFile, privateKey)
	a.Nil(err)

	sshPublicKey, err := ssh.NewPublicKey(&key.PublicKey)
	a.Nil(err)

	pubKeyFileName := fmt.Sprintf("%s.pub", rsaPrivFile.Name())
	ts.rsaPubKeyPath = pubKeyFileName
	rsaPubFile, err := os.Create(pubKeyFileName)
	a.Nil(err)
	defer rsaPubFile.Close()

	err = ioutil.WriteFile(pubKeyFileName, sshPublicKey.Marshal(), os.FileMode(0644))
	a.Nil(err)
}

func (ts *TestSuite) generateECDSAKey() {
	a := assert.New(ts.T())

	reader := rand.Reader
	key, err := ecdsa.GenerateKey(elliptic.P521(), reader)
	a.Nil(err)

	ecdsaPrivFile, err := ioutil.TempFile("", "ecdsa")
	a.Nil(err)
	ts.ecdsaPrivKeyPath = ecdsaPrivFile.Name()
	defer ecdsaPrivFile.Close()

	ecdsaPubFileName := fmt.Sprintf("%s.pub", ecdsaPrivFile.Name())
	ts.ecdsaPubKeyPath = ecdsaPubFileName

	ecdsaPubFile, err := os.Create(ecdsaPubFileName)
	a.Nil(err)
	defer ecdsaPubFile.Close()

	bytes, err := x509.MarshalECPrivateKey(key)
	a.Nil(err)

	privateKey := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: bytes,
	}
	err = pem.Encode(ecdsaPrivFile, privateKey)
	a.Nil(err)

	sshPublicKey, err := ssh.NewPublicKey(&key.PublicKey)
	a.Nil(err)

	err = ioutil.WriteFile(ecdsaPubFileName, sshPublicKey.Marshal(), os.FileMode(0644))
	a.Nil(err)
}
