package ssh

import (
	"fmt"
	"io/ioutil"
	"path"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
)

const (
	timeDelta = time.Second * 30
)

// SSH is a namespace
type SSH struct {
	KeyName string
}

// NewSSH returns a new SSH
func NewSSH(keyName string) *SSH {
	return &SSH{KeyName: keyName}

}

func (s *SSH) sshDir() (string, error) {
	dir, err := homedir.Dir()
	return path.Join(dir, ".ssh"), errors.Wrap(err, "Could not detect user's home directory")
}

// ReadPublicKey reads the SSH public key
func (s *SSH) ReadPublicKey() ([]byte, error) {
	sshDir, err := s.sshDir()
	if err != nil {
		return nil, err
	}
	pubKey := path.Join(sshDir, fmt.Sprintf("%s.pub", s.KeyName))
	bytes, err := ioutil.ReadFile(pubKey)
	return bytes, errors.Wrap(err, "Could not read public key")
}

// ReadCert reads the ssh cert
func (s *SSH) ReadCert() ([]byte, error) {
	sshDir, err := s.sshDir()
	if err != nil {
		return nil, err
	}
	cert := path.Join(sshDir, fmt.Sprintf("%s-cert.pub", s.KeyName))
	return ioutil.ReadFile(cert)
}

// IsCertFresh determines if the cert is still fresh
func (s *SSH) IsCertFresh(certBytes []byte) (bool, error) {
	k, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		return false, errors.Wrap(err, "Could not parse cert")
	}
	cert, ok := k.(*ssh.Certificate)
	if !ok {
		return false, errors.New("Bytes do not correspond to an ssh certificate")
	}

	// Validate Cert
	// Validate Time
	validBefore := time.Unix(int64(cert.ValidBefore), 0).Add(timeDelta)
	validAfter := time.Unix(int64(cert.ValidAfter), 0).Add(-1 * timeDelta)
	now := time.Now()
	if now.After(validBefore) || now.Before(validAfter) {
		return false, nil
	}

	return true, nil
}
