package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
)

// Client is an AWS client
type Client struct {
	Lambda *Lambda
	IAM    *IAM
	STS    *STS
}

// NewClient returns a new aws client
func NewClient(
	c client.ConfigProvider,
	kmsauthConfig *aws.Config,
	lambdaConfig *aws.Config) *Client {
	client := &Client{
		Lambda: NewLambda(c, lambdaConfig),
		// kmsauth
		IAM: NewIAM(c, kmsauthConfig),
		STS: NewSTS(c, kmsauthConfig),
	}
	return client
}
