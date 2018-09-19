package bless_test

import (
	"testing"

	"github.com/chanzuckerberg/blessclient/pkg/bless"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/stretchr/testify/assert"
)

func TestBlessClient(t *testing.T) {
	a := assert.New(t)
	sess, serv := cziAWS.NewMockSession()
	defer serv.Close()

	// consts
	username := "asfasdf"
	kmsKEy := "asfqwer"

	awsClient := cziAWS.New(sess)
	conf := config.DefaultConfig()

	// tg := kmsauth.NewTokenGenerator()

	blessclient := bless.New(conf).WithAwsClient(awsClient).WithUsername(username)

}
