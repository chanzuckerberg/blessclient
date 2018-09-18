package aws

import (
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/stretchr/testify/mock"
)

// This is a mock for the STS service - mock more functions here as needed

// MockSTSSvc is a mock STS service
type MockSTSSvc struct {
	stsiface.STSAPI
	mock.Mock
}

// NewMockSTS returns a new mock sts svc
func NewMockSTS() *MockSTSSvc {
	return &MockSTSSvc{}
}

// GetSessionToken mocks GetSessionToken
func (s *MockSTSSvc) GetSessionToken(in *sts.GetSessionTokenInput) (*sts.GetSessionTokenOutput, error) {
	args := s.Called(in)
	out := args.Get(0).(*sts.GetSessionTokenOutput)
	return out, args.Error(1)
}
