package cmd

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/telemetry"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/honeycombio/opencensus-exporter/honeycomb"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opencensus.io/trace"
)

func init() {
	rootCmd.AddCommand()
}

// server is a namespace
type server struct {
	client *http.Client

	// ec2 id doc
	document  []byte
	signature []byte
}

const (
	awsInstanceIdentityDocument          = "http://169.254.169.254/latest/dynamic/instance-identity/document"
	awsInstanceIdentityDocumentSignature = "http://169.254.169.254/latest/dynamic/instance-identity/signature"
)

var serverCmd = &cobra.Command{
	Use:           "server",
	Short:         "server requests a server certificate",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := uuid.NewUUID()
		if err != nil {
			// Just using for telemetry so ignore this error
			logrus.Debugf("Failed to generate UUID with error %s", err.Error())
		}
		logrus.Debugf("Running blessclient v%s", util.VersionCacheKey())
		logrus.Debugf("RunID: %s", id.String())
		ctx := context.Background()
		expandedConfigFile, err := util.GetConfigPath(cmd)
		if err != nil {
			return err
		}
		conf, err := config.FromFile(expandedConfigFile)
		if err != nil {
			return err
		}
		logrus.Debugf("Parsed config is: %s", spew.Sdump(conf))

		// tracing
		traceSampling := float64(1)
		trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(traceSampling)})
		if conf.Telemetry.Honeycomb != nil {
			honeycombExporter := honeycomb.NewExporter(conf.Telemetry.Honeycomb.WriteKey, conf.Telemetry.Honeycomb.Dataset)
			defer honeycombExporter.Close()
			honeycombExporter.ServiceName = "blessclient"
			honeycombExporter.SampleFraction = traceSampling
			trace.RegisterExporter(honeycombExporter)
		}
		ctx, span := trace.StartSpan(ctx, cmd.Use)
		span.AddAttributes(
			trace.StringAttribute(telemetry.FieldID, id.String()),
			trace.StringAttribute(telemetry.FieldBlessclientVersion, util.VersionCacheKey()),
			trace.StringAttribute(telemetry.FieldBlessclientGitSha, util.GitSha),
			trace.StringAttribute(telemetry.FieldBlessclientRelease, util.Release),
			trace.StringAttribute(telemetry.FieldBlessclientDirty, util.Dirty),
		)
		defer span.End()

		sess, err := session.NewSessionWithOptions(
			session.Options{
				SharedConfigState:       session.SharedConfigEnable,
				AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
				Profile:                 conf.ClientConfig.AWSUserProfile,
			},
		)
		if err != nil {
			span.AddAttributes(trace.StringAttribute(telemetry.FieldError, err.Error()))
			return errors.Wrap(err, "Could not create aws session")
		}
		s := &server{
			client: &http.Client{
				Timeout: time.Second * 10,
			},
		}

		err = s.getInstanceIdentityDocument(ctx)
		if err != nil {
			return err
		}

		return nil
	},
}

func (s *server) processRegion(ctx context.Context, conf *config.Config, sess *session.Session, region config.Region) error {
	ctx, span := trace.StartSpan(ctx, "process_region")
	defer span.End()
	span.AddAttributes(trace.StringAttribute(telemetry.FieldRegion, region.AWSRegion))
	awsClient := getAWSClient(ctx, conf, sess, region)
}

func (s *server) getCert(ctx context.Context, conf *config.Config, awsCient *cziAWS.Client, region config.Region) error {
	ctx, span := trace.StartSpan(ctx, "get_cert")
	defer span.End()

}

func (s *server) getInstanceIdentityDocument(ctx context.Context) error {
	document, err := s.fetch(ctx, awsInstanceIdentityDocument)
	if err != nil {
		return err
	}
	s.document = document

	signature, err := s.fetch(ctx, awsInstanceIdentityDocumentSignature)
	s.signature = signature
	return err
}

func (s *server) fetch(ctx context.Context, url string) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "fetch_url")
	defer span.End()
	span.AddAttributes(trace.StringAttribute(telemetry.FieldURL, url))
	rsp, err := s.client.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not fetch %s", url)
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	return body, errors.Wrap(err, "Could not read body")
}
