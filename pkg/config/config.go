package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/errs"
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
	DefaultKMSAuthCache = "kmsauth-cache"
	// DefaultAWSProfile is the default bless aws profile
	DefaultAWSProfile = "bless"
)

// Config is a blessclient config
type Config struct {
	Version int `json:"version" yaml:"version"`

	ClientConfig ClientConfig `json:"client_config" yaml:"client_config"`
	LambdaConfig LambdaConfig `json:"lambda_config" yaml:"lambda_config"`
	SSHConfig    *SSHConfig   `json:"ssh_config,omitempty" yaml:"ssh_config,omitempty"`
}

// Region is an aws region
type Region struct {
	AWSRegion    string `json:"aws_region" yaml:"aws_region"`
	KMSAuthKeyID string `json:"kms_auth_key_id" yaml:"kms_auth_key_id"`
}

// ClientConfig is the client config
type ClientConfig struct {
	ClientDir       string `json:"client_dir" yaml:"client_dir"`
	ConfigFile      string `json:"config_file" yaml:"config_file"`
	CacheDir        string `json:"cache_dir" yaml:"cache_dir"`
	KMSAuthCacheDir string `json:"kmsauth_cache_dir" yaml:"kmsauth_cache_dir"`

	SSHPrivateKey string `json:"ssh_private_key" yaml:"ssh_private_key"`

	// cert related
	CertLifetime Duration `json:"cert_lifetime" yaml:"cert_lifetime,inline"`
	RemoteUsers  []string `json:"remote_users" yaml:"remote_users"`
	BastionIPS   []string `json:"bastion_ips" yaml:"bastion_ips"`
}

// LambdaConfig is the lambda config
type LambdaConfig struct {
	// RoleARN used to assume and invoke bless lambda
	RoleARN      string   `json:"role_arn" yaml:"role_arn"`
	FunctionName string   `json:"function_name" yaml:"function_name"`
	Regions      []Region `json:"regions,omitempty" yaml:"regions,omitempty"`
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
