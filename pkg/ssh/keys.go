package ssh

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/errs"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
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
	expandedPrivateKey, err := homedir.Expand(privateKey)
	if err != nil {
		return nil, errors.Errorf("Could not expand path %s", privateKey)
	}

	// Basic sanity check key is present
	// TODO maybe parse the file to make sure it is actually a private key
	_, err = os.Stat(expandedPrivateKey)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errs.ErrSSHKeyNotFound
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

// IsCertFresh determines if the cert is still fresh
func (s *SSH) IsCertFresh() (bool, error) {
	certBytes, err := s.ReadCert()
	// err reading cert
	if err != nil {
		return false, err
	}
	// no cert
	if certBytes == nil {
		return false, nil
	}

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
