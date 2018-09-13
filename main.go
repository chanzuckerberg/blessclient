package main

import (
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	cziAWS "github.com/chanzuckerberg/blessclient/pkg/aws"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	err := exec()
	if err != nil {
		log.Fatal(err)
	}
}

func exec() error {
	config := &config.Config{
		ClientConfig: config.ClientConfig{
			CacheDir:         "~/.blessclient",
			MFACacheFile:     "mfa-cache.json",
			KMSAuthCacheFile: "kmsauth-cache.json",
		},
		Regions: []config.Region{
			{
				Name:         "shared-infra-prod-bless",
				AWSRegion:    "us-west-2",
				KMSAuthKeyID: "fe4c9d09-5006-4cb3-bb48-8b98476d3600",
			},
		},
	}

	sess, err := session.NewSessionWithOptions(
		session.Options{
			SharedConfigState:       session.SharedConfigEnable,
			AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
		})
	if err != nil {
		return err
	}

	cacheDir, err := homedir.Expand(config.ClientConfig.CacheDir)
	if err != nil {
		return errors.Wrapf(err, "Could not expand homedir in %s", cacheDir)
	}

	err = os.MkdirAll(config.ClientConfig.CacheDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "Could not create cache dir %s", cacheDir)
	}

	mfaCache := path.Join(cacheDir, config.ClientConfig.MFACacheFile)
	userTokenProvider := cziAWS.NewUserTokenProvider(sess, mfaCache)
	provider := credentials.NewCredentials(userTokenProvider)

	mfaAwsConfig := &aws.Config{
		Credentials: provider,
	}

	kmsAuthAWSClient, err := bless.New(config, sess, mfaAwsConfig)
	if err != nil {
		return err
	}

	token, err := kmsAuthAWSClient.RequestKMSAuthToken()
	if err != nil {
		return err
	}
	log.Warnf("Got token %#v", token)
	return nil
}
