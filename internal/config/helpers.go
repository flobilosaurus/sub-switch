package config

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/florian-balling/sub-switch/internal/launcher"
)

// ProfileNames returns the sorted list of top-level profile names.
func (c Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// EnsureProfile creates the top-level profile if it does not already exist.
// Existing profiles and their env values are preserved.
func (c *Config) EnsureProfile(name string) error {
	if err := validateSafePathSegment(name); err != nil {
		return fmt.Errorf("invalid profile name %q: %w", name, err)
	}
	if c.Profiles == nil {
		c.Profiles = map[string]ProfileConfig{}
	}
	if _, ok := c.Profiles[name]; !ok {
		c.Profiles[name] = ProfileConfig{}
	}
	return nil
}

// EnsureProfileAgent creates a profile-agent entry with empty env if it does
// not already exist. If the entry already exists, it is not modified.
func (c *Config) EnsureProfileAgent(profileName, agent string) error {
	if err := c.EnsureProfile(profileName); err != nil {
		return err
	}
	if err := validateCommandName(agent); err != nil {
		return fmt.Errorf("invalid agent name %q: %w", agent, err)
	}
	pc := c.Profiles[profileName]
	if _, ok := pc[agent]; !ok {
		pc[agent] = AgentProfileConfig{Env: map[string]string{}}
	}
	return nil
}

// SetProjectMapping adds or updates an exact project rule for the given path.
// If a project rule with the exact path exists, it sets the agent->profile mapping.
// If no exact rule exists, a new one is created.
// Returns (created bool, error).
func (c *Config) SetProjectMapping(projectPath, agent, profileName string) (bool, error) {
	if err := validateCommandName(agent); err != nil {
		return false, err
	}
	if err := validateSafePathSegment(profileName); err != nil {
		return false, err
	}
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return false, err
	}
	absPath = filepath.Clean(absPath)

	for i := range c.Projects {
		if filepath.Clean(c.Projects[i].Path) == absPath {
			if c.Projects[i].Profiles == nil {
				c.Projects[i].Profiles = map[string]string{}
			}
			c.Projects[i].Profiles[agent] = profileName
			return false, nil
		}
	}
	c.Projects = append(c.Projects, ProjectRule{
		Path:     absPath,
		Profiles: map[string]string{agent: profileName},
	})
	return true, nil
}

// ConfigureAgentFromPATH looks up the agent on PATH, validates it, and sets agents[agent].command.
// It uses launcher.ValidateCommand to refuse managed wrappers.
// Returns the resolved command path.
func (c *Config) ConfigureAgentFromPATH(agent string) (string, error) {
	if err := validateCommandName(agent); err != nil {
		return "", err
	}
	command, err := exec.LookPath(agent)
	if err != nil {
		return "", fmt.Errorf("agent %q could not be found on PATH: %w", agent, err)
	}
	if abs, err := filepath.Abs(command); err == nil {
		command = abs
	}
	if err := launcher.ValidateCommand(command); err != nil {
		return "", fmt.Errorf("cannot use %s for agent %s: %w", command, agent, err)
	}
	if c.Agents == nil {
		c.Agents = map[string]AgentConfig{}
	}
	c.Agents[agent] = AgentConfig{Command: command}
	return command, nil
}
