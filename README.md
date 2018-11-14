# blessclient
[![codecov](https://codecov.io/gh/chanzuckerberg/blessclient/branch/master/graph/badge.svg)](https://codecov.io/gh/chanzuckerberg/blessclient)

**Please note**: If you believe you have found a security issue, _please responsibly disclose_ by contacting us at [security@chanzuckerberg.com](mailto:security@chanzuckerberg.com).

----

Inspiration for this project comes from [lyft/python-blessclient](https://github.com/lyft/python-blessclient).
We decided to write in Go because it is much easier to distribute a statically linked binary to a large team than having to deal with python environments. Some features from [lyft/python-blessclient](https://github.com/lyft/python-blessclient) are currently missing but will be added over time while others are purposefully excluded.

## Install

### Mac

You can use homebrew -

```
brew tap chanzuckerberg/tap
brew install blessclient
```

### Linux, Windows, etc...
Binaries are available on the [releases](https://github.com/chanzuckerberg/blessclient/releases) page. Download one for your architecture, put it in your path and make it executable.

## Usage

At a high level:
1. [Install](#install) blessclient
1. If you don't have an SSH key, generate one with `ssh-keygen -t rsa -b 4096`
1. [Import](#import-config) or generate a blessclient config
1. Run `blessclient run` and make sure there are no errors
1. Modify your [ssh config](#sshconfig) to be bless compatible
1. ssh, scp, rsync as you normally would

### Config

By default, `blessclient` looks for configs in `~/.blessclient/config.yml`. You can always override this `blessclient run -c /my/new/config.yml`
Some more information on the config can be found [here](pkg/config/config.go).

There are two built-in methods to facilitate the generation of blessclient configs:

#### Import-config
You can also use pre-generated config files.

A few options here:
- `blessclient import-config git@github.com:/..../teamA/blessconfig.yml`
- `blessclient import-config https://www.github.com/..../teamA/blessconfig.yml`
- `blessclient import-config /home/user/.../teamA/blessconfig.yml`
- `blessclient import-config s3::https://s3.amazonaws.com/bucket/teamA/blessconfig.yml`

This command uses [go-getter](https://github.com/hashicorp/go-getter) to fetch a config and thus supports any source that [go-getter](https://github.com/hashicorp/go-getter#supported-protocols-and-detectors) supports.

You can see an example with dummy values [here](examples/config.yml). Download the example, modify the values, and `blessclient import-config <path>` it to get started.

### ssh-agent

You can optionally instruct blessclient to update your ssh-agent with your certificate. To do so, add `update_ssh_agent: true` to your blessclient config.

```yml
client_config:
  update_ssh_agent: true
...
```
### .ssh/config

This is the nice part about blessclient - in general, you can write an ssh config to transparently use blessclient. scp, rsync, etc should all be compatible!

Such an ssh config could look like:

```
Match OriginalHost bastion.foo.com exec "blessclient run"
  IdentityFile ~/.ssh/id_rsa

Host 10.0.*
  ProxyJump bastion.foo.com
  User admin

Host bastion.foo.com
  User admin
```

This ssh config does a couple of interesting things -

- It transparently requests an ssh certificate if needed
- It transparently does a ProxyJump through a bastion host (assuming 10.0.* is an ipblock for machines behind the bastion)

## Telemetry
There currently is some basic trace instrumentation using [honeycomb](https://www.honeycomb.io/). We use this internally to track usage, gather performance statistics, and error reporting. Telemetry is disabled without a honeycomb write key - which you must provide through the [config](pkg/config/config.go).

## Common Errors

### Unsafe RSA public key
Bless lambda is rejecting your key because because it is not cryptographically sound. You can generate a new key `ssh-keygen -t rsa -b 4096` and use that instead.

### SSH client 7.8 can't connect with certificates
There are a couple of outstanding bugs related to openSSH client 7.8
- https://bugs.launchpad.net/ubuntu/+source/openssh/+bug/1790963
- https://bugzilla.redhat.com/show_bug.cgi?id=1623929
- https://bugs.archlinux.org/task/59838

You can check your version with
```
ssh -V
```

## Other
### Deploying BLESS
There are already [several](https://github.com/lyft/python-blessclient#run-a-bless-lambda-in-aws) [great](http://marcyoung.us/post/bless-part1/) [guides](https://www.tastycidr.net/a-practical-guide-to-deploying-netflixs-bless-certificate-authority/) on how to run a BLESS lambda. If you take a moment to skim through these, you'll notice that setting up a successful BLESS deployment requires thorough knowledge of AWS Lambda and IAM. Even then, you'll probably spend hours digging through CloudWatch logs (and who likes doing that).

To further simplify this process, we've put together a terraform [provider](https://github.com/chanzuckerberg/terraform-provider-bless) and [module](https://github.com/chanzuckerberg/cztack/tree/master/bless-ca) to automate BLESS deployments.

### Enabling shell completion
#### bash
##### Linux
```
# Might need to install bash-completion on CentOS
yum install bash-completion
# install completion
echo "source <(blessclient completion bash)" >> ~/.bashrc
```

##### Mac
```
## If running Bash 3.2 included with macOS
brew install bash-completion
## or, if running Bash 4.1+
brew install bash-completion@2

# install completion
blessclient completion bash > $(brew --prefix)/etc/bash_completion.d/blessclient
```

#### zsh
You can add the file generated by `blessclient completion zsh` to a directory in your $fpath.

## Contributing
Contributions and ideas are welcome! Please don't hesitate to open an issue or send a pull request.
This project is governed under the [Contributor Covenant](https://www.contributor-covenant.org/version/1/4/code-of-conduct) code of conduct.
