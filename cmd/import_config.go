package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/chanzuckerberg/blessclient/pkg/bless"
	getter "github.com/hashicorp/go-getter"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/segmentio/go-prompt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	importConfigCmd.Flags().StringP("config", "c", bless.DefaultConfigFile, "Use this to override the bless config file.")
	importConfigCmd.Flags().StringP("url", "u", "", "Use this to specify the url used to fetch your bless bless.")
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
			return bless.ErrMissingConfig
		}
		configFileExpanded, err := homedir.Expand(configFile)
		if err != nil {
			return errors.Wrapf(err, "Could not exapnd %s", configFile)
		}

		// for importing a config
		// we assume user has a standard setup
		sshDirExpanded, err := homedir.Expand("~/.ssh")
		if err != nil {
			return errors.Wrapf(err, "Could not exapnd %s", "~/.ssh")
		}

		src, err := cmd.Flags().GetString("url")
		if err != nil || src == "" {
			return bless.ErrMissingConfigURL
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
		conf, err := bless.FromFile(f.Name())
		if err != nil {
			return err
		}
		conf.SetPaths(configFileExpanded)

		// Try to use the default id_rsa key
		conf.ClientConfig.SSHPrivateKey = path.Join(sshDirExpanded, "id_rsa")
		_, err = os.Stat(conf.ClientConfig.SSHPrivateKey)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("Found no ssh key at %s, please generate one", conf.ClientConfig.SSHPrivateKey)
			}
			return errors.Wrapf(err, "Error reading %s", conf.ClientConfig.SSHPrivateKey)
		}

		// Now try doing something about the ssh config
		err = sshConfig(conf)
		if err != nil {
			return err
		}
		return conf.Persist()
	},
}

func sshConfig(conf *bless.Config) error {
	if conf.SSHConfig == nil {
		return nil // nothing to do
	}

	// Populate the inferred key
	for i := range conf.SSHConfig.Bastions {
		conf.SSHConfig.Bastions[i].IdentityFile = conf.ClientConfig.SSHPrivateKey
	}

	sshConfig, err := conf.SSHConfig.String()
	if err != nil {
		return err
	}
	log.Infof("Generated SSH Config:\n%s", sshConfig)
	openFileFlag := os.O_CREATE | os.O_WRONLY

	options := []string{"append", "overwrite", "nothing"}
	i := prompt.Choose("What would you like us to do with the generated ssh config", options)
	switch options[i] {
	case "append":
		openFileFlag = openFileFlag | os.O_APPEND
	case "overwrite":
		openFileFlag = openFileFlag | os.O_TRUNC
	case "nothing":
		return nil // nothing to do
	}

	sshDir := path.Dir(conf.ClientConfig.SSHPrivateKey)
	sshConfPath := path.Join(sshDir, "config")
	f, err := os.OpenFile(sshConfPath, openFileFlag, 0644)
	if err != nil {
		return errors.Wrapf(err, "Could not open ssh conf at %s", sshConfPath)
	}
	defer f.Close()

	_, err = f.WriteString(sshConfig)
	if err != nil {
		return errors.Wrapf(err, "Could not write ssh conf to %s", sshConfPath)
	}
	log.Infof("ssh config %s to %s", options[i], sshConfPath)
	return nil
}
