package ssh

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	timeSkew = time.Second * 30
)

// SSH is a namespace
type SSH struct {
	keyName      string
	sshDirectory string
}

// NewSSH returns a new SSH object
func NewSSH(privateKey string) (*SSH, error) {
	if privateKey == "" {
		return nil, errors.New("Must provide a non-empty path to the ssh private key")
	}

	expandedPrivateKey, err := homedir.Expand(privateKey)
	if err != nil {
		return nil, errors.Errorf("Could not expand path %s", privateKey)
	}

	// Basic sanity check key is present
	// TODO maybe parse the file to make sure it is actually a private key
	_, err = os.Stat(expandedPrivateKey)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Key %s not found", expandedPrivateKey)
		}
		return nil, errors.Wrapf(err, "Could not stat key at %s", expandedPrivateKey)
	}

	ssh := &SSH{
		keyName:      path.Base(expandedPrivateKey),
		sshDirectory: path.Dir(expandedPrivateKey),
	}
	return ssh, nil
}

// ReadPublicKey reads the SSH public key
func (s *SSH) ReadPublicKey() ([]byte, error) {
	pubKey := path.Join(s.sshDirectory, fmt.Sprintf("%s.pub", s.keyName))
	bytes, err := ioutil.ReadFile(pubKey)
	return bytes, errors.Wrap(err, "Could not read public key")
}

// ReadAndParsePublicKey reads and unmarshals a public key
func (s *SSH) ReadAndParsePublicKey() (ssh.PublicKey, error) {
	bytes, err := s.ReadPublicKey()
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParsePublicKey(bytes)
	return key, errors.Wrap(err, "Could not parse public key")
}

// ReadPrivateKey reads the private key
func (s *SSH) ReadPrivateKey() ([]byte, error) {
	privKey := path.Join(s.sshDirectory, s.keyName)
	bytes, err := ioutil.ReadFile(privKey)
	return bytes, errors.Wrapf(err, "Could not read private key %s", privKey)
}

// ReadAndParsePrivateKey reads and unmarshals a private key
func (s *SSH) ReadAndParsePrivateKey() (interface{}, error) {
	bytes, err := s.ReadPrivateKey()
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParseRawPrivateKey(bytes)
	return key, errors.Wrap(err, "Could not parse private key")
}

// ReadCert reads the ssh cert
func (s *SSH) ReadCert() ([]byte, error) {
	cert := path.Join(s.sshDirectory, fmt.Sprintf("%s-cert.pub", s.keyName))
	bytes, err := ioutil.ReadFile(cert)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no cert
		}
		return nil, errors.Wrap(err, "Could not read cert")
	}
	return bytes, nil
}

// ReadAndParseCert reads a certificate off disk and attempts to unmarshal it
func (s *SSH) ReadAndParseCert() (*ssh.Certificate, error) {
	bytes, err := s.ReadCert()
	if err != nil {
		return nil, err
	}
	// no cert
	if bytes == nil {
		return nil, nil
	}
	k, _, _, _, err := ssh.ParseAuthorizedKey(bytes)
	if err != nil {
		return nil, errors.Wrap(err, "Could not parse cert")
	}
	cert, ok := k.(*ssh.Certificate)
	if !ok {
		return nil, errors.New("Bytes do not correspond to an ssh certificate")
	}
	return cert, nil
}

// IsCertFresh determines if the cert is still fresh
func (s *SSH) IsCertFresh(c *config.Config) (bool, error) {
	cert, err := s.ReadAndParseCert()
	if err != nil {
		return false, err
	}
	if cert == nil {
		return false, nil
	}

	now := time.Now()
	validBefore := time.Unix(int64(cert.ValidBefore), 0).Add(timeSkew)    // uper bound
	validAfter := time.Unix(int64(cert.ValidAfter), 0).Add(-1 * timeSkew) // lower bound

	isFresh := now.After(validAfter) && now.Before(validBefore)

	// TODO: add more validation for certificate critical options
	val, ok := cert.CriticalOptions["source-address"]
	isFresh = isFresh && ok && val == strings.Join(c.ClientConfig.BastionIPS, ",")
	// Compare principals
	isFresh = isFresh && reflect.DeepEqual(cert.ValidPrincipals, c.ClientConfig.RemoteUsers)

	return isFresh, nil
}

// WriteCert writes a cert to disk
func (s *SSH) WriteCert(b []byte) error {
	certPath := path.Join(s.sshDirectory, fmt.Sprintf("%s-cert.pub", s.keyName))
	log.Debugf("Writing cert to %s", certPath)
	err := ioutil.WriteFile(certPath, b, 0644)
	return errors.Wrapf(err, "Could not write cert to %s", certPath)
}
