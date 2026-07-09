package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/florian-balling/sub-switch/internal/resolver"
	"github.com/mattn/go-isatty"
)

// ErrSetupAborted is returned when the user aborts interactive setup.
var ErrSetupAborted = errors.New("setup aborted: no changes saved")

// runPrompter abstracts interactive prompts for testability.
type runPrompter interface {
	Confirm(title, description string, defaultValue bool) (bool, error)
	Select(title string, options []string) (string, error)
	Input(title string, validate func(string) error) (string, error)
}

// isTTY reports whether the current process has a terminal for interactive prompts.
// It can be overridden in tests.
var isTTY = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
}

// huhRunPrompter implements runPrompter using charmbracelet/huh.
type huhRunPrompter struct{}

func (h huhRunPrompter) Confirm(title, description string, defaultValue bool) (bool, error) {
	var result bool
	confirm := huh.NewConfirm().Title(title).Description(description).Value(&result)
	if defaultValue {
		confirm.Affirmative("Yes").Negative("No")
	}
	if err := huh.NewForm(huh.NewGroup(confirm)).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return false, ErrSetupAborted
		}
		return false, err
	}
	return result, nil
}

func (h huhRunPrompter) Select(title string, options []string) (string, error) {
	var result string
	opts := make([]huh.Option[string], len(options))
	for i, o := range options {
		opts[i] = huh.NewOption(o, o)
	}
	sel := huh.NewSelect[string]().Title(title).Options(opts...).Value(&result)
	if err := huh.NewForm(huh.NewGroup(sel)).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", ErrSetupAborted
		}
		return "", err
	}
	return result, nil
}

func (h huhRunPrompter) Input(title string, validate func(string) error) (string, error) {
	var result string
	input := huh.NewInput().Title(title).Value(&result)
	if validate != nil {
		input.Validate(validate)
	}
	if err := huh.NewForm(huh.NewGroup(input)).Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", ErrSetupAborted
		}
		return "", err
	}
	return result, nil
}

const createNewProfileOption = "→ Create new profile"

// runSetup performs interactive setup to fix incomplete run configuration.
// It mutates a copy of the config and saves only if all prompts are accepted.
// Returns the updated config suitable for launching.
func runSetup(
	prompter runPrompter,
	cfg *config.Config,
	cfgPath string,
	analysis resolver.RunAnalysis,
	printer func(format string, a ...interface{}),
) (*config.Config, error) {
	// Clone config to avoid partial mutations on failure.
	clone := cloneCfg(*cfg)

	// Step 1: Ensure agent command is configured.
	if _, ok := clone.Agents[analysis.Agent]; !ok || clone.Agents[analysis.Agent].Command == "" {
		if err := setupAgentCommand(prompter, &clone, analysis.Agent, printer); err != nil {
			return nil, err
		}
	}

	// Step 2: Handle the specific missing configuration.
	switch analysis.Status {
	case resolver.RunNoProject:
		if err := setupNoProject(prompter, &clone, analysis, printer); err != nil {
			return nil, err
		}
	case resolver.RunProjectMissingAgent:
		if err := setupProjectMissingAgent(prompter, &clone, analysis, printer); err != nil {
			return nil, err
		}
	case resolver.RunMissingProfile:
		if err := setupMissingProfile(prompter, &clone, analysis, printer); err != nil {
			return nil, err
		}
	case resolver.RunMissingProfileAgent:
		if err := setupMissingProfileAgent(prompter, &clone, analysis, printer); err != nil {
			return nil, err
		}
	case resolver.RunMissingAgentCommand:
		// Already handled above, nothing else needed.
	}

	// Validate before saving.
	if err := clone.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Save config.
	if err := config.Save(cfgPath, clone, true); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}
	printer("[sub-switch] saved %s\n", cfgPath)

	return &clone, nil
}

func setupAgentCommand(prompter runPrompter, cfg *config.Config, agent string, printer func(string, ...interface{})) error {
	printer("[sub-switch] no configured command for agent %q\n", agent)

	cmd, err := cfg.ConfigureAgentFromPATH(agent)
	if err != nil {
		return fmt.Errorf("cannot auto-configure %s: %w", agent, err)
	}

	ok, err := prompter.Confirm(
		fmt.Sprintf("Configure %s from PATH?", agent),
		fmt.Sprintf("Found: %s", cmd),
		true,
	)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSetupAborted
	}
	printer("[sub-switch] configured %s %s\n", agent, cmd)
	return nil
}

func setupNoProject(prompter runPrompter, cfg *config.Config, analysis resolver.RunAnalysis, printer func(string, ...interface{})) error {
	printer("[sub-switch] no project rule matches %s\n", analysis.CWD)

	profileName, err := selectOrCreateProfile(prompter, cfg, printer)
	if err != nil {
		return err
	}

	ok, err := prompter.Confirm(
		fmt.Sprintf("Add %s with %s → %s?", analysis.CWD, analysis.Agent, profileName),
		"",
		true,
	)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSetupAborted
	}

	if _, err := cfg.SetProjectMapping(analysis.CWD, analysis.Agent, profileName); err != nil {
		return err
	}
	printer("[sub-switch] added project %s\n", analysis.CWD)

	// Ensure profile-agent entry exists.
	if err := cfg.EnsureProfileAgent(profileName, analysis.Agent); err != nil {
		return err
	}
	printer("[sub-switch] allowed %s for profile %s\n", analysis.Agent, profileName)

	return nil
}

func setupProjectMissingAgent(prompter runPrompter, cfg *config.Config, analysis resolver.RunAnalysis, printer func(string, ...interface{})) error {
	printer("[sub-switch] project %s has no mapping for %s\n", analysis.ProjectPath, analysis.Agent)

	profileName, err := selectOrCreateProfile(prompter, cfg, printer)
	if err != nil {
		return err
	}

	// Check for exact current-folder rule conflict.
	cwd := analysis.CWD
	if err := checkExactRuleConflict(prompter, cfg, cwd, analysis.Agent, profileName); err != nil {
		return err
	}

	ok, err := prompter.Confirm(
		fmt.Sprintf("Add %s with %s → %s?", cwd, analysis.Agent, profileName),
		fmt.Sprintf("This adds a new folder-specific rule (does not modify the parent project %s).", analysis.ProjectPath),
		true,
	)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSetupAborted
	}

	if _, err := cfg.SetProjectMapping(cwd, analysis.Agent, profileName); err != nil {
		return err
	}
	printer("[sub-switch] added project %s\n", cwd)

	if err := cfg.EnsureProfileAgent(profileName, analysis.Agent); err != nil {
		return err
	}
	printer("[sub-switch] allowed %s for profile %s\n", analysis.Agent, profileName)

	return nil
}

func setupMissingProfile(prompter runPrompter, cfg *config.Config, analysis resolver.RunAnalysis, printer func(string, ...interface{})) error {
	printer("[sub-switch] profile %q referenced by project %s is not defined in top-level profiles\n", analysis.Profile, analysis.ProjectPath)

	ok, err := prompter.Confirm(
		fmt.Sprintf("Create profile %q and allow %s?", analysis.Profile, analysis.Agent),
		fmt.Sprintf("Profile %q will be added to top-level profiles with an entry for %s.", analysis.Profile, analysis.Agent),
		true,
	)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSetupAborted
	}

	if err := cfg.EnsureProfileAgent(analysis.Profile, analysis.Agent); err != nil {
		return err
	}
	printer("[sub-switch] created profile %s\n", analysis.Profile)
	printer("[sub-switch] allowed %s for profile %s\n", analysis.Agent, analysis.Profile)

	return nil
}

func setupMissingProfileAgent(prompter runPrompter, cfg *config.Config, analysis resolver.RunAnalysis, printer func(string, ...interface{})) error {
	printer("[sub-switch] profile %q exists but does not have an entry for %s\n", analysis.Profile, analysis.Agent)

	ok, err := prompter.Confirm(
		fmt.Sprintf("Allow %s for profile %q?", analysis.Agent, analysis.Profile),
		"This creates an empty agent entry in the profile (no env variables).",
		true,
	)
	if err != nil {
		return err
	}
	if !ok {
		return ErrSetupAborted
	}

	if err := cfg.EnsureProfileAgent(analysis.Profile, analysis.Agent); err != nil {
		return err
	}
	printer("[sub-switch] allowed %s for profile %s\n", analysis.Agent, analysis.Profile)

	return nil
}

func selectOrCreateProfile(prompter runPrompter, cfg *config.Config, printer func(string, ...interface{})) (string, error) {
	names := cfg.ProfileNames()

	if len(names) == 0 {
		// No existing profiles, must create one.
		name, err := prompter.Input("Enter new profile name", func(s string) error {
			return validateProfileNameInput(s)
		})
		if err != nil {
			return "", err
		}
		if err := cfg.EnsureProfile(name); err != nil {
			return "", err
		}
		printer("[sub-switch] created profile %s\n", name)
		return name, nil
	}

	options := make([]string, 0, len(names)+1)
	options = append(options, names...)
	options = append(options, createNewProfileOption)

	choice, err := prompter.Select("Select a profile", options)
	if err != nil {
		return "", err
	}

	if choice == createNewProfileOption {
		name, err := prompter.Input("Enter new profile name", func(s string) error {
			return validateProfileNameInput(s)
		})
		if err != nil {
			return "", err
		}
		if err := cfg.EnsureProfile(name); err != nil {
			return "", err
		}
		printer("[sub-switch] created profile %s\n", name)
		return name, nil
	}

	return choice, nil
}

func checkExactRuleConflict(prompter runPrompter, cfg *config.Config, cwd, agent, newProfile string) error {
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	absPath = filepath.Clean(absPath)

	for i := range cfg.Projects {
		if filepath.Clean(cfg.Projects[i].Path) == absPath {
			if existing, ok := cfg.Projects[i].Profiles[agent]; ok && existing != newProfile {
				ok, err := prompter.Confirm(
					fmt.Sprintf("Replace existing mapping %s → %s with %s → %s?", agent, existing, agent, newProfile),
					fmt.Sprintf("The folder %s already maps %s to profile %q.", cwd, agent, existing),
					false,
				)
				if err != nil {
					return err
				}
				if !ok {
					return ErrSetupAborted
				}
			}
			break
		}
	}
	return nil
}

func validateProfileNameInput(s string) error {
	if s == "" {
		return fmt.Errorf("profile name must not be empty")
	}
	// Reuse the config package validation logic for safe path segments.
	cfg := config.Config{Profiles: map[string]config.ProfileConfig{}}
	return cfg.EnsureProfile(s)
}

func cloneCfg(c config.Config) config.Config {
	// Deep-clone by copying maps.
	agents := make(map[string]config.AgentConfig, len(c.Agents))
	for k, v := range c.Agents {
		agents[k] = v
	}

	profiles := make(map[string]config.ProfileConfig, len(c.Profiles))
	for pn, pc := range c.Profiles {
		newPC := make(config.ProfileConfig, len(pc))
		for an, apc := range pc {
			envCopy := make(map[string]string, len(apc.Env))
			for k, v := range apc.Env {
				envCopy[k] = v
			}
			newPC[an] = config.AgentProfileConfig{Env: envCopy}
		}
		profiles[pn] = newPC
	}

	projects := make([]config.ProjectRule, len(c.Projects))
	for i, p := range c.Projects {
		profCopy := make(map[string]string, len(p.Profiles))
		for k, v := range p.Profiles {
			profCopy[k] = v
		}
		projects[i] = config.ProjectRule{Path: p.Path, Profiles: profCopy}
	}

	return config.Config{
		Default:  c.Default,
		UI:       c.UI,
		Agents:   agents,
		Profiles: profiles,
		Projects: projects,
	}
}
