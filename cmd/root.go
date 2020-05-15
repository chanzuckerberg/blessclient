package cmd

import (
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	flagVerbose = "verbose"
)

func init() {
	rootCmd.PersistentFlags().BoolP(flagVerbose, "v", false, "Use this to enable verbose mode")
}

var pidLock *util.Lock
var rootCmd = &cobra.Command{
	Use:   "blessclient",
	Short: "",
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(pidLock.Unlock(), "Error releasing lock")
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Parse flags
		verbose, err := cmd.Flags().GetBool(flagVerbose)
		if err != nil {
			return errors.Wrap(err, "Missing verbose flag")
		}
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		// pid lock
		configPath, err := config.GetOrCreateConfigPath(config.DefaultConfigFile)
		if err != nil {
			return err
		}

		pidLock, err = util.NewLock(configPath)
		if err != nil {
			return err
		}
		return errors.Wrap(pidLock.Lock(), "Error acquiring lock")
	},
}

// Execute executes the command
func Execute() error {
	return rootCmd.Execute()
}
