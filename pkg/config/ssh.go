package config

import (
	"bytes"
	"text/template"
	"time"

	"github.com/chanzuckerberg/blessclient/pkg/util"
	"github.com/pkg/errors"
)

const (
	// small template - just inline so we don't have to deal with packing it to binary
	sshConfigTemplate = `
######### Generated by blessclient v{{ version }} at {{ now }}#############
{{ range .Bastions }}{{ $bastion := . }}
Match OriginalHost  {{ .Pattern }} exec "{{ .SSHExecCommand.String }}"
  IdentityFile {{ .IdentityFile }}
  User {{ .User }}

Host {{ .Pattern }}
  User {{ .User }}
{{ range .Hosts }}
Host {{ .Pattern }}
  ProxyJump {{ $bastion.Pattern }}
  User {{ $bastion.User }}
{{ end }}{{ end }}
`
)

func now() string {
	return time.Now().UTC().Format(time.RFC822Z)
}

// SSHConfig is an SSH config
// We make some assumptions here around the structure of the machines
// A bastion is internet accessible and can be used to reach other machines
type SSHConfig struct {
	Bastions []Bastion `yaml:"bastions"`
}

// String generates the ssh config string
func (s *SSHConfig) String() (string, error) {
	fnMap := make(template.FuncMap)
	fnMap["now"] = now
	fnMap["version"] = util.VersionString
	fnMap["defaulted"] = defaulted

	t, err := template.New("ssh_config").Funcs(fnMap).Parse(sshConfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "Could not parse ssh_config template")
	}

	b := bytes.NewBuffer(nil)
	err = t.Execute(b, s)
	if err != nil {
		return "", errors.Wrap(err, "Could not templetize ssh_config")
	}
	return b.String(), nil
}

// Bastion is an internet accessibly server used to "jump" to other servers
type Bastion struct {
	Host `yaml:",inline"`

	Hosts          []Host          `yaml:"hosts"`
	IdentityFile   string          `yaml:"identity_file"`
	User           string          `yaml:"user"`
	SSHExecCommand *SSHExecCommand `yaml:"ssh_exec_command,omitempty"`
}

// SSHExecCommand is a command to execute on successful ssh match
type SSHExecCommand string

// String gets the value of this exec command
func (ec *SSHExecCommand) String() string {
	if ec == nil {
		return "blessclient run"
	}
	return string(*ec)
}

// Host represents a Host block in an ssh config
type Host struct {
	Pattern string `yaml:"pattern"`
}
