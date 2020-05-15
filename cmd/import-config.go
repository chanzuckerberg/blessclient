package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	getter "github.com/hashicorp/go-getter"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	prompt "github.com/segmentio/go-prompt"
	log "github.com/sirupsen/logrus"
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

	sshConfPath := path.Join(sshDir, "config")
	err = backupFile(sshConfPath, fmt.Sprintf("%s.%d.bak", sshConfPath, time.Now().UTC().Unix()))
	if err != nil {
		return err
	}

	sshConfig, err := conf.SSHConfig.String()
	if err != nil {
		return err
	}

	log.Infof("Generated SSH Config:\n%s", sshConfig)
	openFileFlag := os.O_CREATE | os.O_WRONLY

	options := []string{"append", "overwrite", "nothing"}
	i := prompt.Choose("What would you like us to do with the generated ~/.ssh/config", options)
	switch options[i] {
	case "append":
		openFileFlag = openFileFlag | os.O_APPEND
	case "overwrite":
		openFileFlag = openFileFlag | os.O_TRUNC
	case "nothing":
		return nil // nothing to do
	}

	f, err := os.OpenFile(sshConfPath, openFileFlag, 0644) // #nosec
	if err != nil {
		return errors.Wrapf(err, "Could not open ssh conf at %s", sshConfPath)
	}
	defer f.Close()

	_, err = f.WriteString(sshConfig)
	if err != nil {
		return errors.Wrapf(err, "Could not write ssh conf to %s", sshConfPath)
	}
	log.Infof("%s ssh config to %s", options[i], sshConfPath)
	return nil
}

func backupFile(src string, dst string) error {
	if !prompt.Confirm("Backup %s to %s (y/n)", src, dst) {
		return nil
	}

	infile, err := os.Open(src) // #nosec
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("%s does not exist, no need to back up!", src)
			return nil
		}
		return errors.Wrapf(err, "Could not open %s", src)
	}
	defer infile.Close()

	outfile, err := os.Create(dst)
	if err != nil {
		return errors.Wrapf(err, "Error opening destination %s", dst)
	}
	defer outfile.Close()

	_, err = io.Copy(outfile, infile)
	if err != nil {
		return errors.Wrap(err, "Could not copy file")
	}
	log.Infof("%s backed up to %s", src, dst)
	return nil
}
