package cmd

import (
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	getter "github.com/hashicorp/go-getter"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	importConfigCmd.Flags().StringP("config", "c", config.DefaultConfigFile, "Use this to override the bless config file.")
	importConfigCmd.Flags().StringP("url", "u", "", "Use this to specify the url used to fetch your bless config.")
}

var importConfigCmd = &cobra.Command{
	Use:           "import-config",
	Short:         "Import a blessclient config from a remote source",
	Long:          "This command fetches a config from a remote source and writes it to disk",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return errs.ErrMissingConfig
		}
		configFileExpanded, err := homedir.Expand(configFile)
		if err != nil {
			return errors.Wrapf(err, "Could not exapnd %s", configFile)
		}

		src, err := cmd.Flags().GetString("url")
		if err != nil {
			return errs.ErrMissingConfigURL
		}
		if src == "" {
			return errs.ErrMissingConfigURL
		}
		err = getter.GetFile(configFileExpanded, src)
		return errors.Wrapf(err, "Could not fetch %s to %s", src, configFileExpanded)
	},
}
