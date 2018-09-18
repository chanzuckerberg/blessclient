package aws

import (
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/stretchr/testify/mock"
)

// MockLambdaSvc mocks the lambda service
type MockLambdaSvc struct {
	lambdaiface.LambdaAPI
	mock.Mock
}

// NewMockLambda returns a mock of the lambda service
func NewMockLambda() *MockLambdaSvc {
	return &MockLambdaSvc{}
}

// Invoke mocks invoke
func (l *MockLambdaSvc) Invoke(in *lambda.InvokeInput) (*lambda.InvokeOutput, error) {
	args := l.Called(in)
	out := args.Get(0).(*lambda.InvokeOutput)
	return out, args.Error(1)
}
