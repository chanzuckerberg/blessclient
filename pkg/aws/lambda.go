package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/pkg/errors"
)

// Lambda is a Lambda client
type Lambda struct {
	Svc lambdaiface.LambdaAPI
}

// NewLambda returns a Lambda client
func NewLambda(s *session.Session, region *string) *Lambda {
	return &Lambda{Svc: lambda.New(s, &aws.Config{Region: region})}
}

// Execute executes the given function with the given payload and returns the output
func (l *Lambda) Execute(functionName string, payload []byte) ([]byte, error) {
	input := &lambda.InvokeInput{}
	input.
		SetPayload(payload).
		SetFunctionName(functionName).
		SetInvocationType(lambda.InvocationTypeRequestResponse).
		SetLogType(lambda.LogTypeTail)

	output, err := l.Svc.Invoke(input)
	if err != nil {
		return nil, errors.Wrapf(err, "Error invoking lambda function %s", functionName)
	}
	return output.Payload, nil
}
