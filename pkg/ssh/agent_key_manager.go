package ssh

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type AgentKeyManager struct {
	agent agent.ExtendedAgent
}

func NewAgentKeyManager(agent agent.ExtendedAgent) KeyManager {
	return &AgentKeyManager{
		agent: agent,
	}
}

// GetKey will generate new ssh keypair
func (a *AgentKeyManager) GetKey() (crypto.PublicKey, crypto.PrivateKey, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	return public, private, errors.Wrap(err, "could not generate ed25519 keys")
}

// WriteKey will write the key and certificate to the agent
func (a *AgentKeyManager) WriteKey(
	priv crypto.PrivateKey,
	cert *ssh.Certificate,
) error {
	err := a.agent.Add(agent.AddedKey{
		PrivateKey:   priv,
		Certificate:  cert,
		Comment:      a.getComment(cert),
		LifetimeSecs: getLifetimeSecs(cert),
	})
	return errors.Wrap(err, "could not add keys to agent")
}

func (a *AgentKeyManager) getComment(cert *ssh.Certificate) string {
	now := time.Now().Local().Format(time.UnixDate)
	return fmt.Sprintf("Added by blessclient at %s", now)
}

func (a *AgentKeyManager) ListCertificates() ([]*ssh.Certificate, error) {
	agentKeys, err := a.agent.List()
	if err != nil {
		return nil, errors.Wrap(err, "could not list agent keys")
	}

	allCerts := []*ssh.Certificate{}

	for _, agentKey := range agentKeys {
		pub, err := ssh.ParsePublicKey(agentKey.Marshal())
		if err != nil {
			logrus.Warnf("could not parse public key: %s", err.Error())
			continue
		}

		cert, ok := pub.(*ssh.Certificate)
		if !ok {
			continue
		}

		_, ok = cert.Extensions["ssh-ca-lambda"]
		if !ok {
			// not a certificate we care about
			continue
		}

		now := time.Now()
		validAfter := time.Unix(int64(cert.ValidAfter), 0)
		validBefore := time.Unix(int64(cert.ValidBefore), 0)

		if !(now.After(validAfter) && now.Before(validBefore)) {
			continue // expired
		}

		allCerts = append(allCerts, cert)
	}

	return allCerts, nil
}

func (a *AgentKeyManager) HasValidCertificate() (bool, error) {
	certs, err := a.ListCertificates()
	if err != nil {
		return false, err
	}

	return len(certs) > 0, nil
}
