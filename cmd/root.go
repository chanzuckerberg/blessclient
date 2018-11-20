package cmd

import (
	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Use this to enable verbose mode")
}

var pidLock *util.Lock
var rootCmd = &cobra.Command{
	Use:   "blessclient",
	Short: "",
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		return pidLock.Unlock()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err != nil {
			return errors.Wrap(err, "Missing verbose flag")
		}
		if verbose {
			log.SetLevel(log.DebugLevel)
		}

		// pid lock
		configPath, err := util.GetConfigPath(cmd)
		if err != nil {
			return err
		}
		pidLock, err = util.NewLock(configPath)
		if err != nil {
			return err
		}
		return pidLock.Lock()
	},
}

// Execute executes the command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
