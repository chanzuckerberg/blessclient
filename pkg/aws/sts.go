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
func NewSTS(c client.ConfigProvider) *STS {
	return &STS{Svc: sts.New(c)}
}

// GetSTSToken gets an sts token
func (s *STS) GetSTSToken(mfaSerial string, mfaToken string) (*sts.Credentials, error) {
	duration := int64(64800) // 18 hours
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(duration),
		SerialNumber:    aws.String(mfaSerial),
		TokenCode:       aws.String(mfaToken),
	}
	output, err := s.Svc.GetSessionToken(input)
	if err != nil {
		return nil, errors.Wrap(err, "Could not request sts tokens")
	}

	return output.Credentials, nil
}
