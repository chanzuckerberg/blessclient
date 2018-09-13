package config

import (
	"encoding/json"
	"errors"
	"time"
)

// Config is a blessclient config
type Config struct {
	Regions      []Region     `json:"regions"`
	ClientConfig ClientConfig `json:"client_config"`
	LambdaConfig LambdaConfig `json:"lambda_config"`
}

// Region is an aws region
type Region struct {
	Name         string `json:"name"`
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
	UserRole       string   `json:"user_role"`
	AccountID      string   `json:"account_id"`
	FunctionName   string   `json:"function_name"`
	CertLifetime   Duration `json:"cert_lifetime"`
	TimeoutConnect Duration `json:"timeout_connect"`
	TimeoutRead    Duration `json:"timeout_read"`
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
