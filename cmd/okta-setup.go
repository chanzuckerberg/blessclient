package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/99designs/keyring"
	"github.com/chanzuckerberg/blessclient/pkg/config"
	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	awsokta "github.com/segmentio/aws-okta/lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(oktaSetupCmd)
}

var oktaSetupCmd = &cobra.Command{
	Use:           "okta-setup",
	Short:         "okta-setup sets up Okta login credentials",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		expandedConfigFile, err := util.GetConfigPath(cmd)
		if err != nil {
			return err
		}
		log.Debugf("Reading config from %s", expandedConfigFile)
		conf, err := config.FromFile(expandedConfigFile)
		if err != nil {
			return err
		}
		log.Debugf("Parsed config is: %s", spew.Sdump(conf))

		if conf.OktaConfig == nil {
			return errors.Errorf("The okta_config section is not found in your config")
		}

		kr, err := awsokta.OpenKeyring(nil)
		if err != nil {
			return err
		}
		username, err := awsokta.Prompt("Okta username", false)
		if err != nil {
			return err
		}

		password, err := awsokta.Prompt("Okta password", true)
		if err != nil {
			return err
		}
		fmt.Println()

		organization := conf.OktaConfig.Organization
		domain := conf.OktaConfig.Domain

		creds := awsokta.OktaCreds{
			Organization: organization,
			Username:     username,
			Password:     password,
			Domain:       domain,
		}

		mfaConfig := conf.GetOktaMFAConfig()
		if err = creds.Validate(mfaConfig); err != nil {
			return errors.Wrap(err, "Failed to verify Okta credentials")
		}

		encoded, err := json.Marshal(creds)
		if err != nil {
			return errors.Wrap(err, "Failed to encode credentials in JSON")
		}

		// The key corresponds to the same key id set via
		// aws-okta.
		oktaKeyringKeyID := "okta-creds"
		if conf.OktaConfig.KeyringKeyID != nil {
			oktaKeyringKeyID = *conf.OktaConfig.KeyringKeyID
		}
		item := keyring.Item{
			Key:                         oktaKeyringKeyID,
			Data:                        encoded,
			Label:                       "okta credentials",
			KeychainNotTrustApplication: false,
		}

		if err := kr.Set(item); err != nil {
			return errors.Wrap(err, "Failed to save credentials in credential store")
		}

		log.Infof("Added credentials for user %s", username)
		return nil
	},
}
