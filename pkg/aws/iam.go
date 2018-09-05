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

//GetUser gets the username for this aws user
func (i *IAM) GetUser() (*iam.User, error) {
	output, err := i.Svc.GetUser(&iam.GetUserInput{})
	if err != nil {
		return nil, errors.Wrap(err, "Could not get user")
	}
	if output == nil || output.User == nil || output.User.UserName == nil || output.User.Arn == nil {
		return nil, errors.New("nil output returned from aws.iam.get_user")
	}
	return output.User, nil
}
