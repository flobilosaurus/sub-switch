package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultPolicyDeny = "deny"

type Config struct {
	Agents   map[string]AgentConfig `yaml:"agents"`
	Default  string                 `yaml:"default"`
	Projects []ProjectRule          `yaml:"projects"`
	UI       UIConfig               `yaml:"ui"`
}

type UIConfig struct {
	StartupBanner bool `yaml:"startup_banner"`
}
type AgentConfig struct {
	Command string `yaml:"command"`
}
type ProjectRule struct {
	Profiles map[string]string `yaml:"profiles"`
	Path     string            `yaml:"path"`
}

func ApplyDefaults(c *Config) {
	if c.Default == "" {
		c.Default = DefaultPolicyDeny
	}
	// Startup banner defaults true; detect zero-value impossible after unmarshal, so callers use Load/Starter.
	if c.Agents == nil {
		c.Agents = map[string]AgentConfig{}
	}
}

func DefaultPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "sub-switch", "config.yaml"), nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".config", "sub-switch", "config.yaml"), nil
}

func ResolvePath(p string) (string, error) {
	if p != "" {
		return ExpandPath(p)
	}
	return DefaultPath()
}

func ExpandPath(p string) (string, error) {
	if strings.HasPrefix(p, "~") {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			p = h
		} else if strings.HasPrefix(p, "~/") {
			p = filepath.Join(h, p[2:])
		}
	}
	return filepath.Abs(p)
}

func Load(path string) (*Config, string, error) {
	resolved, err := ResolvePath(path)
	if err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(resolved)
	if err != nil {
		return nil, resolved, err
	}
	var raw struct {
		Default  string                 `yaml:"default"`
		UI       *UIConfig              `yaml:"ui"`
		Agents   map[string]AgentConfig `yaml:"agents"`
		Projects []ProjectRule          `yaml:"projects"`
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, resolved, err
	}
	c := Config{Default: raw.Default, UI: UIConfig{StartupBanner: true}, Agents: raw.Agents, Projects: raw.Projects}
	if raw.UI != nil {
		c.UI = *raw.UI
	}
	ApplyDefaults(&c)
	for name, ac := range c.Agents {
		if ac.Command != "" {
			e, err := ExpandPath(ac.Command)
			if err != nil {
				return nil, resolved, err
			}
			ac.Command = e
			c.Agents[name] = ac
		}
	}
	for i := range c.Projects {
		e, err := ExpandPath(c.Projects[i].Path)
		if err != nil {
			return nil, resolved, err
		}
		c.Projects[i].Path = filepath.Clean(e)
	}
	return &c, resolved, c.Validate()
}

func Save(path string, c Config, force bool) error {
	resolved, err := ResolvePath(path)
	if err != nil {
		return err
	}
	if !force {
		_, statErr := os.Stat(resolved)
		if statErr == nil {
			return fmt.Errorf("config already exists: %s (use --force to overwrite)", resolved)
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
	}
	mkdirErr := os.MkdirAll(filepath.Dir(resolved), 0o755)
	if mkdirErr != nil {
		return mkdirErr
	}
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(resolved, b, 0o600)
}

func (c Config) Validate() error {
	if c.Default == "" || c.Default == DefaultPolicyDeny {
	} else {
		return fmt.Errorf("unsupported default policy %q", c.Default)
	}
	for name, ac := range c.Agents {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("agent name must not be empty")
		}
		if name != filepath.Base(name) || name == "." || name == ".." {
			return fmt.Errorf("agent %s name must be a command name, not a path", name)
		}
		if strings.TrimSpace(ac.Command) == "" {
			return fmt.Errorf("agent %s command must not be empty", name)
		}
	}
	for i, p := range c.Projects {
		if strings.TrimSpace(p.Path) == "" {
			return fmt.Errorf("project %d path must not be empty", i)
		}
		if len(p.Profiles) == 0 {
			return fmt.Errorf("project %s must define profiles", p.Path)
		}
		for a, prof := range p.Profiles {
			if strings.TrimSpace(a) == "" || strings.TrimSpace(prof) == "" {
				return fmt.Errorf("project %s has invalid profile mapping", p.Path)
			}
		}
	}
	return nil
}

func AgentNames(agents map[string]AgentConfig) []string {
	names := make([]string, 0, len(agents))
	for name := range agents {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func StarterConfig() Config {
	return Config{Default: DefaultPolicyDeny, UI: UIConfig{StartupBanner: true}, Agents: map[string]AgentConfig{}, Projects: []ProjectRule{}}
}
