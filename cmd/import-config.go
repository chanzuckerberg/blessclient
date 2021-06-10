package cmd

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	getter "github.com/hashicorp/go-getter"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	prompt "github.com/segmentio/go-prompt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(importConfigCmd)
}

var importConfigCmd = &cobra.Command{
	Use:           "import-config",
	Short:         "Import a blessclient config from a remote source",
	Args:          cobra.ExactArgs(1),
	Long:          "This command fetches a config from a remote source and writes it to disk",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]

		f, err := ioutil.TempFile("", "blessconfig")
		if err != nil {
			return errors.Wrap(err, "Could not create temporary file for config")
		}
		defer f.Close()
		defer os.Remove(f.Name())

		err = getter.GetFile(f.Name(), src)
		if err != nil {
			return errors.Wrapf(err, "Could not fetch %s", src)
		}

		conf, err := config.FromFile(f.Name())
		if err != nil {
			return err
		}

		// Now try doing something about the ssh config
		err = sshConfig(conf)
		if err != nil {
			return err
		}

		return conf.Persist(config.DefaultConfigFile)
	},
}

func sshConfig(conf *config.Config) error {
	if conf.SSHConfig == nil {
		return nil // nothing to do
	}

	sshDir, err := homedir.Expand("~/.ssh")
	if err != nil {
		return errors.Wrap(err, "could not expand (~/.ssh)")
	}

	// Make sure we have an SSH config file, create one if not present
	sshConfPath := path.Join(sshDir, "config")
	sshConf, err := os.OpenFile(sshConfPath, os.O_CREATE|os.O_RDONLY, 0644) // #nosec
	if err != nil {
		return errors.Wrapf(err, "Could not open ssh conf at %s", sshConfPath)
	}
	defer sshConf.Close()

	// We rely on the ssh_config Include directive to make things more intuitive
	blessSSHConfPath := path.Join(sshDir, "blessconfig")
	blessConfig, err := conf.SSHConfig.String()
	if err != nil {
		return err
	}

	// check if ok to overwrite old blessconfig
	if !prompt.Confirm("OK to overwrite your %s", blessSSHConfPath) {
		logrus.Infof("Exiting, won't overwrite %s", blessSSHConfPath)
		return nil
	}

	f, err := os.OpenFile(blessSSHConfPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644) // #nosec
	if err != nil {
		return errors.Wrapf(err, "Could not open ssh conf at %s", blessSSHConfPath)
	}
	defer f.Close()

	_, err = f.WriteString(blessConfig)
	if err != nil {
		return errors.Wrapf(err, "Could not write bless ssh conf to %s", blessSSHConfPath)
	}

	// Prompt users to Include our blessconfig in their ssh config
	logrus.Warnf(`
Please add the following to the top of your %s:

Include %s
`, sshConfPath, blessSSHConfPath)
	return nil
}
