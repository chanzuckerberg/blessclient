package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootNoError(t *testing.T) {
	r := require.New(t)
	err := rootCmd.Execute()
	r.Nil(err)
}

func TestRootMissingVerbose(t *testing.T) {
	r := require.New(t)
	cmd := &cobra.Command{}
	err := rootCmd.PersistentPreRunE(cmd, nil)
	r.NotNil(err)
	r.Contains(err.Error(), "flag accessed but not defined: verbose")
}
