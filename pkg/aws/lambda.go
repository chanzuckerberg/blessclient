package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

// Lambda is a Lambda client
type Lambda struct {
	Svc lambdaiface.LambdaAPI
}

// NewLambda returns a Lambda client
func NewLambda(s *session.Session) *Lambda {
	return &Lambda{Svc: lambda.New(s, &aws.Config{})}
}
