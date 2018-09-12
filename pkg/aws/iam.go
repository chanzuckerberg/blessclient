package aws

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/pkg/errors"
)

// IAM is an IAM client
type IAM struct {
	Svc iamiface.IAMAPI
}

// NewIAM returns a IAM client
func NewIAM(s *session.Session) *IAM {
	return &IAM{Svc: iam.New(s)}
}

//GetUsername gets the username for this aws user
func (i *IAM) GetUsername() (string, error) {
	output, err := i.Svc.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", errors.Wrap(err, `
		Can't get your user information from AWS.
		Either you don't have your AWS User credentials set up as [default] in ~/.aws/credentials
		or something else is setting AWS credentials.
		`)
	}
	if output == nil || output.User == nil || output.User.UserName == nil || output.User.Arn == nil {
		return "", errors.New("Nil output returned from aws.iam.get_user")
	}
	return *output.User.UserName, nil
}
