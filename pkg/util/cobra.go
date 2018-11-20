package util

import (
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// GetConfigPath gets the config path from a cobra cmd
func GetConfigPath(cmd *cobra.Command) (string, error) {
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return "", errors.New("Missing config")
	}
	expandedConfigFile, err := homedir.Expand(configFile)
	if err != nil {
		return "", errors.Wrapf(err, "Could not expand %s", configFile)
	}
	return expandedConfigFile, nil
}
