package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultClientDir is the default dir where blessclient will look for a config and cache
	DefaultClientDir = "~/.blessclient"
	// DefaultConfigFile is the default file where blessclient will look for its config
	DefaultConfigFile = "~/.blessclient/config.yml"
	// DefaultCacheDir is a default cache dir
	DefaultCacheDir = "~/.blessclient/cache"
	// DefaultKMSAuthCache is the default kmsauth cache
	DefaultKMSAuthCache = "kmsauth"
	// DefaultAWSProfile is the default bless aws profile
	DefaultAWSProfile = "bless"
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

	// Telemetry does telemetry
	Telemetry Telemetry `yaml:"telemetry"`
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
	// client dir is a directory used by blessclient to hold config and cache
	ClientDir string `json:"client_dir" yaml:"client_dir"`
	// ConfigFile is the path to blessclient config file
	ConfigFile string `json:"config_file" yaml:"config_file"`
	// CacheDir is a path to the blessclient cache
	CacheDir string `json:"cache_dir" yaml:"cache_dir"`
	// KMSAuthCacheDir is a path to the kmsauth cache directory
	KMSAuthCacheDir string `json:"kmsauth_cache_dir" yaml:"kmsauth_cache_dir"`
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

// Telemetry to track adoption
type Telemetry struct {
	Honeycomb Honeycomb `yaml:"honeycomb,omitempty"`
}

// Honeycomb telemetry configuration
type Honeycomb struct {
	WriteKey string `yaml:"write_key,omitempty"`
	Dataset  string `yaml:"dataset,omitempty"`
}

// DefaultConfig generates a config with some defaults
func DefaultConfig() *Config {
	return &Config{
		ClientConfig: ClientConfig{
			ClientDir:       DefaultClientDir,
			CacheDir:        DefaultCacheDir,
			KMSAuthCacheDir: DefaultKMSAuthCache,
			CertLifetime:    Duration{30 * time.Minute},
		},
		LambdaConfig: LambdaConfig{
			RoleARN: DefaultAWSProfile, // seems like a sane default
		},
	}
}

// FromFile reads the config from file
func FromFile(file string) (*Config, error) {
	conf := DefaultConfig()
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
	err := os.MkdirAll(c.ClientConfig.ClientDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "Could not create client config dir %s", c.ClientConfig.ClientDir)
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

// SetPaths sets paths on the config
func (c *Config) SetPaths(configPath string) {
	c.ClientConfig.ClientDir = path.Dir(configPath)
	c.ClientConfig.ConfigFile = configPath
	c.ClientConfig.CacheDir = path.Join(c.ClientConfig.ClientDir, "cache", util.VersionCacheKey())
	c.ClientConfig.KMSAuthCacheDir = path.Join(c.ClientConfig.CacheDir, DefaultKMSAuthCache)
}
