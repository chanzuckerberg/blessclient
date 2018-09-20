# blessclient

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

### Config

By default, `blessclient` looks for configs in `~/.blessclient/config.yml`. You can always override this `blessclient run -c /my/new/config.yml`
Some more information around the config can be found [here](pkg/config/config.go).

#### Init

`blessclient init` will ask you some questions in order to assemble some basic configuration.

#### Import-config
You can also use pre-generated config files.

`blessclient import-config -url http://github.com/..../teamA_blessclient.yml`

This command uses [go-getter](https://github.com/hashicorp/go-getter) to fetch a config and thus supports any source that [go-getter](https://github.com/hashicorp/go-getter) supports.

### Run

Once you have configured blessclient, running is as simple as `blessclient run`. If needed, this command will fetch a new certificate. We do a fair amount of caching to avoid round-trips to the Bless lambda.


### .ssh/config

This is the nice part about blessclient - in general, you can write an ssh config to transparently use blessclient. scp, rsync, etc should all be compatible!

Such an ssh config could look like:

```
Match OriginalHost bastion.foo.com exec "blessclient run"
  IdentityFile ~/.ssh/id_rsa

Host 10.0.*
  ProxyJump bastion.foo.com
  User czi-admin

Host bastion.foo.com
  User czi-admin
```

This ssh config does a couple of interesting things -

- It transparently requests an ssh certificate if needed
- It transparently does a ProxyJump through a bastion host (assuming 10.0.* is an ipblock for machines behind the bastion)
