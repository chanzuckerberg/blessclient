package aws

import (
	"github.com/aws/aws-sdk-go/aws/client"
)

// Client is an AWS client
type Client struct {
	Lambda *Lambda
	IAM    *IAM
	STS    *STS
}

// NewClient returns a new aws client
func NewClient(c client.ConfigProvider) *Client {
	client := &Client{
		// TODO: these need some work for multi-region failover
		Lambda: NewLambda(c, nil),
		IAM:    NewIAM(c),
		STS:    NewSTS(c),
	}
	return client
}
