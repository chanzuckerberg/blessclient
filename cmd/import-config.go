package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/errs"
	cziAWS "github.com/chanzuckerberg/go-misc/aws"
	getter "github.com/hashicorp/go-getter"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	prompt "github.com/segmentio/go-prompt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	importConfigCmd.Flags().StringP("config", "c", config.DefaultConfigFile, "Use this to override the bless config file.")
	importConfigCmd.Flags().StringP("key-file", "k", config.DefaultSSHPrivateKey, "Location of SSH private key")
	rootCmd.AddCommand(importConfigCmd)
}

var importConfigCmd = &cobra.Command{
	Use:           "import-config",
	Short:         "Import a blessclient config from a remote source",
	Args:          cobra.ExactArgs(1),
	Long:          "This command fetches a config from a remote source and writes it to disk",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			return errs.ErrMissingConfig
		}
		configFileExpanded, err := homedir.Expand(configFile)
		if err != nil {
			return errors.Wrapf(err, "Could not expand %s", configFile)
		}

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

		// Need to add some specific conf for user environment
		conf, err := config.FromFile(f.Name())
		if err != nil {
			return err
		}
		conf.ClientConfig.ConfigFile = configFileExpanded

		sshPrivateKey, err := cmd.Flags().GetString("key-file")
		if err != nil {
			return errors.Wrapf(err, "Could not get ssh directory")
		}

		// Try to use the default id_rsa key
		sshPrivateKeyExpanded, err := homedir.Expand(sshPrivateKey)
		if err != nil {
			return errors.Wrapf(err, "Could not expand ssh private key %s", sshPrivateKey)
		}
		conf.ClientConfig.SSHPrivateKey = sshPrivateKeyExpanded
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

		sess, err := session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
				Profile:           conf.ClientConfig.AWSUserProfile,
			},
		)
		if err != nil {
			return errors.Wrap(err, "Could not create aws session")
		}
		awsClient := cziAWS.New(sess).WithAllServices(nil)
		err = setTelemetrySecret(ctx, conf, awsClient)
		if err != nil {
			return err
		}
		err = conf.SetAWSUsernameIfMissing(ctx, awsClient)
		if err != nil {
			return err
		}
		return conf.Persist()
	},
}

func setTelemetrySecret(ctx context.Context, conf *config.Config, awsClient *cziAWS.Client) error {
	// TODO: fail here or just ignore errors?
	if conf.Telemetry.Honeycomb == nil {
		return nil
	}
	parsedARN, err := arn.Parse(conf.Telemetry.Honeycomb.SecretManagerARN)
	if err != nil {
		return errors.Wrapf(err, "Could not parse arn %s", conf.Telemetry.Honeycomb.SecretManagerARN)
	}

	// Configure the region
	awsClient.WithAllServices(&aws.Config{Region: aws.String(parsedARN.Region)})

	secretARN := conf.Telemetry.Honeycomb.SecretManagerARN
	secret, err := awsClient.SecretsManager.ReadStringLatestVersion(ctx, secretARN)
	if err != nil {
		return err
	}
	if secret == nil {
		return errors.New("No telemetry secret found")
	}
	conf.Telemetry.Honeycomb.WriteKey = *secret
	return nil
}

func sshConfig(conf *config.Config) error {
	if conf.SSHConfig == nil {
		return nil // nothing to do
	}

	sshDir := path.Dir(conf.ClientConfig.SSHPrivateKey)
	sshConfPath := path.Join(sshDir, "config")
	err := backupFile(sshConfPath, fmt.Sprintf("%s.%d.bak", sshConfPath, time.Now().UTC().Unix()))
	if err != nil {
		// Unsure if we want to error out here
		log.Warnf("Error backing up %s: %s", sshConfPath, err.Error())
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
	i := prompt.Choose("What would you like us to do with the generated ~/.ssh/config", options)
	switch options[i] {
	case "append":
		openFileFlag = openFileFlag | os.O_APPEND
	case "overwrite":
		openFileFlag = openFileFlag | os.O_TRUNC
	case "nothing":
		return nil // nothing to do
	}

	f, err := os.OpenFile(sshConfPath, openFileFlag, 0644)
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

	infile, err := os.Open(src)
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
