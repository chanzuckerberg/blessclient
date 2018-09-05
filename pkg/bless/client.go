package client

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Client is a bless client
type Client struct{}

// LambdaPayload is the payload for the bless lambda
type LambdaPayload struct {
	BastionUser     string   `json:"bastion_user,omitempty"`
	RemoteUsernames []string `json:"remote_usernames,omitempty"`
	BastionIPs      []string `json:"bastion_ips,omitempty"`
	Command         string   `json:"command,omitempty"`
	PublicKeyToSign string   `json:"public_key_to_sign,omitempty"`
}

// RequestCert requests a cert
func (c *Client) RequestCert(payload *LambdaPayload) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "Could not json encode payload")
	}
	return nil
}
