package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/errs"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Config is a blessclient config
type Config struct {
	Regions      []Region     `json:"regions"`
	ClientConfig ClientConfig `json:"client_config"`
	LambdaConfig LambdaConfig `json:"lambda_config"`
}

// Region is an aws region
type Region struct {
	AWSRegion    string `json:"aws_region"`
	KMSAuthKeyID string `json:"kms_auth_key_id"`
}

// ClientConfig is the client config
type ClientConfig struct {
	CacheDir         string `json:"cache_dir"`
	MFACacheFile     string `json:"mfa_cache_file"`
	KMSAuthCacheFile string `json:"kms_auth_cache_file"`
}

// LambdaConfig is the lambda config
type LambdaConfig struct {
	// AWS profile to assume and invoke lambda
	Profile      string   `json:"profile"`
	FunctionName string   `json:"function_name"`
	CertLifetime Duration `json:"cert_lifetime"`
}

// Duration is a wrapper around Duration to marshal/unmarshal
type Duration struct {
	time.Duration
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
			CacheDir:         "~/.blessclient",
			MFACacheFile:     "mfa-cache.json",
			KMSAuthCacheFile: "kmsauth-cache.json",
		},
		LambdaConfig: LambdaConfig{},
		Regions:      []Region{},
	}
}

// NewFromFile reads the config from file
func NewFromFile(file string) (*Config, error) {
	conf := DefaultConfig()
	b, err := ioutil.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errs.ErrMissingConfig
		}
		return nil, errors.Wrapf(err, "Could not read config %s, you can generate one with bless init", file)
	}

	err = yaml.Unmarshal(b, conf)
	if err != nil {
		return nil, errors.WithMessage(err, "Invalid config, make sure it is valid yaml")
	}
	return conf, nil
}
