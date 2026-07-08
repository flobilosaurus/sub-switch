---
date: 2026-07-08T19:00:36.860640+00:00
git_commit: ""
branch: ""
topic: "Go MVP bootstrap for sub-switch"
tags: [plan, go, cli, wrappers, subscriptions]
status: ready
---

# PLAN: Go MVP Bootstrap for sub-switch

Create the initial open-source Go implementation of `sub-switch`: a CLI that selects an allowed subscription/profile from the current folder, launches agent CLIs with profile-isolated XDG directories, and installs PATH wrapper executables so direct commands like `pi` implicitly go through `sub-switch`.

Sandbox/Docker support is intentionally out of scope for this MVP, but the architecture should leave a clear future integration point.

## Acceptance Criteria

- A Go module exists with standard build and test tooling.
- `sub-switch init` creates a local YAML config file.
- Config supports project path rules and per-agent profile names.
- Config supports explicit real command paths for supported agents.
- Unknown folders are denied by default.
- Longest-prefix project path matching is implemented.
- `sub-switch which <agent>` shows the selected project/profile for the current folder.
- `sub-switch run <agent> -- ...` launches the configured real agent binary.
- Launching avoids wrapper recursion by using explicit configured real command paths.
- Arguments are forwarded to the real agent command.
- The launched process receives profile-specific `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, and `XDG_DATA_HOME`.
- `sub-switch run <agent>` prints a startup confirmation banner before launching by default.
- `sub-switch run <agent> --quiet` suppresses the startup banner.
- If no project rule matches, the agent is not launched and a clear denied message is printed.
- `sub-switch install-wrappers --dir <path>` creates executable wrappers for `pi`, `claude`, `codex`, and `opencode`.
- Generated wrappers forward all arguments to `sub-switch run <agent>`.
- Wrapper installation creates missing wrappers and updates managed wrappers.
- Wrapper installation refuses to overwrite unrelated files unless `--force` is used.
- `sub-switch doctor` detects wrapper/PATH issues that can happen after reinstalling `pi`, `opencode`, or other agents.
- Unit tests cover config loading, path matching, deny behavior, env construction, launching with fake commands, startup banner behavior, and wrapper generation.
- README documents install, bootstrap, config, wrapper usage, safety behavior, doctor checks, and MVP limitations.

## Technical Key Decisions and Tradeoffs

1. **Language:** Go.
   - Why: The MVP is a system CLI focused on config parsing, path handling, environment construction, and process launching; Go provides simple static binary distribution and a straightforward standard library.
   - Impact: The repository uses Go modules, Go tests, and can later add GoReleaser/Homebrew distribution.

2. **CLI framework:** Cobra.
   - Why: The command tree includes subcommands such as `init`, `run`, `which`, `install-wrappers`, and `doctor`.
   - Impact: Commands live under `cmd/sub-switch` and delegate implementation to `internal/*` packages.

3. **Config format:** YAML.
   - Why: The user-facing config is hierarchical and should be readable/editable by humans.
   - Impact: Use `gopkg.in/yaml.v3` or an equivalent YAML package for config load/save.

4. **MVP launch mode:** Direct command execution only.
   - Why: The user explicitly deferred sandbox support from the MVP.
   - Impact: No Docker or existing `sandbox run` integration is implemented yet; docs mention it as future work.

5. **Profile isolation:** XDG env vars only.
   - Why: This isolates config/cache/data while avoiding the broad side effects of replacing `HOME`.
   - Impact: The launcher sets `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, and `XDG_DATA_HOME` for the child process.

6. **Implicit usage:** Generated wrapper executables, not shell aliases.
   - Why: Wrapper executables work across shells and support direct commands such as `pi` without relying on alias setup.
   - Impact: MVP includes `install-wrappers` and `doctor` checks for PATH/wrapper safety.

7. **Real binary resolution:** Explicit configured command paths.
   - Why: A wrapper named `pi` can shadow the real `pi`; explicit command paths avoid recursive self-invocation.
   - Impact: Config validation and `doctor` verify that configured commands exist and are not managed wrappers.

8. **User-visible safety:** Startup banner enabled by default.
   - Why: The banner proves `sub-switch` is active and shows the selected profile before any agent starts.
   - Impact: Launcher prints a compact banner unless `--quiet` is set.

## Current State

The repository currently has no Go application code and is not a Git repository. Existing work consists of research documentation under `docs/agents/research/` describing the desired subscription switching approach.

Current file shape:

```text
sub-switch/
└── docs/
    └── agents/
        └── research/
            └── 2026-07-08-subscription-switching-agents.md
```

There is no executable, config schema, resolver, launcher, wrapper installer, or test suite yet.

## Desired End State

Target MVP architecture:

```text
sub-switch CLI
├── init
│   └── writes starter YAML config
├── which <agent>
│   └── cwd -> longest-prefix project -> profile
├── run <agent> [--quiet] -- [args...]
│   ├── resolve profile from cwd
│   ├── build profile-specific XDG env
│   ├── print startup banner
│   └── exec configured real command
├── install-wrappers --dir <path> [--force]
│   └── writes managed pi/claude/codex/opencode wrappers
└── doctor
    ├── validates config
    ├── validates real command paths
    ├── detects wrapper recursion risk
    └── detects PATH shadowing problems
```

Runtime flow:

```text
pi wrapper
  |
  v
sub-switch run pi -- "$@"
  |
  v
load config + inspect cwd
  |
  v
match project by longest prefix
  |
  v
select profile for pi
  |
  v
set XDG_CONFIG_HOME / XDG_CACHE_HOME / XDG_DATA_HOME
  |
  v
print: [sub-switch] pi -> profile company-a (/path/to/project)
  |
  v
launch configured real pi binary
```

Example config shape:

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
  - path: /Users/florian/work/company-a
    profiles:
      pi: company-a
      claude: company-a
      codex: company-a
      opencode: company-a
  - path: /Users/florian/work/client-b
    profiles:
      pi: client-b
      claude: client-b
```

Profile XDG directories:

```text
~/.config/sub-switch/config.yaml
~/.local/state/sub-switch/audit.log        # optional/future
~/.local/share/sub-switch/profiles/<profile>/<agent>/config
~/.cache/sub-switch/profiles/<profile>/<agent>/cache
~/.local/share/sub-switch/profiles/<profile>/<agent>/data
```

For a selected profile `company-a` and agent `pi`, child env receives:

```text
XDG_CONFIG_HOME=$HOME/.local/share/sub-switch/profiles/company-a/pi/config
XDG_CACHE_HOME=$HOME/.cache/sub-switch/profiles/company-a/pi/cache
XDG_DATA_HOME=$HOME/.local/share/sub-switch/profiles/company-a/pi/data
```

## Abstractions and Code Reuse

New project layout:

- `go.mod` - Go module definition.
- `cmd/sub-switch/main.go` - CLI entrypoint.
- `internal/cli/root.go` - Cobra root command wiring.
- `internal/cli/init.go` - `init` command.
- `internal/cli/which.go` - `which` command.
- `internal/cli/run.go` - `run` command.
- `internal/cli/wrappers.go` - `install-wrappers` command.
- `internal/cli/doctor.go` - `doctor` command.
- `internal/config/config.go` - YAML schema, load/save, defaults, validation.
- `internal/resolver/resolver.go` - cwd normalization and longest-prefix project matching.
- `internal/profile/env.go` - XDG profile directory and env construction.
- `internal/launcher/launcher.go` - real command execution and argument forwarding.
- `internal/wrappers/wrappers.go` - managed wrapper file generation and overwrite policy.
- `internal/doctor/doctor.go` - config, command, PATH, and wrapper diagnostics.
- `examples/config.yaml` - example config.
- `README.md` - install/bootstrap/usage docs.

Core abstractions:

- `config.Config` - loaded YAML config.
- `config.AgentConfig` - real command path for an agent.
- `config.ProjectRule` - project path and per-agent profiles.
- `resolver.Selection` - resolved agent, cwd, project path, and profile.
- `profile.Env` - XDG env values and created directories.
- `launcher.CommandSpec` - command path, args, env, cwd, quiet/banner behavior.
- `wrappers.ManagedWrapper` - generated wrapper metadata and content.
- `doctor.CheckResult` - status, severity, message, and optional remediation.

## Logging & Observability

The MVP uses user-facing CLI output rather than a full logging system.

Startup banner example:

```text
[sub-switch] pi -> profile company-a (/Users/florian/work/company-a/project)
```

Denied example:

```text
[sub-switch] denied: no project rule matches /Users/florian/Downloads
```

Missing profile example:

```text
[sub-switch] denied: project /Users/florian/work/client-b has no profile for opencode
```

Doctor output examples:

```text
ok      config loaded: /Users/florian/.config/sub-switch/config.yaml
ok      pi command exists: /opt/homebrew/bin/pi
warn    active pi resolves to /opt/homebrew/bin/pi, expected wrapper in /Users/florian/.local/bin
error   configured command for opencode does not exist: /opt/homebrew/bin/opencode
error   pi command points to a managed sub-switch wrapper; this would recurse
```

## Implementation

### Phase 1: Project Bootstrap

Dependencies: None.

Create the Go project skeleton, CLI entrypoint, and baseline developer tooling.

**Tasks**:
- [x] Create `go.mod` with the chosen module path and Go version.
- [x] Add Cobra dependency and create `cmd/sub-switch/main.go`.
- [x] Add `internal/cli/root.go` with root command metadata and a `--version` flag.
- [x] Add a placeholder `README.md` with project description and MVP scope.
- [x] Add `.gitignore` for Go build artifacts, temp files, and local config fixtures.
- [x] Add `LICENSE` with the MIT license.
- [x] Add GitHub Actions workflow for `go test ./...`, `go vet ./...`, and `go build ./cmd/sub-switch`.
- [x] Add baseline tests for root command construction.
- [x] Add a `Makefile` or documented commands for `go test`, `go vet`, and `go build`.

**Automated Verification**:
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go build ./cmd/sub-switch` succeeds.
- [ ] GitHub Actions workflow syntax is valid and runs `go test`, `go vet`, and `go build`.
- [ ] `./sub-switch --help` or `go run ./cmd/sub-switch --help` prints command help.

### Phase 2: Config, Init, and Resolver

Dependencies: Phase 1.

Implement YAML config loading/saving, default deny behavior, project path matching, and the `which` command.

**Tasks**:
- [x] Add `internal/config/config.go` with YAML structs for `default`, `ui`, `agents`, and `projects`.
- [x] Implement config defaults, including `default: deny` and `ui.startup_banner: true`.
- [x] Implement config path resolution, defaulting to `$XDG_CONFIG_HOME/sub-switch/config.yaml` or `~/.config/sub-switch/config.yaml`.
- [x] Implement `sub-switch init [--force]` to create parent directories and a starter config file, refusing to overwrite by default and overwriting only with `--force`.
- [x] Add `--config <path>` global flag for tests and advanced usage.
- [x] Add config validation for supported default policy values and project/profile structure.
- [x] Add path normalization and `~` expansion where config accepts filesystem paths.
- [x] Add `internal/resolver/resolver.go` with exact/child path matching and longest-prefix selection.
- [x] Implement denied results for no matching project and missing agent profile.
- [x] Implement `sub-switch which <agent>` to print selected project/profile or denied reason.
- [x] Document initial config format in `README.md` and add `examples/config.yaml`.

**Automated Verification**:
- [ ] Config tests load valid YAML and apply defaults.
- [ ] Config tests reject invalid YAML and invalid default policies.
- [ ] `init` command tests create a config in a temp location, refuse overwrite by default, and overwrite with `--force`.
- [ ] Resolver tests cover exact match, child directory match, longest-prefix match, no match denial, and missing agent profile denial.
- [ ] `which` command tests cover allowed and denied output using temp configs and temp working directories.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.

**Manual Verification**:
- [ ] Run `go run ./cmd/sub-switch init --config /tmp/sub-switch-config.yaml` and confirm a starter YAML file is created.
- [ ] Run `go run ./cmd/sub-switch --config /tmp/sub-switch-config.yaml which pi` from a configured folder and confirm the selected profile is shown.

### Phase 3: Profile XDG Environment and Direct Launcher

Dependencies: Phase 2.

Implement `run`, profile-specific XDG env construction, real command execution, argument forwarding, startup banner, and quiet mode.

**Tasks**:
- [x] Add `internal/profile/env.go` to compute profile-specific config/cache/data directories for `<profile>/<agent>`.
- [x] Ensure profile XDG directories are created before launching.
- [x] Implement env merging that preserves unrelated environment variables and overrides `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, and `XDG_DATA_HOME`.
- [x] Add `internal/launcher/launcher.go` to execute configured real command paths with args, cwd, and env.
- [x] Implement `sub-switch run <agent> [--quiet] -- [args...]`.
- [x] Configure `run` argument parsing so all arguments after `--` are passed through unchanged, including flags such as `--help`.
- [x] Ensure `run` resolves the current folder, denies before launch when policy fails, and validates configured command paths.
- [x] Ensure `run` avoids recursion by refusing configured command paths that point to managed sub-switch wrappers.
- [x] Print startup banner before launch when `ui.startup_banner` is true and `--quiet` is false.
- [x] Forward all arguments after `--` to the configured real command.
- [x] Add fake-command test fixtures that print received args and env for launcher tests.
- [x] Update README with `run`, `--quiet`, banner, and XDG isolation behavior.

**Automated Verification**:
- [ ] Profile env tests assert generated `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, and `XDG_DATA_HOME` paths.
- [ ] Profile env tests assert expected directories are created.
- [ ] Launcher tests execute a fake command and verify argument forwarding.
- [ ] Launcher tests verify selected XDG env vars are visible to the fake command.
- [ ] Launcher tests verify no process is started for no-match and missing-profile denials.
- [ ] Launcher tests verify missing configured command path fails clearly.
- [ ] Launcher tests verify managed-wrapper recursion paths are refused.
- [ ] CLI tests verify startup banner appears by default.
- [ ] CLI tests verify `--quiet` suppresses the startup banner.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.

**Manual Verification**:
- [ ] Configure a temp project and fake agent script, then run `go run ./cmd/sub-switch --config <temp-config> run pi -- --version` and confirm banner plus fake command output.
- [ ] Run the same command with `--quiet` and confirm the banner is suppressed.

### Phase 4: Wrapper Installer

Dependencies: Phase 3.

Implement generated executable wrappers for implicit direct command usage.

**Tasks**:
- [x] Add `internal/wrappers/wrappers.go` with supported wrapper agent names: `pi`, `claude`, `codex`, `opencode`.
- [x] Define a managed wrapper marker comment so `sub-switch` can distinguish managed wrappers from unrelated files.
- [x] Generate POSIX shell wrapper content that calls `sub-switch run <agent> -- "$@"`.
- [x] Ensure generated wrappers use an absolute path to the current `sub-switch` binary when possible.
- [x] Set executable file permissions on generated wrappers.
- [x] Refuse to overwrite unrelated existing files by default.
- [x] Allow `--force` to overwrite existing files.
- [x] Allow rerunning installer to repair/update managed wrappers idempotently.
- [x] Implement `sub-switch install-wrappers --dir <path> [--force]`.
- [x] Print a summary of created, updated, skipped, and refused wrappers.
- [x] Update README with wrapper installation, PATH requirements, and reinstall repair workflow.

**Automated Verification**:
- [ ] Wrapper tests create all expected wrapper files in a temp directory.
- [ ] Wrapper tests verify generated files are executable.
- [ ] Wrapper tests verify wrapper content includes the managed marker and forwards `"$@"`.
- [ ] Wrapper tests verify existing unrelated files are not overwritten by default.
- [ ] Wrapper tests verify `--force` overwrites existing files.
- [ ] Wrapper tests verify managed wrappers are updated idempotently.
- [ ] CLI tests verify `install-wrappers --dir <temp>` reports created wrappers.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.

**Manual Verification**:
- [ ] Run `go run ./cmd/sub-switch install-wrappers --dir /tmp/sub-switch-bin` and confirm wrapper files exist and are executable.
- [ ] Put `/tmp/sub-switch-bin` first in `PATH`, run `pi --help` with a fake configured real command, and confirm the startup banner appears before fake command output.

### Phase 5: Doctor Checks and Documentation

Dependencies: Phases 2, 3, and 4.

Add diagnostics that help users detect broken wrappers, PATH order problems, missing real binaries, and recursion risks, especially after reinstalling agent CLIs.

**Tasks**:
- [x] Add `internal/doctor/doctor.go` with check result types and severity levels: ok, warn, error.
- [x] Implement config load/validation checks.
- [x] Implement configured real command existence checks for each configured agent.
- [x] Implement check that real command paths are not managed sub-switch wrappers.
- [x] Implement wrapper existence checks for a user-provided or configured wrapper directory.
- [x] Implement PATH lookup checks that compare active `which <agent>`/`exec.LookPath` result against expected wrapper path.
- [x] Implement PATH order warnings when wrapper directory is not before real binary directory.
- [x] Implement `sub-switch doctor [--wrapper-dir <path>]` CLI output.
- [x] Ensure doctor exits non-zero when any error-level check fails.
- [x] Expand README with full new-PC bootstrap instructions.
- [x] Document what happens after reinstalling `pi`, `opencode`, or other agents and how to repair wrappers.
- [x] Document MVP limitations, especially that sandbox support is not included yet.
- [x] Add a short future architecture note for `sub-switch sandbox run <agent>`.

**Automated Verification**:
- [ ] Doctor tests report ok for valid config, existing fake command, and valid wrappers.
- [ ] Doctor tests report error for missing configured real command.
- [ ] Doctor tests report error for real command path pointing to a managed wrapper.
- [ ] Doctor tests report warn when active PATH lookup resolves to a real binary before the wrapper.
- [ ] Doctor tests report missing wrapper when wrapper file does not exist.
- [ ] CLI tests verify `doctor` exits non-zero on error-level findings.
- [ ] README examples are consistent with command names and config schema.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go build ./cmd/sub-switch` succeeds.

**Manual Verification**:
- [ ] Simulate a reinstalled `pi` by putting a fake real binary before the wrapper in `PATH`, run `sub-switch doctor`, and confirm it warns that `pi` bypasses the wrapper.
- [ ] Run the documented bootstrap flow from README against temp directories and fake agent binaries.

## Implementation Notes

- Implemented the MVP Go code, tests, examples, CI workflow, and README documentation.
- Added `mise.toml` to install Go 1.22.12 and provide `test`, `vet`, `build`, and `check` tasks.
- Automated verification passed via `mise run check` after generating `go.sum` with `mise exec -- go mod tidy`.

## References

- `docs/agents/research/2026-07-08-subscription-switching-agents.md` - research and sketches for folder-based subscription switching, sandbox follow-up, bootstrapping, and open-source/private boundaries.
