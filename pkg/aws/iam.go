package aws

import (
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/pkg/errors"
)

// IAM is an IAM client
type IAM struct {
	Svc iamiface.IAMAPI
}

// NewIAM returns a IAM client
func NewIAM(c client.ConfigProvider) *IAM {
	return &IAM{Svc: iam.New(c)}
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

// GetMFASerial attempts to recover the aws mfa serial
func (i *IAM) GetMFASerial() (string, error) {
	input := &iam.ListMFADevicesInput{}
	serialNumbers := []string{}
	err := i.Svc.ListMFADevicesPages(input, func(output *iam.ListMFADevicesOutput, lastPage bool) bool {
		if output == nil {
			return true
		}
		// We foudn some MFA devices
		if len(output.MFADevices) > 0 {
			for _, mfaDevice := range output.MFADevices {
				if mfaDevice != nil && mfaDevice.SerialNumber != nil {
					serialNumbers = append(serialNumbers, *mfaDevice.SerialNumber)
				}
			}
		}
		return true
	})

	// Some more error checking
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == "AccessDenied" {
			return "", errors.Wrap(err, "Access denied when listing MFA devices")
		}
		return "", errors.Wrap(err, "Error fetching MFA devices")
	}
	if len(serialNumbers) == 0 {
		return "", errors.New("MFA not configured")
	}

	// Just pick one
	return serialNumbers[0], nil
}
