package aws

import (
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Client is an AWS client
type Client struct {
	sess   *session.Session
	Lambda *Lambda
	IAM    *IAM
}

// NewClient returns a new aws client
func NewClient(s *session.Session) *Client {
	client := &Client{
		sess:   s,
		Lambda: NewLambda(s, nil), // TODO: region
		IAM:    NewIAM(s),
	}
	return client
}

// TODO get this working
func getMFA() (string, error) {
	if false {
		return stscreds.StdinTokenProvider()
	}
	return "", nil
}
