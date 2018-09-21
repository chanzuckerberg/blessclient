package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/mitchellh/go-homedir"

	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/blessclient/pkg/util"
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
)

// Config is a blessclient config
type Config struct {
	// Version versions this config
	Version int `json:"version" yaml:"version"`

	// ClientConfig is config for blessclient
	ClientConfig ClientConfig `json:"client_config" yaml:"client_config"`
	// LambdaConfig holds configuration around the bless lambda
	LambdaConfig LambdaConfig `json:"lambda_config" yaml:"lambda_config"`
	// For convenience, you can bundle an ~/.ssh/config template here
	SSHConfig *SSHConfig `json:"ssh_config,omitempty" yaml:"ssh_config,omitempty"`
}

// Region is an aws region that contains an aws lambda
type Region struct {
	// name of the aws region (us-west-2)
	AWSRegion string `json:"aws_region" yaml:"aws_region"`
	// region specific kms key id (not arn) of the key used for kmsauth
	KMSAuthKeyID string `json:"kms_auth_key_id" yaml:"kms_auth_key_id"`
}

// ClientConfig is the client config
type ClientConfig struct {
	// ConfigFile is the path to blessclient config file
	ConfigFile string

	// AWSUserProfile is an aws profile that references a user (not a role)
	// leaving this empty typically means use `default` profile
	AWSUserProfile string `json:"aws_user_profile" yaml:"aws_user_profule"`

	// Path to your ssh private key
	SSHPrivateKey string `json:"ssh_private_key" yaml:"ssh_private_key"`

	// cert related
	CertLifetime Duration `json:"cert_lifetime" yaml:"cert_lifetime,inline"`
	// ask bless to sign for these remote users
	RemoteUsers []string `json:"remote_users" yaml:"remote_users"`
	// bless calls these bastion ips - your source ip. 0.0.0.0/0 is all
	BastionIPS []string `json:"bastion_ips" yaml:"bastion_ips"`
}

// LambdaConfig is the lambda config
type LambdaConfig struct {
	// RoleARN used to assume and invoke bless lambda
	RoleARN string `json:"role_arn" yaml:"role_arn"`
	// Bless lambda function name
	FunctionName string `json:"function_name" yaml:"function_name"`
	// bless lambda regions
	Regions []Region `json:"regions,omitempty" yaml:"regions,omitempty"`
}

// Duration is a wrapper around Duration to marshal/unmarshal
type Duration struct {
	time.Duration
}

// AsDuration returns as duration
func (d Duration) AsDuration() time.Duration {
	return d.Duration
}

// MarshalJSON marshals to json
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON unmarshals
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

// DefaultConfig generates a config with some defaults
func DefaultConfig() (*Config, error) {
	expandedDefaultConfigFile, err := homedir.Expand(DefaultConfigFile)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not expand %s", DefaultConfigFile)
	}

	c := &Config{
		ClientConfig: ClientConfig{
			ConfigFile:   expandedDefaultConfigFile,
			CertLifetime: Duration{30 * time.Minute},
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

// GetAWSSessionCachePath gets path to aws user session cache file
func (c *Config) GetAWSSessionCachePath() string {
	return path.Join(c.getCacheDir(), defaultAWSSessionCache, "cache.json")
}
