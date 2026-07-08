---
date: 2026-07-08T19:26:18.456851+00:00
git_commit: da696101a560bc8b357b14928d82fb7784c255e6
branch: main
topic: "Wrapper end-to-end integration tests"
tags: [plan, cli, wrappers, integration-tests]
status: ready
---

# PLAN: Wrapper End-to-End Integration Tests

Add process-level integration tests for generated agent wrappers. The tests should prove that a user invoking a wrapper command such as `pi` reaches the `sub-switch run` path, resolves policy from the current working directory, and starts only the configured real agent binary with profile-isolated XDG directories.

## Acceptance Criteria

- `go test ./internal/cli` builds a temporary `sub-switch` binary for wrapper end-to-end tests.
- Tests install wrappers into a temporary directory via the built `sub-switch install-wrappers --dir <dir>` command.
- Invoking the temporary `pi` wrapper starts a fake real agent binary.
- The fake agent records and the test asserts forwarded args, working directory, and XDG env values.
- Running the wrapper from an unmatched directory returns a denial and does not invoke the fake agent.
- Configuring `agents.pi.command` to a managed wrapper returns the managed-wrapper recursion/refusal error before launching.
- Wrapper end-to-end tests skip on Windows because generated wrappers are POSIX `/bin/sh` scripts.
- `mise run check` passes.

## Technical Key Decisions and Tradeoffs

1. **Test location:** Add the tests under `internal/cli`.
   - Why: This matches the existing CLI command tests and the requested location.
   - Impact: New helpers and tests can live next to `commands_test.go`, but should avoid interfering with its in-process Cobra helper.

2. **Test style:** Execute a real temporary `sub-switch` binary and generated wrapper scripts.
   - Why: In-process Cobra tests do not prove that `os.Executable()`, generated wrapper content, shell argument forwarding, or process boundaries work together.
   - Impact: Tests need to call `go build` into `t.TempDir()` and then run commands with `os/exec`.

3. **Fake real agents:** Use temporary shell scripts as virtual agent binaries.
   - Why: They are simple, deterministic, and can record argv/env/cwd without depending on real `pi`, `claude`, `codex`, or `opencode` installations.
   - Impact: Tests are Unix-only and must skip on Windows.

4. **Safety coverage:** Include happy path, unmatched-project denial, and managed-wrapper recursion guard.
   - Why: These cover both the primary user flow and the core safety model.
   - Impact: The suite should include multiple focused tests rather than a single broad scenario.

## Current State

Current command flow for `sub-switch run`:

```text
sub-switch run pi -- [args]
  |
  v
internal/cli/run.go
  - load config
  - resolve cwd + agent to project/profile
  - deny if no matching project/profile
  - build XDG profile env
  - print banner unless --quiet
  |
  v
internal/launcher.Run
  - ValidateCommand(command)
  - refuse missing/dir/managed wrapper commands
  - exec configured real command with args/env/cwd
```

Existing tests already cover part of this flow in-process:

- `internal/cli/commands_test.go` has a fake-agent test that calls `NewRootCommand().Execute()` directly and asserts forwarded args plus `XDG_CONFIG_HOME`.
- `internal/wrappers/wrappers_test.go` verifies generated wrapper files contain the managed marker and forward `"$@"`.
- `internal/launcher/launcher_test.go` verifies `ValidateCommand` refuses a managed wrapper.

What is not yet covered is the full installed-wrapper path:

```text
temp sub-switch binary
  |
  v
sub-switch install-wrappers --dir <tmp>/wrappers
  |
  v
<tmp>/wrappers/pi --version
  |
  v
#!/bin/sh wrapper execs:
  <tmp>/sub-switch run pi -- "$@"
  |
  v
fake real agent binary
```

## Desired End State

`internal/cli` contains wrapper integration tests that exercise the user-facing PATH-wrapper behavior without relying on any real agent installation.

Target test topology:

```text
t.TempDir()
├── sub-switch-test          # built from ./cmd/sub-switch
├── wrappers/
│   ├── pi                   # generated managed wrapper
│   ├── claude
│   ├── codex
│   └── opencode
├── real-agents/
│   └── fake-pi              # virtual real agent shell script
├── project/                 # matching cwd
├── unknown/                 # denied cwd
├── config.yaml              # points pi.command at fake-pi or wrapper
└── records/
    └── pi.env               # fake-agent output for assertions
```

## Abstractions and Code Reuse

- `internal/cli`
  - `commands_test.go` or a new `wrappers_e2e_test.go` - add wrapper process-level tests and local helpers.
    - `buildTestSubSwitch(t)` - builds `./cmd/sub-switch` to a temp path.
    - `runCmd(t, dir, env, exe, args...)` - executes subprocesses and returns combined output/error.
    - `writeFakeAgent(t, path, recordFile)` - writes a POSIX shell script that records args, cwd, and XDG env.
    - `writeConfig(t, path, projectPath, commandPath, profile)` - writes minimal YAML config for the test scenario.

Prefer a new `internal/cli/wrappers_e2e_test.go` to keep process-level integration tests separate from concise command tests.

## Logging & Observability

No product logging changes are planned.

Test observability should come from fake-agent record files and subprocess combined output. Example fake-agent record content:

```text
args:--version --json
cwd:/tmp/.../project
XDG_CONFIG_HOME:/Users/.../.local/share/sub-switch/profiles/company/pi/config
XDG_CACHE_HOME:/Users/.../.cache/sub-switch/profiles/company/pi/cache
XDG_DATA_HOME:/Users/.../.local/share/sub-switch/profiles/company/pi/data
```

For denial and recursion tests, assert on `sub-switch` error/output text and absence of the fake-agent record file.

## Implementation

### Phase 1: Add Wrapper E2E Test Harness and Happy Path

Dependencies: None

Create the reusable test harness in `internal/cli` and add the first vertical scenario: generated `pi` wrapper launches the fake configured agent with correct args/env/cwd.

**Tasks**:
- [x] Create `internal/cli/wrappers_e2e_test.go` in package `cli`.
- [x] Add a Windows guard at the start of each wrapper E2E test or in a shared helper:
  ```go
  if runtime.GOOS == "windows" { t.Skip("POSIX shell wrappers") }
  ```
- [x] Add `repoRoot(t)` helper for tests running from `internal/cli`, resolving the repository root as `filepath.Clean(filepath.Join(packageDir, "..", ".."))` or equivalent.
- [x] Add `buildTestSubSwitch(t)` helper that runs:
  ```sh
  go build -o <temp>/sub-switch-test ./cmd/sub-switch
  ```
  with `cmd.Dir` set to the repository root and fails the test with combined build output on error.
- [x] Add `writeFakeAgent(t, path, recordFile)` helper that writes an executable shell script recording `args:$@`, `pwd`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, and `XDG_DATA_HOME`.
- [x] Add `writeConfig(t, cfg, projectPath, realAgentPath, profile)` helper for a minimal `default: deny` config with `agents.pi.command` and one `projects` rule.
- [x] Add a happy-path test that:
  - [x] Creates temp `home`, `xdg-config`, `project`, `wrappers`, `records`, and fake-agent paths.
  - [x] Builds the temp `sub-switch` binary.
  - [x] Runs `<sub-switch-test> install-wrappers --dir <wrappers>`.
  - [x] Writes config to `<xdg-config>/sub-switch/config.yaml` pointing `pi` to the fake real agent.
  - [x] Executes `<wrappers>/pi --version --json` from the matching project directory with `HOME=<home>` and `XDG_CONFIG_HOME=<xdg-config>` in the environment.
  - [x] Asserts the fake record file exists.
  - [x] Asserts forwarded args include `--version --json` in order.
  - [x] Asserts recorded cwd equals the temp project path.
  - [x] Asserts recorded XDG env is rooted under the temp `home` and contains `/profiles/company/pi/config`, `/profiles/company/pi/cache`, and `/profiles/company/pi/data`.
- [x] Do not pass `--config` through the wrapper; it would be interpreted as an agent arg after `run pi -- "$@"`. Use test-specific `HOME` and `XDG_CONFIG_HOME` instead.

**Automated Verification**:
- [x] `go test ./internal/cli -run TestWrapper.*HappyPath -count=1` passes.
- [x] `go test ./internal/cli` passes.

### Phase 2: Add Denial E2E Coverage

Dependencies: Phase 1

Extend the wrapper E2E suite to prove that unknown directories are denied before any fake agent process starts.

**Tasks**:
- [x] Add a denial test that reuses the wrapper harness but runs `<wrappers>/pi --version` from a temp directory outside the configured project path.
- [x] Configure the wrapper run to load the test config via isolated `HOME` and `XDG_CONFIG_HOME` without changing generated wrapper contents.
- [x] Assert the command exits non-zero.
- [x] Assert combined output contains `[sub-switch] denied` and mentions no matching project rule.
- [x] Assert the fake-agent record file does not exist.

**Automated Verification**:
- [x] `go test ./internal/cli -run TestWrapper.*Denied -count=1` passes.
- [x] `go test ./internal/cli` passes.

### Phase 3: Add Managed-Wrapper Recursion Guard E2E Coverage

Dependencies: Phase 1

Add an end-to-end regression test for the safety rule that a configured real command must not point to a generated managed wrapper.

**Tasks**:
- [x] Add a recursion-guard test that installs wrappers and writes a config where `agents.pi.command` points to `<wrappers>/pi` or another generated managed wrapper path.
- [x] Run the generated `pi` wrapper from a matching project directory.
- [x] Assert the command exits non-zero.
- [x] Assert combined output contains the managed-wrapper recursion error text, e.g. `configured command points to a managed sub-switch wrapper`.
- [x] Assert no fake-agent record file exists, if a fake-agent path is present for control purposes.
- [x] Keep a timeout or context on subprocess execution if needed to guard against accidental recursion hangs.

**Automated Verification**:
- [x] `go test ./internal/cli -run TestWrapper.*Recursion -count=1` passes.
- [x] `go test ./internal/cli` passes.
- [x] `mise run check` passes.

## Implementation Notes

During implementation, document user feedback, problems, and decisions here.

Known test-design detail: generated wrappers do not include `--config`, so wrapper E2E tests should set `XDG_CONFIG_HOME` to a temporary directory containing `sub-switch/config.yaml`. Passing `--config` to `pi` would become an agent argument after `run pi -- "$@"` and would not configure `sub-switch`.

Set `HOME` to a temp directory too. `config.DefaultPath` uses `XDG_CONFIG_HOME`, but `profile.BuildForCurrentUser` uses `os.UserHomeDir()`, so a temp `HOME` keeps generated profile directories out of the real user home.

## References

- `internal/cli/run.go` - `run` command policy resolution and launch path.
- `internal/cli/wrappers.go` - `install-wrappers` command uses `os.Executable()` for generated wrapper target.
- `internal/wrappers/wrappers.go` - wrapper content, supported agents, managed marker.
- `internal/launcher/launcher.go` - command validation and `exec.Command` launch.
- `internal/cli/commands_test.go` - existing fake-agent CLI test pattern.
- `internal/wrappers/wrappers_test.go` - existing wrapper generation unit tests.
- `internal/launcher/launcher_test.go` - existing recursion guard unit test.
- `README.md` - documented wrapper and XDG behavior.
