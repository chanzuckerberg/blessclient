package main

import (
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	cziAWS "github.com/chanzuckerberg/blessclient/pkg/aws"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
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
			Config:                  aws.Config{},
		})
	if err != nil {
		return err
	}

	err = os.MkdirAll(config.ClientConfig.CacheDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "Could not create cache dir %s", config.ClientConfig.CacheDir)
	}

	mfaCache := path.Join(config.ClientConfig.CacheDir, config.ClientConfig.MFACacheFile)
	userTokenProvider := cziAWS.NewUserTokenProvider(sess, mfaCache)
	provider := credentials.NewCredentials(userTokenProvider)
	sess.Config.Credentials = provider

	client, err := bless.New(config, sess)
	if err != nil {
		return err
	}

	token, err := client.RequestKMSAuthToken()
	if err != nil {
		return err
	}
	log.Warnf("Got token %#v", token)
	return nil
}
