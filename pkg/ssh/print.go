package ssh

import (
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type certToPrint struct {
	ID         string   `yaml:"id"`
	Principals []string `yaml:"principals"`

	Extensions map[string]string

	ValidBefore time.Time
	ValidAfter  time.Time
}

func PrintCertificate(cert *ssh.Certificate, w io.Writer) error {
	if cert == nil {
		return nil
	}

	toPrint := &certToPrint{}

	toPrint.ID = cert.KeyId
	toPrint.Principals = cert.ValidPrincipals
	toPrint.ValidAfter = time.Unix(int64(cert.ValidAfter), 0)
	toPrint.ValidBefore = time.Unix(int64(cert.ValidBefore), 0)
	toPrint.Extensions = cert.Extensions

	data, err := yaml.Marshal(toPrint)
	if err != nil {
		return errors.Wrap(err, "could not yaml marshal cert")
	}

	_, err = fmt.Fprintln(w, string(data))
	return errors.Wrap(err, "could not print cert")
}
