package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

// SSM is an ssm service
type SSM struct {
	Svc ssmiface.SSMAPI
}

// NewSSM returns a new SSM svc
func NewSSM(c client.ConfigProvider, config *aws.Config) *SSM {
	return &SSM{Svc: ssm.New(c, config)}
}
