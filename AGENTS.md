# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project summary

`sub-switch` is a Go CLI that selects an allowed subscription/profile for agent CLIs based on the current working directory. It then launches the configured real agent binary with profile-isolated XDG directories.

Supported MVP agents are defined by the `agents` map in config; fresh starter configs intentionally contain no initial agents.

Safety model:

- Unknown folders are denied by default.
- Project matching uses exact-or-child path matching.
- If multiple project rules match, the longest path prefix wins.
- If a matched project has no profile for the requested agent, launch is denied.
- Real agent commands must be explicit configured paths to avoid PATH wrapper recursion.
- Managed wrappers are generated executables for configured agents that call `sub-switch run <agent> -- "$@"`.

Sandbox/Docker support is intentionally out of scope for the MVP, but README notes it as future work.

## Tooling

Use `mise` for development tools.

```sh
mise install
```

Pinned tools are in `mise.toml`:

- Go `1.26.4`
- Additional Go static-analysis tools from analysis-tools.dev suggestions:
  - `nilaway` for nil-panic analysis
  - `gopls` for official Go analyzer checks
  - `deadcode` for unreachable/unused code
  - `shadow` for accidental variable shadowing
  - `fieldalignment` for struct layout checks
  - Staticcheck's `U1000` check for precise unused-code detection
  - `depguard` for import allow/deny rules
  - `gci` for import grouping/formatting
  - `go-licenses` for dependency license checks
  - `nancy` for dependency vulnerability scanning
  - `arch-go` and `go-cleanarch` for optional architecture boundary checks

Preferred verification command:

```sh
mise run check
```

This runs:

```sh
go test ./...
go vet ./...
go build ./cmd/sub-switch
```

Individual required tasks:

```sh
mise run test
mise run vet
mise run build
```

Additional focused static-analysis tasks:

```sh
mise run gci              # import grouping/formatting diff
mise run deadcode         # official Go dead-code analyzer
mise run unused           # precise unused-code analyzer via Staticcheck U1000
mise run nilaway          # nil-panic analyzer
mise run shadow           # shadowed-variable analyzer
mise run fieldalignment   # struct layout suggestions
mise run gopls-check      # gopls analyzers over tracked Go files
mise run licenses         # dependency license checks
mise run nancy            # dependency vulnerability scan
mise run depguard         # import rules, once depguard policy is configured
mise run arch             # arch-go/go-cleanarch, when config files exist
```

Use `mise run static-analysis` for the main non-security analyzer bundle, and `mise run security-analysis` for license/vulnerability checks. These are intentionally stricter and are not part of `mise run check` unless the project decides to gate CI on them.

If dependencies change, run:

```sh
mise exec -- go mod tidy
```

Keep `go.mod` and `go.sum` in sync.

## Repository layout

```text
cmd/sub-switch/main.go        CLI entrypoint
internal/cli/                 Cobra command wiring and CLI tests
internal/config/              YAML schema, defaults, config path resolution, validation
internal/resolver/            cwd -> project/profile resolution
internal/profile/             XDG profile directory/env construction
internal/launcher/            Real command validation and execution
internal/wrappers/            Managed wrapper generation and overwrite policy
internal/doctor/              Config, command, wrapper, and PATH diagnostics
examples/config.yaml          Example user config
README.md                     User-facing docs
mise.toml                     Tool versions and dev tasks
docs/agents/plans/            Implementation plans/state trackers
docs/agents/research/         Background research
```

## Command behavior to preserve

### `sub-switch init [--force]`

Creates starter YAML config at `--config <path>` or default config path:

- `$XDG_CONFIG_HOME/sub-switch/config.yaml`, or
- `~/.config/sub-switch/config.yaml`

It must refuse to overwrite by default and only overwrite with `--force`.

### `sub-switch which <agent>`

Loads config, resolves the current directory, and prints either selected project/profile or a clear denied message.

### `sub-switch run <agent> [--quiet] -- [args...]`

Must:

- Resolve cwd before launching.
- Deny before launching on no matching project or missing profile.
- Validate configured real command exists.
- Refuse configured command paths that point to managed sub-switch wrappers.
- Create profile-specific XDG dirs.
- Preserve unrelated env vars and override only:
  - `XDG_CONFIG_HOME`
  - `XDG_CACHE_HOME`
  - `XDG_DATA_HOME`
- Forward all args after `--` unchanged.
- Print startup banner by default when `ui.startup_banner: true`.
- Suppress banner with `--quiet`.

### `sub-switch install-wrappers <agent> --dir <path> [--force]`

Must resolve `<agent>` from `PATH`, add/update that agent in config, save the config, and generate executable wrappers for all configured agents. It must refuse to add a managed sub-switch wrapper as the real command. Wrappers should include the managed marker and forward `"$@"`.

Do not overwrite unrelated existing files unless `--force` is used. Re-running should update managed wrappers idempotently.

### `sub-switch doctor [--wrapper-dir <path>]`

Should report config loading, configured command existence, managed-wrapper recursion risks, missing wrappers, and PATH lookup issues. It exits non-zero on error-level findings.

## Config schema

Example shape:

```yaml
default: deny
ui:
  startup_banner: true
agents:
  pi:
    command: /opt/homebrew/bin/pi
projects:
  - path: /path/to/work/company-a
    profiles:
      pi: company-a
```

Only `default: deny` is currently supported. Paths may use `~` and should be normalized.

## Testing expectations

When changing code, add or update unit tests in the relevant package. Important coverage areas:

- Config defaults, loading, invalid YAML/policies, init overwrite behavior.
- Resolver exact match, child match, longest-prefix match, denial cases.
- XDG env path construction and directory creation.
- Launcher argument forwarding and env visibility using fake commands.
- Refusal to launch missing commands and managed-wrapper recursion paths.
- CLI banner and `--quiet` behavior.
- Wrapper generation, executable mode, overwrite policy, idempotent update.
- Doctor error/warn behavior.

Before finishing, run:

```sh
mise run check
```

If you cannot run verification, state why clearly in your final response.

## Style and implementation notes

- Keep packages small and focused under `internal/*`.
- Prefer standard library APIs unless a dependency is justified.
- Cobra commands should delegate domain behavior to internal packages.
- Use explicit paths for real agent binaries; never resolve real agents through wrapper names.
- Do not store credentials, personal profile data, or machine-specific private config in the repo.
- Public docs may show example paths/profile names only.
- Keep README user-facing; put agent/developer operational notes here.
