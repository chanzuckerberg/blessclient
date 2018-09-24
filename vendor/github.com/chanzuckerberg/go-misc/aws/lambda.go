package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/pkg/errors"
)

// Lambda is a Lambda client
type Lambda struct {
	Svc lambdaiface.LambdaAPI
}

// NewLambda returns a Lambda client
func NewLambda(c client.ConfigProvider, config *aws.Config) *Lambda {
	return &Lambda{Svc: lambda.New(c, config)}
}

// Execute executes the given function with the given payload and returns the output
func (l *Lambda) Execute(ctx context.Context, functionName string, payload []byte) ([]byte, error) {
	input := &lambda.InvokeInput{}
	input.
		SetPayload(payload).
		SetFunctionName(functionName).
		SetInvocationType(lambda.InvocationTypeRequestResponse).
		SetLogType(lambda.LogTypeTail)

	output, err := l.Svc.InvokeWithContext(ctx, input)
	if err != nil {
		return nil, errors.Wrapf(err, "Error invoking lambda function %s", functionName)
	}
	return output.Payload, nil
}
