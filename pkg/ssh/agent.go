package ssh

import (
	"net"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/agent"
)

// Get an SSH agent
func GetSSHAgent(authSock string) (agent.ExtendedAgent, error) {
	agentConn, err := net.Dial("unix", authSock)
	if err != nil {
		return nil, errors.Wrapf(err, "could not dial %s", authSock)
	}
	defer agentConn.Close()

	return agent.NewClient(agentConn), nil
}
