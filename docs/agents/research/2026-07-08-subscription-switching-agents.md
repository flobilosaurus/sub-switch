---
date: 2026-07-08T18:39:39.699254+00:00
git_commit: ""
branch: ""
topic: "Folder-based subscription switching for pi, codex, claude code, and opencode"
tags: [research, agents, subscriptions, shell-tooling]
status: complete
---

# Research: Folder-based subscription switching for pi, codex, claude code, and opencode

## Research Question
The goal is to create a tool that lets agent CLIs such as pi, codex, claude code, and opencode use the correct subscription/account based on the folder where they are started, to avoid using a subscription that is not allowed for a project.

## Summary
This directory currently contains no application code and is not a Git repository. The research therefore documents implementation sketches rather than existing repository internals.

The core pattern is a launcher/wrapper that resolves the current working directory to a configured subscription profile before starting the target agent. The wrapper can enforce policy by refusing to start when no matching profile exists, by setting agent-specific environment variables, by selecting agent-specific config/cache directories, or by invoking each CLI with profile/account flags where supported.

## Sketch Ideas

### 1. Single `sub-switch` launcher
A command such as:

```sh
sub-switch claude
sub-switch codex
sub-switch pi
sub-switch opencode
```

Flow:

```text
current directory
      |
      v
find nearest rule in config
      |
      v
resolve agent + project -> subscription profile
      |
      v
prepare isolated environment/config dirs
      |
      v
exec real agent CLI
```

Example config:

```yaml
rules:
  - path: ~/work/company-a
    profiles:
      claude: company-a
      codex: company-a
      pi: company-a
      opencode: company-a

  - path: ~/work/client-b
    profiles:
      claude: client-b
      codex: client-b

default: deny
```

### 2. Symlink-style per-agent wrappers
Install commands named like the real tools earlier in `$PATH`:

```text
~/bin/claude   -> sub-switch claude
~/bin/codex    -> sub-switch codex
~/bin/pi       -> sub-switch pi
~/bin/opencode -> sub-switch opencode
```

This makes normal usage safe because typing `claude` first enters the subscription switcher. The switcher then finds and executes the real binary from a known path.

### 3. Directory allow/deny policy
The safest default is deny-by-default:

```yaml
default: deny
```

Behavior:

- If the folder matches exactly one configured project root, start the agent with that profile.
- If no rule matches, refuse to start.
- If multiple rules match, choose the longest path prefix.
- If a profile is missing for the selected agent, refuse to start.

### 4. Profile isolation strategy
Each subscription profile can have isolated state:

```text
~/.sub-switch/profiles/company-a/claude/
~/.sub-switch/profiles/client-b/claude/
~/.sub-switch/profiles/company-a/codex/
```

The wrapper can set environment variables such as `HOME`, `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, or agent-specific variables before starting the real CLI. This avoids account/session files leaking between projects.

### 5. Shell integration
A shell hook can expose the active profile in the prompt:

```text
~/work/company-a/project [claude: company-a] $
```

The hook should be informational only; enforcement should stay in the launcher so non-interactive starts are also protected.

### 6. Local project marker files
Instead of one central config, projects can contain a marker file:

```text
.agent-subscription.yaml
```

Example:

```yaml
profiles:
  claude: company-a
  codex: company-a
policy: require-match
```

The tool searches upward from the current directory until it finds the marker. A central config can still define which profile names are valid and where their credentials/config directories live.

### 7. Audit log
The launcher can write a small local audit log:

```text
2026-07-08T18:39:39Z cwd=/work/company-a/project agent=claude profile=company-a result=started
2026-07-08T18:40:12Z cwd=/tmp/test agent=codex profile=none result=denied
```

This documents which subscription was selected or when startup was blocked.

## Suggested Minimal Architecture

```text
sub-switch
├── config loader
├── cwd matcher
├── policy engine
├── per-agent adapter
│   ├── claude
│   ├── codex
│   ├── pi
│   └── opencode
└── process launcher
```

The per-agent adapter is the only place that needs to know how a specific agent selects accounts, config directories, or environment variables.

## Open Questions

- Which exact CLIs and installed binary paths should be wrapped on this machine?
- For each agent, which account/session files or environment variables determine the active subscription?
- Should project rules live centrally, in each project, or both?
- Should unknown directories be denied or allowed with an explicit personal/default profile?

## Follow-up Research 2026-07-08T18:41:53Z

### Sandbox wrapper integration
The existing sketch can include a sandbox-aware path where commands are launched as:

```sh
sandbox run pi
sandbox run claude
sandbox run codex
sandbox run opencode
```

In this setup, subscription selection should happen before the Docker container starts, because the host shell still knows the original project directory and can choose which profile/config to mount into the container.

A sandbox-aware flow looks like this:

```text
host cwd
  |
  v
sub-switch / sandbox wrapper resolves allowed profile
  |
  v
prepare profile-specific host config directory
  |
  v
start docker container with:
  - current project mounted
  - only selected profile config mounted
  - selected env vars passed
  |
  v
run requested agent inside container
```

### Two placement options

#### Option A: Wrap outside sandbox
Use `sub-switch` as the outer command:

```sh
sub-switch sandbox run pi
```

or:

```sh
sub-switch pi --sandbox
```

Here, `sub-switch` resolves the current folder on the host, then invokes `sandbox run ...` with extra mounts and environment variables for the selected subscription.

#### Option B: Teach `sandbox run` about profiles
Extend the sandbox command so it owns subscription switching:

```sh
sandbox run pi
```

The sandbox command would:

1. inspect the host current directory,
2. resolve the matching project/subscription profile,
3. refuse to run if no profile is allowed,
4. mount only the selected profile into the container,
5. start the requested agent inside the container.

This keeps the user-facing command unchanged.

### Container mount model
Instead of mounting all agent credentials, mount only the selected profile:

```text
Host:
~/.sub-switch/profiles/company-a/pi/        -> Container: /home/agent/.config/pi
~/.sub-switch/profiles/company-a/claude/    -> Container: /home/agent/.config/claude
~/.sub-switch/profiles/client-b/pi/         -> Container: /home/agent/.config/pi
```

Example generated Docker arguments:

```sh
-v "$PWD:/workspace"
-v "$HOME/.sub-switch/profiles/company-a/pi:/home/agent/.config/pi:rw"
-e SUB_SWITCH_PROFILE=company-a
-e SUB_SWITCH_AGENT=pi
-w /workspace
```

The important property is that the container never receives credentials/config for non-selected subscriptions.

### Sandbox adapter in architecture

```text
sub-switch / sandbox
├── host cwd matcher
├── profile policy engine
├── sandbox adapter
│   ├── project mount
│   ├── profile config mount
│   ├── env var injection
│   └── working directory mapping
└── agent adapter
    ├── pi
    ├── claude
    ├── codex
    └── opencode
```

### Path matching detail
The profile should be selected from the host path before container path translation. For example:

```text
Host path:      /Users/florian/work/company-a/project
Container path: /workspace
Matched rule:   /Users/florian/work/company-a -> company-a
```

The config should therefore store host project roots, not container-only paths.

### Policy behavior with sandbox
The same deny-by-default policy applies:

- no matching host folder: do not start Docker,
- matching folder but no profile for requested agent: do not start Docker,
- matching profile: start Docker with only that profile mounted,
- optional audit log records both the host path and container command.

## Follow-up Research 2026-07-08T18:44:07Z

### Bootstrapping process on a new PC
For an open-source version, the repository should be able to install the generic switching/sandbox logic without containing any private subscription credentials. A new machine bootstrap separates public setup from private profile enrollment.

#### 1. Install the public tool
A user installs the open-source package with the package manager chosen by the project, for example:

```sh
brew install sub-switch
# or
cargo install sub-switch
# or
npm install -g sub-switch
```

The installed commands could include:

```text
sub-switch
sandbox
sub-switch doctor
sub-switch init
sub-switch profile add
sub-switch agent install-wrappers
```

#### 2. Initialize local config
The user creates local machine config:

```sh
sub-switch init
```

This creates a private config directory, for example:

```text
~/.config/sub-switch/config.yaml
~/.local/state/sub-switch/audit.log
~/.sub-switch/profiles/
```

The public repo contains example config templates, but the generated local config contains machine-specific absolute paths and profile names.

#### 3. Register project folders
The user maps host folders to logical subscription profiles:

```sh
sub-switch project add ~/work/company-a --profile company-a
sub-switch project add ~/work/client-b --profile client-b
```

Resulting config shape:

```yaml
default: deny
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

#### 4. Enroll credentials per profile and agent
The user signs in once per agent/profile combination. The tool creates an isolated home/config area and runs the real agent inside it:

```sh
sub-switch profile login company-a pi
sub-switch profile login company-a claude
sub-switch profile login client-b pi
```

Internal profile storage could look like:

```text
~/.sub-switch/profiles/company-a/pi/
~/.sub-switch/profiles/company-a/claude/
~/.sub-switch/profiles/client-b/pi/
```

Each directory contains only the state needed for that profile/agent pair.

#### 5. Configure sandbox support
If agents are normally started through Docker, the bootstrap validates that Docker and the sandbox image are available:

```sh
sub-switch sandbox setup
sub-switch doctor
```

The sandbox integration records how host profile directories map into the container:

```yaml
sandbox:
  workspace_mount: /workspace
  container_user_home: /home/agent
  agents:
    pi:
      config_mount: /home/agent/.config/pi
    claude:
      config_mount: /home/agent/.config/claude
```

#### 6. Install command wrappers
The user can optionally install wrapper commands earlier in `$PATH`:

```sh
sub-switch agent install-wrappers --dir ~/.local/bin
```

This creates commands such as:

```text
~/.local/bin/pi       -> sub-switch run pi
~/.local/bin/claude   -> sub-switch run claude
~/.local/bin/codex    -> sub-switch run codex
~/.local/bin/opencode -> sub-switch run opencode
```

For sandbox-first usage, wrappers can target sandbox mode:

```text
~/.local/bin/pi -> sub-switch sandbox run pi
```

#### 7. Validate behavior
The bootstrap ends with explicit checks:

```sh
cd ~/work/company-a/project
sub-switch which pi
sub-switch sandbox run pi --version

cd ~/Downloads
sub-switch sandbox run pi
# expected: denied, no matching project rule
```

`sub-switch doctor` can report:

- matching project rule for current folder,
- selected profile for each agent,
- whether wrapper commands shadow the real binaries correctly,
- whether Docker is available,
- whether the sandbox image is present,
- whether selected profile directories exist,
- whether default policy is deny or allow.

### Open-source/private boundary
The open-source repository should contain:

```text
CLI implementation
agent adapters
sandbox adapter
config schema
example configs
doctor checks
wrapper installer
documentation
```

A user's private machine contains:

```text
project path rules
profile names
agent credentials/session files
audit logs
machine-specific sandbox paths
```

This boundary allows the tool to be public while keeping subscription data local and uncommitted.
