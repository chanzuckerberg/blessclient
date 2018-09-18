package aws

import (
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/stretchr/testify/mock"
)

// This is a mock for the IAM Svc - mock more functions here as needed

// MockIAMSvc is a mock of IAM service
type MockIAMSvc struct {
	iamiface.IAMAPI
	mock.Mock
}

// NewMockIAM returns a mock IAM SVC
func NewMockIAM() *MockIAMSvc {
	return &MockIAMSvc{}
}

// GetUser mocks getuser
func (i *MockIAMSvc) GetUser(in *iam.GetUserInput) (*iam.GetUserOutput, error) {
	args := i.Called(in)
	out := args.Get(0).(*iam.GetUserOutput)
	return out, args.Error(1)
}

// ListMFADevicesPages lists
func (i *MockIAMSvc) ListMFADevicesPages(in *iam.ListMFADevicesInput, fn func(*iam.ListMFADevicesOutput, bool) bool) error {
	args := i.Called(in)
	out := args.Get(0).(*iam.ListMFADevicesOutput)
	err := args.Error(1)
	if err != nil {
		return err
	}
	fn(out, true)
	return nil
}
