package cmd

import (
	"path"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	prompt "github.com/segmentio/go-prompt"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.Flags().StringP("config", "c", config.DefaultConfigFile, "Use this to override the bless config file.")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:           "init",
	Short:         "Initialize your bless config",
	Long:          "This command asks for input and generates your blessclient config",
	SilenceErrors: true, // to handle them centrally
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return errs.ErrMissingConfig
		}
		conf := config.DefaultConfig()
		configFileExpanded, err := homedir.Expand(configFile)
		if err != nil {
			return errors.Wrapf(err, "Could not exapnd %s", configFile)
		}

		conf.ClientConfig.ClientDir = path.Dir(configFileExpanded)
		conf.ClientConfig.ConfigFile = configFileExpanded
		conf.ClientConfig.CacheDir = path.Join(conf.ClientConfig.ClientDir, "cache") // TODO: version the cache
		conf.ClientConfig.KMSAuthCacheFile = path.Join(conf.ClientConfig.CacheDir, config.DefaultKMSAuthCache)
		conf.ClientConfig.MFACacheFile = path.Join(conf.ClientConfig.CacheDir, config.DefaultMFACache)

		// Ask for some user values
		conf.LambdaConfig.RoleARN = prompt.StringRequired("aws profile to invoke bless lambda")
		conf.LambdaConfig.FunctionName = prompt.StringRequired("bless lambda function name")

		// Add regions
		regions := []config.Region{}
		for prompt.Confirm("Would you like to add another region to your bless config? (y/n)") {
			region := config.Region{
				AWSRegion:    prompt.StringRequired("Aws region (ex: us_west_2)"),
				KMSAuthKeyID: prompt.StringRequired("The kms auth key_id for this region"),
			}
			regions = append(regions, region)
		}

		conf.LambdaConfig.Regions = regions

		return conf.Persist()
	},
}
