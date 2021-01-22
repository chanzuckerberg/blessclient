package config

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

const (
	// ConfigVersion specifies the current config version
	ConfigVersion = 1

	// DefaultConfigFile is the default file where blessclient will look for its config
	DefaultConfigFile = "~/.blessclient/config.yml"
)

// Config is a blessclient config
type Config struct {
	// Version versions this config
	Version int `yaml:"version"`

	// ClientConfig has configuration related to blessclient
	ClientConfig ClientConfig `yaml:"client_config"`
	// LambdaConfig holds configuration around the bless lambda
	LambdaConfig LambdaConfig `yaml:"lambda_config"`
	// For convenience, you can bundle an ~/.ssh/config template here
	SSHConfig *SSHConfig `yaml:"ssh_config,omitempty"`
}

type ClientConfig struct {
	// The OIDC client_id
	OIDCClientID string `yaml:"oidc_client_id"`
	// Oidc issuer url: eg: foo.okta.com
	OIDCIssuerURL string `yaml:"oidc_issuer_url"`
	// RoleARN is the aws role arn to assume to invoke the CA lambda
	RoleARN string `yaml:"role_arn"`
}

// Region is an aws region that contains an aws lambda
type Region struct {
	// name of the aws region (us-west-2)
	AWSRegion string `yaml:"aws_region"`
}

// LambdaConfig is the lambda config
type LambdaConfig struct {
	// Bless lambda function name
	FunctionName string `yaml:"function_name"`
	// Bless lambda function version (lambda alias or version qualifier)
	FunctionVersion *string `yaml:"function_version,omitempty"`
	// bless lambda regions
	Regions []Region `yaml:"regions,omitempty"`
}

// DefaultConfig generates a config with some defaults
func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
	}
}

func FromFile(confPath string) (*Config, error) {
	expandedConfPath, err := homedir.Expand(confPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not expand homedir")
	}

	b, err := ioutil.ReadFile(expandedConfPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read config at %s", confPath)
	}

	conf := &Config{}
	err = yaml.Unmarshal(b, conf)
	if err != nil {
		return nil, errors.Wrapf(err, "could not yaml unmarshal config at %s", confPath)
	}

	if conf.Version != ConfigVersion {
		return nil, errors.Errorf("expected config version %d but got %d", ConfigVersion, conf.Version)
	}
	return conf, nil
}

// Persist persists a config to disk
func (c *Config) Persist(configPath string) error {
	configPath, err := GetOrCreateConfigPath(configPath)
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return errors.Wrap(err, "Error marshaling config")
	}

	err = ioutil.WriteFile(configPath, b, 0644)
	if err != nil {
		return errors.Wrapf(err, "Could not write config to %s", configPath)
	}
	log.Debugf("Config written to %s", configPath)
	return nil
}

func GetOrCreateConfigPath(configPath string) (string, error) {
	expandedConfigFile, err := homedir.Expand(configPath)
	if err != nil {
		return "", errors.Wrapf(err, "could not expand %s", expandedConfigFile)
	}
	blessclientDir := path.Dir(expandedConfigFile)

	err = os.MkdirAll(blessclientDir, 0755) // #nosec
	if err != nil {
		return "", errors.Wrapf(err, "Could not create client config dir %s", blessclientDir)
	}
	return expandedConfigFile, nil
}
