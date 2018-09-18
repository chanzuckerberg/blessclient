package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/pkg/errors"
)

// STS is an STS client
type STS struct {
	Svc stsiface.STSAPI
}

// NewSTS returns an sts client
func NewSTS(c client.ConfigProvider, config *aws.Config) *STS {
	return &STS{Svc: sts.New(c, config)}
}

// GetSTSToken gets an sts token
func (s *STS) GetSTSToken(input *sts.GetSessionTokenInput) (*sts.Credentials, error) {
	output, err := s.Svc.GetSessionToken(input)
	if err != nil {
		return nil, errors.Wrap(err, "Could not request sts tokens")
	}
	if output == nil {
		return nil, errors.New("Nil output from aws")
	}
	return output.Credentials, nil
}
