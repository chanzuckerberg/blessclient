package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	bless "github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
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
			CacheDir:  "~/.blessclient",
			CacheFile: "cache.json",
		},
		Regions: []config.Region{
			{
				Name:         "shared-infra-prod-bless",
				AWSRegion:    "us-west-2",
				KMSAuthKeyID: "arn:aws:kms:us-west-2:416703108729:key/fe4c9d09-5006-4cb3-bb48-8b98476d3600",
			},
		},
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return err
	}

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
