package ssh

import (
	"crypto"

	"golang.org/x/crypto/ssh"
)

type KeyManager interface {
	GetKey() (crypto.PublicKey, crypto.PrivateKey, error)
	WriteKey(crypto.PublicKey, crypto.PrivateKey, *ssh.Certificate) error
	HasValidCertificate() (bool, error)
	ListCertificates() ([]*ssh.Certificate, error)
}
