package cmd

import (
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
		conf, err := config.DefaultConfig()
		if err != nil {
			return err
		}

		// override the config path if needed
		configFileExpanded, err := homedir.Expand(configFile)
		if err != nil {
			return errors.Wrapf(err, "Could not expand %s", configFile)
		}
		conf.ClientConfig.ConfigFile = configFileExpanded

		// Ask for some user values
		conf.ClientConfig.SSHPrivateKey = promptWithDefault("~/.ssh/id_rsa", "path to the ssh private key to use (~/.ssh/id_rsa)")
		conf.ClientConfig.AWSUserProfile = promptWithDefault("default", "Enter AWS User Profile (default)")
		conf.LambdaConfig.RoleARN = prompt.StringRequired("role arn to invoke lambda")
		conf.LambdaConfig.FunctionName = prompt.StringRequired("bless lambda function name")

		// Add regions
		regions := []config.Region{}
		article := "a"
		for prompt.Confirm("Would you like to add %s region to your bless config? (y/n)", article) {
			article = "another"
			region := config.Region{
				AWSRegion:    promptWithDefault("us-west-2", "Aws region (us-west-2)"),
				KMSAuthKeyID: prompt.StringRequired("The kms auth key_id for this region"),
			}
			regions = append(regions, region)
		}

		conf.LambdaConfig.Regions = regions
		return conf.Persist()
	},
}

func promptWithDefault(defaultValue string, promptString string, args ...interface{}) string {
	input := prompt.String(promptString, args...)
	if input == "" {
		return defaultValue
	}
	return input
}
