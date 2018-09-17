package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Client is an AWS client
type Client struct {
	KMS KMS
}

// NewClient returns a new aws client
func NewClient(sess *session.Session, conf *aws.Config) *Client {
	return &Client{KMS: NewKMS(sess, conf)}
}
