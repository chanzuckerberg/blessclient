package cmd

import (
	"fmt"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of blessclient",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, err := util.VersionString()
		if err != nil {
			return err
		}
		fmt.Println(v)
		return nil
	},
}
