package ssh

import (
	"reflect"
	"strings"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/config"
)

const (
	timeSkew = 10 * time.Second
)

// SSH is a namespace
type SSH struct {
	keyName      string
	sshDirectory string
}

// NewSSH returns a new SSH object
func NewSSH() (*SSH, error) {
	ssh := &SSH{}
	return ssh, nil
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

	// to protect against time-skew issues we potentially generate a certificate timeSkew duration
	//    earlier than we might've otherwise
	validBefore := time.Unix(int64(cert.ValidBefore), 0).Add(-1 * timeSkew) // upper bound
	isFresh := now.Before(validBefore)

	// TODO: add more validation for certificate critical options
	val, ok := cert.CriticalOptions["source-address"]
	isFresh = isFresh && ok && val == strings.Join(c.ClientConfig.BastionIPS, ",")
	isFresh = isFresh && (c.ClientConfig.SkipPrincipalValidation || reflect.DeepEqual(cert.ValidPrincipals, c.ClientConfig.RemoteUsers))

	return isFresh, nil
}
