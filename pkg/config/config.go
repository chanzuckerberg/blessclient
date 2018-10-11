package config

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/blessclient/pkg/telemetry"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	beeline "github.com/honeycombio/beeline-go"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultConfigFile is the default file where blessclient will look for its config
	DefaultConfigFile = "~/.blessclient/config.yml"
	// defaultCacheDir is a default cache dir
	defaultCacheDir = "cache"
	// defaultKMSAuthCache is the default kmsauth cache
	defaultKMSAuthCache = "kmsauth"
	// defaultAWSSessionCache is the default aws session cache
	defaultAWSSessionCache = "session"
	// DefaultSSHPrivateKey is a path to where users usually keep an ssh key
	DefaultSSHPrivateKey = "~/.ssh/id_rsa"
)

// Config is a blessclient config
type Config struct {
	// Version versions this config
	Version int `yaml:"version"`

	// ClientConfig is config for blessclient
	ClientConfig ClientConfig `yaml:"client_config"`
	// LambdaConfig holds configuration around the bless lambda
	LambdaConfig LambdaConfig `yaml:"lambda_config"`
	// For convenience, you can bundle an ~/.ssh/config template here
	SSHConfig *SSHConfig `yaml:"ssh_config,omitempty"`

	// Telemetry does telemetry
	Telemetry Telemetry `yaml:"telemetry,omitempty"`
}

// Region is an aws region that contains an aws lambda
type Region struct {
	// name of the aws region (us-west-2)
	AWSRegion string `yaml:"aws_region"`
	// region specific kms key id (not arn) of the key used for kmsauth
	KMSAuthKeyID string `yaml:"kms_auth_key_id"`
}

// ClientConfig is the client config
type ClientConfig struct {
	// ConfigFile is the path to blessclient config file
	ConfigFile string

	// AWSUserProfile is an aws profile that references a user (not a role)
	// leaving this empty typically means use `default` profile
	AWSUserProfile string ` yaml:"aws_user_profile"`
	// AWSUserName is your AWS username
	AWSUserName *string ` yaml:"aws_username,omitempty"`

	// Path to your ssh private key
	SSHPrivateKey string `yaml:"ssh_private_key"`

	// cert related
	CertLifetime Duration `yaml:"cert_lifetime,inline"`
	// ask bless to sign for these remote users
	RemoteUsers []string `yaml:"remote_users"`
	// bless calls these bastion ips - your source ip. 0.0.0.0/0 is all
	BastionIPS []string `yaml:"bastion_ips"`
}

// LambdaConfig is the lambda config
type LambdaConfig struct {
	// RoleARN used to assume and invoke bless lambda
	RoleARN string `yaml:"role_arn"`
	// Bless lambda function name
	FunctionName string `yaml:"function_name"`
	// bless lambda regions
	Regions []Region `yaml:"regions,omitempty"`
}

// Telemetry to track adoption, performance, errors
type Telemetry struct {
	Honeycomb *Honeycomb `yaml:"honeycomb,omitempty"`
}

// Honeycomb telemetry configuration
type Honeycomb struct {
	WriteKey string `yaml:"write_key,omitempty"`
	Dataset  string `yaml:"dataset,omitempty"`
	// SecretManagerARN is a secret that holds the honeycomb write key
	SecretManagerARN string `yaml:"secret_manager_arn,omitempty"`
}

// Duration is a wrapper around Duration to marshal/unmarshal
type Duration struct {
	time.Duration
}

// AsDuration returns as duration
func (d Duration) AsDuration() time.Duration {
	return d.Duration
}

// DefaultConfig generates a config with some defaults
func DefaultConfig() (*Config, error) {
	expandedDefaultConfigFile, err := homedir.Expand(DefaultConfigFile)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not expand %s", DefaultConfigFile)
	}

	expandedSSHPrivateKey, err := homedir.Expand(DefaultSSHPrivateKey)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not expand %s", DefaultSSHPrivateKey)
	}

	c := &Config{
		ClientConfig: ClientConfig{
			ConfigFile:    expandedDefaultConfigFile,
			CertLifetime:  Duration{30 * time.Minute},
			SSHPrivateKey: expandedSSHPrivateKey,
			RemoteUsers:   []string{},
			BastionIPS:    []string{},
		},
		LambdaConfig: LambdaConfig{},
	}
	return c, nil
}

// FromFile reads the config from file
func FromFile(file string) (*Config, error) {
	conf, err := DefaultConfig()
	b, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrapf(
				errs.ErrMissingConfig,
				"Missing config at %s, please run blessclient init to generate one",
				file)
		}
		return nil, errors.Wrapf(err, "Could not read config %s, you can generate one with bless init", file)
	}

	err = yaml.Unmarshal(b, conf)
	if err != nil {
		return nil, errors.WithMessage(err, "Invalid config, make sure it is valid yaml")
	}
	return conf, nil
}

// Persist persists a config to disk
func (c *Config) Persist() error {
	err := os.MkdirAll(c.getBlessclientDir(), 0755)
	if err != nil {
		return errors.Wrapf(err, "Could not create client config dir %s", c.getBlessclientDir())
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "Error marshaling config")
	}

	err = ioutil.WriteFile(c.ClientConfig.ConfigFile, b, 0644)
	if err != nil {
		return errors.Wrapf(err, "Could not write config to %s", c.ClientConfig.ConfigFile)
	}
	log.Infof("Config written to %s", c.ClientConfig.ConfigFile)
	return nil
}

func (c *Config) getBlessclientDir() string {
	return path.Dir(c.ClientConfig.ConfigFile)
}
func (c *Config) getCacheDir() string {
	return path.Join(c.getBlessclientDir(), defaultCacheDir, util.VersionCacheKey())
}

// GetKMSAuthCachePath gets a path to kmsauth cache file
// kmsauth is regional
func (c *Config) GetKMSAuthCachePath(region string) string {
	return path.Join(c.getCacheDir(), defaultKMSAuthCache, fmt.Sprintf("%s.json", region))
}

// GetAWSUsername gets the caller's aws username for kmsauth
func (c *Config) GetAWSUsername(ctx context.Context, awsClient *cziAWS.Client) (string, error) {
	ctx, span := beeline.StartSpan(ctx, "get_aws_username")
	defer span.Send()
	log.Debugf("Getting current aws iam user")
	if c.ClientConfig.AWSUserName != nil {
		log.Debugf("Using username %s from config", *c.ClientConfig.AWSUserName)
		span.AddField(telemetry.FieldIsCached, true)
		return *c.ClientConfig.AWSUserName, nil
	}
	user, err := awsClient.IAM.GetCurrentUser(ctx)
	if err != nil {
		span.AddField(telemetry.FieldError, err.Error())
		return "", err
	}
	if user == nil || user.UserName == nil {
		err = errors.New("AWS returned nil user")
		span.AddField(telemetry.FieldError, err.Error())
		return "", err
	}
	return *user.UserName, nil
}

// SetAWSUsernameIfMissing queries AWS for the username and sets it in the config if missing
func (c *Config) SetAWSUsernameIfMissing(ctx context.Context, awsClient *cziAWS.Client) error {
	username, err := c.GetAWSUsername(ctx, awsClient)
	if err != nil {
		return err
	}
	c.ClientConfig.AWSUserName = &username
	return nil
}
