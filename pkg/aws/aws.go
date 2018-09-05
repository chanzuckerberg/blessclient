package aws

import (
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
)

// Client is an AWS client
type Client struct {
	sess *session.Session
}

// NewClient returns a new aws client
func NewClient() (*Client, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		AssumeRoleTokenProvider: nil,
		SharedConfigState:       session.SharedConfigEnable,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Could not load aws session")
	}
	client := &Client{
		sess: sess,
	}

	return client, nil
}

// GUI for
func getMFA() (string, error) {
	if false {
		return stscreds.StdinTokenProvider()
	}
	return "", nil
}
