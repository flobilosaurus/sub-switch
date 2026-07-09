# sub-switch

`sub-switch` selects an allowed subscription/profile for agent CLIs from the current folder and launches the real binary with profile-isolated XDG directories, supported agent state env vars, strict known auth/cloud env scrubbing, and optional profile-specific env values.

The MVP supports configurable agent commands, denies unknown folders by default, installs PATH wrappers for configured agents, and includes `doctor` checks. Sandbox/Docker support is intentionally not included yet.

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

The starter config has no agents or projects; add the agent commands and project profile mappings you want to allow. The default config path is `$XDG_CONFIG_HOME/sub-switch/config.yaml`, or `~/.config/sub-switch/config.yaml` when `XDG_CONFIG_HOME` is unset.

The fastest way to onboard a new folder is simply to run from it:

```sh
cd /path/to/new/project
sub-switch run pi -- --help
```

When the configuration is incomplete and the terminal is interactive, `run` guides you through the missing setup — configuring the agent command, selecting or creating a profile, and adding the current folder — then saves and launches in one step. See [Interactive setup](#interactive-setup) below.

## Config

A complete configuration requires all three pieces for a launch to be allowed:

1. **`agents.<agent>.command`** — the real binary path for each agent.
2. **`projects[].profiles.<agent>`** — maps a folder to a profile name for a given agent.
3. **`profiles.<profile>.<agent>`** — explicitly allows that agent under the profile (and may contain env values).

Example:

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
profiles:
  company-a:
    pi:
      env:
        ANTHROPIC_API_KEY: sk-ant-company-a-example
    claude:
      env:
        CLAUDE_CODE_OAUTH_TOKEN: claude-oauth-token-example
    codex:
      env:
        CODEX_API_KEY: codex-api-key-example
    opencode:
      env:
        OPENAI_API_KEY: openai-key-example
projects:
  - path: /path/to/work/company-a
    profiles:
      pi: company-a
      claude: company-a
      codex: company-a
      opencode: company-a
```

Project rules use exact-or-child matching. If multiple rules match, the longest path prefix wins. If no project matches, the selected project has no profile for the requested agent, or the top-level profile/profile-agent entry is missing, the agent is not launched.

Agent names are command names from the `agents` map. Agent `command` values must point at the real binary, not a generated wrapper, to avoid recursion.

Top-level `profiles:` is the canonical list of profiles. When interactive setup prompts for a profile, only top-level profiles are offered as existing choices. `profiles.<profile>.<agent>` must exist for the agent/profile combination to be allowed — even when no extra env values are needed, an empty entry is required.

To add your current folder to the config manually (alternative to interactive `run` setup), pass one or more agent/profile mappings:

```sh
cd /path/to/work/company-a
sub-switch add-project pi=company-a claude=company-a
```

If an agent is not yet configured, `add-project` resolves it from `PATH` and stores that real command path under `agents`. Use `--force` to replace an existing profile mapping for the current folder.

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

### Interactive setup

When `sub-switch run <agent>` is invoked from an interactive terminal (TTY) and the configuration is incomplete, it guides you through setup instead of simply denying:

```text
$ sub-switch run pi -- --help
[sub-switch] no project rule matches /Users/me/new-project

? Select a profile
  company-a
  personal
  → Create new profile

? Add /Users/me/new-project with pi → company-a? Yes
? Allow pi for profile company-a? Yes

[sub-switch] saved /Users/me/.config/sub-switch/config.yaml
[sub-switch] pi -> profile company-a (/Users/me/new-project)
```

Setup can:
- Auto-detect and configure `agents.<agent>.command` from PATH.
- Select an existing top-level profile or create a new one.
- Add the current folder as a project rule.
- Create missing `profiles.<profile>.<agent>` entries.
- Handle conflicts when re-mapping an already-configured folder.

Accepted changes are saved with mode `0600`, then the original `run` continues. Aborting (Ctrl+C or Esc) saves nothing and launches nothing.

### Non-interactive (non-TTY) behavior

When stdin/stdout are not terminals (wrappers, scripts, CI, editor integrations), `run` never prompts. It fails fast with a clear error:

```text
[sub-switch] denied: no project rule matches /path (run from a terminal to set this up interactively)
```

This keeps wrappers and automated pipelines safe.

## Credential profile isolation

For profile `company-a` and agent `pi`, the child process receives isolated XDG dirs:

```text
XDG_CONFIG_HOME=$HOME/.local/share/sub-switch/profiles/company-a/pi/config
XDG_CACHE_HOME=$HOME/.cache/sub-switch/profiles/company-a/pi/cache
XDG_DATA_HOME=$HOME/.local/share/sub-switch/profiles/company-a/pi/data
```

Unrelated environment variables are preserved. Known auth/token/cloud variables and agent-state override variables are scrubbed/replaced. Profile env values are injected last for non-reserved names, so they can provide credentials such as `ANTHROPIC_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`, `CODEX_API_KEY`, and `OPENAI_API_KEY`.

Agent-specific state:

- `pi`: sets `PI_CODING_AGENT_DIR` and `PI_CODING_AGENT_SESSION_DIR` under the selected profile/agent base dir.
- `claude`: sets `CLAUDE_CONFIG_DIR`; on macOS, Keychain state may still be shared by Claude itself, so prefer profile env such as `CLAUDE_CODE_OAUTH_TOKEN` when isolating auth.
- `codex`: sets `CODEX_HOME`.
- `opencode`: uses isolated XDG dirs; inherited `OPENCODE_CONFIG` and `OPENCODE_CONFIG_DIR` are scrubbed.
- Generic configured agents: receive XDG isolation, auth/state scrubbing, and optional profile env only.

Reserved managed env names cannot be set in profile env: XDG homes, `PI_CODING_AGENT_DIR`, `PI_CODING_AGENT_SESSION_DIR`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `OPENCODE_CONFIG`, and `OPENCODE_CONFIG_DIR`.

### Literal secret warning

`sub-switch` saves config files with mode `0600`, including overwrites, but external editors/tools may change permissions. Verify permissions for configs containing credentials. Do not commit or share configs containing secrets. Prefer a local-only config path if credentials are present.

## Wrappers

Install wrapper executables earlier in `PATH`:

```sh
sub-switch install-wrappers pi --dir ~/.local/bin
```

This resolves `pi` from `PATH`, stores it in `agents`, and creates shell wrappers for all configured agents. Each wrapper calls:

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
- Not a full sandbox: `HOME` is not changed, and filesystem outside profile dirs remains accessible to the child process.

Future architecture may add `sub-switch sandbox run <agent>` to resolve the host project path, mount only the selected profile into a container, and then run the agent inside the sandbox.
