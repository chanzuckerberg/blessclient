package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/stretchr/testify/mock"
)

// This is a mock for the Secrets Manager service - mock more functions here as needed

// MockSecretsManagerSvc is a mock SecretsManager service
type MockSecretsManagerSvc struct {
	secretsmanageriface.SecretsManagerAPI
	mock.Mock
}

// NewMockSecretsManager returns a new mock SecretsManager svc
func NewMockSecretsManager() *MockSecretsManagerSvc {
	return &MockSecretsManagerSvc{}
}

func (c *MockSecretsManagerSvc) GetSecretValueWithContext(ctx aws.Context, input *secretsmanager.GetSecretValueInput, opts ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
	args := c.Called(input)
	out := args.Get(0).(*secretsmanager.GetSecretValueOutput)
	return out, args.Error(1)
}

func (c *MockSecretsManagerSvc) PutSecretValueWithContext(ctx aws.Context, input *secretsmanager.PutSecretValueInput, opts ...request.Option) (*secretsmanager.PutSecretValueOutput, error) {
	args := c.Called(input)
	out := args.Get(0).(*secretsmanager.PutSecretValueOutput)
	return out, args.Error(1)
}
