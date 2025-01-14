package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/config"
	oidc "github.com/chanzuckerberg/go-misc/oidc_cli/oidc_impl"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(tokenCmd)
}

const (
	stdoutTokenVersion = 1
)

type stdoutToken struct {
	Version int `json:"version,omitempty"`

	IDToken     string    `json:"id_token,omitempty"`
	AccessToken string    `json:"access_token,omitempty"`
	Expiry      time.Time `json:"expiry,omitempty"`
}

var tokenCmd = &cobra.Command{
	Use:           "token",
	Short:         "token prints the oidc tokens to stdout",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		stdoutToken := &stdoutToken{
			Version: stdoutTokenVersion,
		}

		config, err := config.FromFile(config.DefaultConfigFile)
		if err != nil {
			return err
		}

		token, err := oidc.GetToken(
			cmd.Context(),
			config.ClientConfig.OIDCClientID,
			config.ClientConfig.OIDCIssuerURL,
		)
		if err != nil {
			return err
		}

		stdoutToken.AccessToken = token.AccessToken
		stdoutToken.IDToken = token.IDToken
		stdoutToken.Expiry = token.Expiry

		data, err := json.Marshal(stdoutToken)
		if err != nil {
			return errors.Wrap(err, "could not json marshal oidc token")
		}

		_, err = fmt.Fprintln(os.Stdout, string(data))
		return errors.Wrap(err, "could not print token to stdout")
	},
}
