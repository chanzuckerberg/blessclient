package ssh

import (
	"net"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Agent struct {
	agent.ExtendedAgent

	conn net.Conn
}

// Get an SSH agent
func GetSSHAgent(authSock string) (*Agent, error) {
	agentConn, err := net.Dial("unix", authSock)
	if err != nil {
		return nil, errors.Wrapf(err, "could not dial %s", authSock)
	}

	return &Agent{
		ExtendedAgent: agent.NewClient(agentConn),

		conn: agentConn,
	}, nil
}

func (a *Agent) Close() error {
	return a.conn.Close()
}

func getLifetimeSecs(cert *ssh.Certificate) uint32 {
	certLifetime := int64(cert.ValidBefore) - time.Now().Unix()
	return uint32(certLifetime)
}
