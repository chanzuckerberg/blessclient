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
	timeSkew = time.Second * 30
)

// SSH is a namespace
type SSH struct {
	KeyName      string
	SSHDirectory string
}

// NewSSH returns a new SSH object
func NewSSH(keyName string, SSHDirectory *string) (*SSH, error) {
	ssh := &SSH{KeyName: keyName}
	if SSHDirectory == nil {
		dir, err := homedir.Dir()
		if err != nil {
			return nil, errors.Wrap(err, "Could not detect user's home directory")
		}
		ssh.SSHDirectory = path.Join(dir, ".ssh")
	} else {
		ssh.SSHDirectory = *SSHDirectory
	}
	return ssh, nil
}

// ReadPublicKey reads the SSH public key
func (s *SSH) ReadPublicKey() ([]byte, error) {
	pubKey := path.Join(s.SSHDirectory, fmt.Sprintf("%s.pub", s.KeyName))
	bytes, err := ioutil.ReadFile(pubKey)
	return bytes, errors.Wrap(err, "Could not read public key")
}

// ReadCert reads the ssh cert
func (s *SSH) ReadCert() ([]byte, error) {
	cert := path.Join(s.SSHDirectory, fmt.Sprintf("%s-cert.pub", s.KeyName))
	bytes, err := ioutil.ReadFile(cert)
	return bytes, errors.Wrap(err, "Could not read cert")
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

	now := time.Now()
	validBefore := time.Unix(int64(cert.ValidBefore), 0).Add(timeSkew)    // uper bound
	validAfter := time.Unix(int64(cert.ValidAfter), 0).Add(-1 * timeSkew) // lower bound

	isFresh := now.After(validAfter) && now.Before(validBefore)
	// TODO validation around principals and other cert things we might want
	return isFresh, nil
}
