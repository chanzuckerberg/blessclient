# blessclient
[![codecov](https://codecov.io/gh/chanzuckerberg/blessclient/branch/master/graph/badge.svg)](https://codecov.io/gh/chanzuckerberg/blessclient) [![Gitter chat](https://badges.gitter.im/gitterHQ/gitter.png)](https://gitter.im/chanzuckerberg/blessclient)

**Please note**: If you believe you have found a security issue, _please responsibly disclose_ by contacting us at [security@chanzuckerberg.com](mailto:security@chanzuckerberg.com).

----

Inspiration for this project comes from [lyft/python-blessclient](https://github.com/lyft/python-blessclient).
We decided to write in Go because it is much easier to distribute a statically linked binary to a large team than having to deal with python environments. Some features from [lyft/python-blessclient](https://github.com/lyft/python-blessclient) are currently missing but will be added over time while others are purposefully excluded.

## Versions
We are currently in the process of releasing a new major version of blessclient that will replace [netflix/bless](https://github.com/Netflix/bless) for a version that relies on federated identity.

### v0.x.x - deprecation notice
This version will soon be deprecated.
For the time-being `brew install blessclient` will still point to `v0.x.x`

You can use homebrew to install with
```
brew tap chanzuckerberg/tap
brew install blessclient@1
```

We will keep a v0 branch around for high priority fixes until migrated fully to `v1.x.x`.

### v1.x.x - in active development
More to come.

## Install

### Linux + macOS
We recommend using [homebrew](https://brew.sh/):
```
brew tap chanzuckerberg/tap
brew install blessclient@1
```

### WSL
We have tested on WSL Ubuntu-18. A couple extra steps are required:
```
sudo apt update && sudo apt install xdg-utils
brew tap chanzuckerberg/tap
brew install blessclient@1
```

## Usage

At a high level:
1. [Install](#install) blessclient
1. If you don't have an SSH key, generate one with `ssh-keygen -t rsa -b 4096`
1. [Import](#import-config) or generate a blessclient config. You can find an example config [here](examples/config.yml).
1. Run `blessclient run` and make sure there are no errors
1. Modify your [ssh config](#sshconfig) to be bless compatible
1. ssh, scp, rsync as you normally would

### Config

By default, `blessclient` looks for configs in `~/.blessclient/config.yml`. You can always override this `blessclient run -c /my/new/config.yml`
Some more information on the config can be found [here](pkg/config/config.go).

There is a built-in method to facilitate the generation of blessclient configs:

#### Import-config

A few options here:
- `blessclient import-config git@github.com:/..../teamA/blessconfig.yml`
- `blessclient import-config https://www.github.com/..../teamA/blessconfig.yml`
- `blessclient import-config /home/user/.../teamA/blessconfig.yml`
- `blessclient import-config s3::https://s3.amazonaws.com/bucket/teamA/blessconfig.yml`

This command uses [go-getter](https://github.com/hashicorp/go-getter) to fetch a config and thus supports any source that [go-getter](https://github.com/hashicorp/go-getter#supported-protocols-and-detectors) supports.

You can see an example config with dummy values [here](examples/config.yml). Download the example, modify the values, and `blessclient import-config <path>` it to get started.

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

## Commands

### run
`run` will run blessclient and attempt to fetch an SSH certificate from the CA. It requires blessclient to be properly configured beforehand.

### import-config
`import-config` will import blessclient configuration from a remote location and configure your local blessclient.

### token
`token` will print, json formatted, your oauth2/oidc id_token and access_token. This command requires blessclient to be properly configured beforehand. This command is not typically part of a common workflow.

The output will be written to stdout. The output is json formatted and looks like
```json
{
  "version": 1,
  "id_token": "<string>",
  "access_token": "<string>",
  "expiry": "2020-07-20T12:18:02-04:00"
}
```
When running this command, no other output will be written to stdout.

### version
`version` will print blessclient's version.

## Other
### Deploying BLESS
There are already [several](https://github.com/lyft/python-blessclient#run-a-bless-lambda-in-aws) [great](http://marcyoung.us/post/bless-part1/) [guides](https://www.tastycidr.net/a-practical-guide-to-deploying-netflixs-bless-certificate-authority/) on how to run a BLESS lambda. If you take a moment to skim through these, you'll notice that setting up a successful BLESS deployment requires thorough knowledge of AWS Lambda and IAM. Even then, you'll probably spend hours digging through CloudWatch logs (and who likes doing that).

To further simplify this process, we've put together a terraform [provider](https://github.com/chanzuckerberg/terraform-provider-bless) and [module](https://github.com/chanzuckerberg/cztack/tree/master/bless-ca) to automate BLESS deployments.

## Contributing
Contributions and ideas are welcome! Please don't hesitate to open an issue, join our [gitter chat room](https://gitter.im/chanzuckerberg/blessclient), or send a pull request.

Go version >= 1.12 required.

## Code of Conduct

This project adheres to the Contributor Covenant [code of conduct](https://github.com/chanzuckerberg/.github/blob/master/CODE_OF_CONDUCT.md).
By participating, you are expected to uphold this code. 
Please report unacceptable behavior to [opensource@chanzuckerberg.com](mailto:opensource@chanzuckerberg.com).
