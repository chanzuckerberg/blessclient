package cmd

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootNoError(t *testing.T) {
	a := assert.New(t)
	err := rootCmd.Execute()
	a.Nil(err)
}

func TestRootMissingVerbose(t *testing.T) {
	a := assert.New(t)
	cmd := &cobra.Command{}
	err := rootCmd.PersistentPreRunE(cmd, nil)
	a.NotNil(err)
	a.Contains(err.Error(), "flag accessed but not defined: verbose")
}

func TestRootMissingConfig(t *testing.T) {
	a := assert.New(t)
	cmd := &cobra.Command{}
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Use this to enable verbose mode")

	flags := []string{"--verbose"}
	err := cmd.ParseFlags(flags)
	a.Nil(err)

	err = rootCmd.PersistentPreRunE(cmd, nil)
	a.NotNil(err)
	a.Contains(err.Error(), "Missing config")
}

func TestRootLock(t *testing.T) {
	a := assert.New(t)
	cmd := &cobra.Command{}

	name, err := ioutil.TempDir("", "blessclient-tests")
	a.Nil(err)
	defer os.RemoveAll(name)

	cmd.PersistentFlags().BoolP("verbose", "v", false, "Use this to enable verbose mode")
	cmd.PersistentFlags().StringP(flagConfig, "c", name, "Use this to override the bless config file.")

	flags := []string{"--verbose", "--config", name}
	err = cmd.ParseFlags(flags)
	a.Nil(err)

	err = rootCmd.PersistentPreRunE(cmd, nil)
	a.Nil(err)

	err = rootCmd.PersistentPostRunE(cmd, nil)
	a.Nil(err)
}
