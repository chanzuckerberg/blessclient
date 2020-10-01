package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/stretchr/testify/mock"
)

// This is a mock for the SSM service - mock more functions here as needed

// MockSSMSvc is a mock SSM service
type MockSSMSvc struct {
	ssmiface.SSMAPI
	mock.Mock
}

// NewMockSSM returns a new mock SSM svc
func NewMockSSM() *MockSSMSvc {
	return &MockSSMSvc{}
}

func (s *MockSSMSvc) GetParameterWithContext(ctx aws.Context, input *ssm.GetParameterInput, opts ...request.Option) (*ssm.GetParameterOutput, error) {
	args := s.Called(input)
	out := args.Get(0).(*ssm.GetParameterOutput)
	return out, args.Error(1)
}

func (s *MockSSMSvc) PutParameterWithContext(ctx aws.Context, input *ssm.PutParameterInput, opts ...request.Option) (*ssm.PutParameterOutput, error) {
	args := s.Called(input)
	out := args.Get(0).(*ssm.PutParameterOutput)
	return out, args.Error(1)
}
