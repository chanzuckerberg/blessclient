package ssh

import (
	"net"

	"github.com/pkg/errors"
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
