package config

import (
	"encoding/json"
	"errors"
	"time"
)

// Config is a blessclient config
type Config struct {
	Regions      []Region `json:"regions,omitempty"`
	ClientConfig ClientConfig
}

// Region is an aws region
type Region struct {
	Name         string `json:"name,omitempty"`
	AWSRegion    string `json:"aws_region,omitempty"`
	KMSAuthKeyID string `json:"kms_auth_key_id,omitempty"`
}

// ClientConfig is the client config
type ClientConfig struct {
	CacheDir     string
	CacheFile    string
	MFACacheDir  string
	MFACacheFile string
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

// LambdaConfig is the lambda config
type LambdaConfig struct {
	UserRole       string
	AccountID      string
	FunctionName   string
	CertLifetime   Duration
	TimeoutConnect Duration
	TimeoutRead    Duration
}
