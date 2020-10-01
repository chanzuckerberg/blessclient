package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"
)

// Firehose is a firehose service
type Firehose struct {
	Svc firehoseiface.FirehoseAPI
}

// NewFirehose returns a new firehose service
func NewFirehose(c client.ConfigProvider, config *aws.Config) *Firehose {
	return &Firehose{Svc: firehose.New(c, config)}
}
