package aws

import (
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/stretchr/testify/mock"
)

// MockKMSSvc is a mock of the KMS service
type MockKMSSvc struct {
	kmsiface.KMSAPI
	mock.Mock
}

// NewMockKMS returns a new mock kms svc
func NewMockKMS() *MockKMSSvc {
	return &MockKMSSvc{}
}

// Encrypt mocks Encrypt
func (k *MockKMSSvc) Encrypt(in *kms.EncryptInput) (*kms.EncryptOutput, error) {
	args := k.Called(in)
	out := args.Get(0).(*kms.EncryptOutput)
	return out, args.Error(1)
}

// Decrypt decrypts
func (k *MockKMSSvc) Decrypt(in *kms.DecryptInput) (*kms.DecryptOutput, error) {
	args := k.Called(in)
	out := args.Get(0).(*kms.DecryptOutput)
	return out, args.Error(1)
}
