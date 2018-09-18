package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/pkg/errors"
)

// IAM is an iam service
type IAM struct {
	Svc iamiface.IAMAPI
}

// NewIAM returns a new IAM svc
func NewIAM(c client.ConfigProvider, config *aws.Config) *IAM {
	return &IAM{Svc: iam.New(c, config)}
}

// GetCurrentUser describes the calling user
func (i *IAM) GetCurrentUser() (*iam.User, error) {
	return i.GetUser(nil)
}

// GetUser returns the caller aws user
func (i *IAM) GetUser(username *string) (*iam.User, error) {
	output, err := i.Svc.GetUser(&iam.GetUserInput{UserName: username})
	if err != nil {
		return nil, errors.Wrap(err, "Can't get your user information from AWS.")
	}
	if output == nil {
		return nil, errors.New("Nil output returned from aws.iam.get_user")
	}
	return output.User, nil
}

// GetMFASerials gets the mfaSerials for the username
func (i *IAM) GetMFASerials(username *string) ([]string, error) {
	input := &iam.ListMFADevicesInput{
		UserName: username,
	}
	serialNumbers := []string{}
	err := i.Svc.ListMFADevicesPages(input, func(output *iam.ListMFADevicesOutput, lastPage bool) bool {
		if output == nil {
			return true
		}
		// We found some MFA devices
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
			return nil, errors.Wrap(err, "Access denied when listing MFA devices")
		}
		return nil, errors.Wrap(err, "Error fetching MFA devices")
	}
	return serialNumbers, nil
}

// GetAnMFASerial returns the first MFA serial on the user, errors if no MFA found
func (i *IAM) GetAnMFASerial(username *string) (string, error) {
	serials, err := i.GetMFASerials(username)
	if err != nil {
		return "", err
	}
	if len(serials) < 1 {
		return "", errors.New("No MFA serial Configured")
	}
	return serials[0], nil
}
