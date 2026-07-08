# sub-switch

`sub-switch` selects an allowed subscription/profile for agent CLIs from the current folder and launches the real binary with profile-isolated XDG directories.

The MVP supports `pi`, `claude`, `codex`, and `opencode`, denies unknown folders by default, installs PATH wrappers, and includes `doctor` checks. Sandbox/Docker support is intentionally not included yet.

## Install

Install the latest release to `~/.local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/florian-balling/sub-switch/main/install.sh | sh
```

Use a custom install directory or version:

```sh
curl -fsSL https://raw.githubusercontent.com/florian-balling/sub-switch/main/install.sh | INSTALL_DIR=/usr/local/bin sh
curl -fsSL https://raw.githubusercontent.com/florian-balling/sub-switch/main/install.sh | SUB_SWITCH_VERSION=v0.1.0 sh
```

## Development tools with mise

This project uses [mise](https://mise.jdx.dev/) to install required development tools.

Install and activate mise, then install the pinned tools from `mise.toml`:

```sh
mise install
```

Required tools:

- Go `1.26.4`
- Additional static-analysis Go CLIs declared in `mise.toml` for deeper local checks

Useful mise tasks:

```sh
mise run test              # go test ./...
mise run vet               # go vet ./...
mise run build             # go build ./cmd/sub-switch
mise run check             # test + vet + build
mise run static-analysis   # stricter Go analyzers, not CI-gating by default
mise run security-analysis # dependency license/vulnerability checks
```

Equivalent direct Go commands:

```sh
go test ./...
go vet ./...
go build ./cmd/sub-switch
```

Put the built `sub-switch` binary somewhere on `PATH`.

## Bootstrap

Create a starter config:

```sh
sub-switch init
# or for testing
sub-switch --config /tmp/sub-switch-config.yaml init
```

The default config path is `$XDG_CONFIG_HOME/sub-switch/config.yaml`, or `~/.config/sub-switch/config.yaml` when `XDG_CONFIG_HOME` is unset.

## Config

```yaml
default: deny
ui:
  startup_banner: true
agents:
  pi:
    command: /opt/homebrew/bin/pi
  claude:
    command: /opt/homebrew/bin/claude
  codex:
    command: /opt/homebrew/bin/codex
  opencode:
    command: /opt/homebrew/bin/opencode
projects:
  - path: /path/to/work/company-a
    profiles:
      pi: company-a
      claude: company-a
      codex: company-a
      opencode: company-a
```

Project rules use exact-or-child matching. If multiple rules match, the longest path prefix wins. If no project matches, or the selected project has no profile for the requested agent, the agent is not launched.

Agent `command` values must point at the real binary, not a generated wrapper, to avoid recursion.

## Usage

Show what would be selected:

```sh
sub-switch which pi
```

Run an agent and forward arguments after `--`:

```sh
sub-switch run pi -- --help
sub-switch run pi --quiet -- --version
```

By default, `run` prints a confirmation banner before launching:

```text
[sub-switch] pi -> profile company-a (/path/to/work/company-a)
```

`--quiet` suppresses the banner.

## XDG profile isolation

For profile `company-a` and agent `pi`, the child process receives:

```text
XDG_CONFIG_HOME=$HOME/.local/share/sub-switch/profiles/company-a/pi/config
XDG_CACHE_HOME=$HOME/.cache/sub-switch/profiles/company-a/pi/cache
XDG_DATA_HOME=$HOME/.local/share/sub-switch/profiles/company-a/pi/data
```

Other environment variables are preserved.

## Wrappers

Install wrapper executables earlier in `PATH`:

```sh
sub-switch install-wrappers --dir ~/.local/bin
```

This creates `pi`, `claude`, `codex`, and `opencode` shell wrappers that call:

```sh
sub-switch run <agent> -- "$@"
```

Managed wrappers contain a marker comment. Re-running the installer updates managed wrappers. Existing unrelated files are not overwritten unless `--force` is used.

After reinstalling an agent CLI, ensure the real binary path in config still points to the real command and that your wrapper directory is still before real binary directories in `PATH`.

## Doctor

Run diagnostics:

```sh
sub-switch doctor --wrapper-dir ~/.local/bin
```

Doctor validates config loading, configured command paths, wrapper recursion risks, missing wrappers, and PATH lookup issues where a real binary shadows the wrapper.

## Releasing

Push a version tag to build release binaries and publish a GitHub Release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

Release notes are generated from semantic/conventional commit subjects such as `feat: ...`, `fix: ...`, and `feat!: ...`.

## MVP limitations

- No Docker/sandbox launching yet.
- No profile login/enrollment workflow yet.
- No audit log yet.
- Profile isolation is XDG-only; `HOME` is not changed.

Future architecture may add `sub-switch sandbox run <agent>` to resolve the host project path, mount only the selected profile into a container, and then run the agent inside the sandbox.
