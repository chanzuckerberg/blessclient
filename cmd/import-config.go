package cmd

import (
	"io/ioutil"
	"os"
	"path"

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
	rootCmd.AddCommand(importConfigCmd)
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
		if err != nil || src == "" {
			return errs.ErrMissingConfigURL
		}
		f, err := ioutil.TempFile("", "blessconfig")
		if err != nil {
			return errors.Wrap(err, "Could not create temporary file for config")
		}
		defer f.Close()
		defer os.Remove(f.Name())

		err = getter.GetFile(f.Name(), src)
		if err != nil {
			return errors.Wrapf(err, "Could not fetch %s to %s", src, configFileExpanded)
		}

		// Need to add some specific conf for user environment
		conf, err := config.FromFile(f.Name())
		if err != nil {
			return err
		}

		conf.ClientConfig.ClientDir = path.Dir(configFileExpanded)
		conf.ClientConfig.ConfigFile = configFileExpanded
		conf.ClientConfig.CacheDir = path.Join(conf.ClientConfig.ClientDir, "cache") // TODO: version the cache
		conf.ClientConfig.KMSAuthCacheDir = path.Join(conf.ClientConfig.CacheDir, config.DefaultKMSAuthCache)
		return conf.Persist()
	},
}
