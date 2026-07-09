package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("agents:\n  pi:\n    command: /bin/echo\nprojects:\n  - path: .\n    profiles:\n      pi: test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Default != "deny" || !c.UI.StartupBanner || c.Profiles == nil {
		t.Fatalf("defaults not applied: %#v", c)
	}
}

func TestLoadProfiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	text := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  company:\n    pi:\n      env:\n        ANTHROPIC_API_KEY: key\n        EMPTY: ''\nprojects:\n  - path: .\n    profiles:\n      pi: company\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	c, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Profiles["company"]["pi"].Env["ANTHROPIC_API_KEY"]; got != "key" {
		t.Fatalf("profile env not loaded: %q", got)
	}
	if _, ok := c.Profiles["company"]["pi"].Env["EMPTY"]; !ok {
		t.Fatal("empty env value not preserved")
	}
}

func TestLoadRejectsInvalidDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("default: allow\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateProfiles(t *testing.T) {
	base := Config{
		Default: DefaultPolicyDeny,
		Agents:  map[string]AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]ProfileConfig{
			"company": ProfileConfig{"pi": AgentProfileConfig{Env: map[string]string{"ANTHROPIC_API_KEY": "key", "EMPTY": ""}}},
			"future":  ProfileConfig{"codex": AgentProfileConfig{Env: map[string]string{"OPENAI_API_KEY": "key"}}},
		},
		Projects: []ProjectRule{{Path: "/tmp/project", Profiles: map[string]string{"pi": "missing-profile-ok"}}},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	tests := []struct {
		name string
		mut  func(*Config)
	}{
		{"top profile slash", func(c *Config) { c.Profiles = map[string]ProfileConfig{"bad/name": ProfileConfig{}} }},
		{"top profile trim", func(c *Config) { c.Profiles = map[string]ProfileConfig{" bad": ProfileConfig{}} }},
		{"project profile unsafe", func(c *Config) { c.Projects[0].Profiles["pi"] = ".." }},
		{"profile agent path", func(c *Config) {
			c.Profiles = map[string]ProfileConfig{"company": ProfileConfig{"bad/name": AgentProfileConfig{}}}
		}},
		{"bad env key", func(c *Config) { c.Profiles["company"]["pi"] = AgentProfileConfig{Env: map[string]string{"1BAD": "x"}} }},
		{"reserved env key", func(c *Config) {
			c.Profiles["company"]["pi"] = AgentProfileConfig{Env: map[string]string{"XDG_CONFIG_HOME": "x"}}
		}},
		{"nul env value", func(c *Config) {
			c.Profiles["company"]["pi"] = AgentProfileConfig{Env: map[string]string{"OK": "x\x00y"}}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cloneConfig(base)
			tt.mut(&c)
			if err := c.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestYAMLRoundTripIncludesProfiles(t *testing.T) {
	c := StarterConfig()
	c.Profiles["company"] = ProfileConfig{"pi": {Env: map[string]string{"ANTHROPIC_API_KEY": "key"}}}
	b, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var got Config
	if err := yaml.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	ApplyDefaults(&got)
	if got.Profiles["company"]["pi"].Env["ANTHROPIC_API_KEY"] != "key" {
		t.Fatalf("round trip lost profiles:\n%s", b)
	}
}

func TestSaveRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := Save(path, StarterConfig(), false); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, StarterConfig(), false); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if err := Save(path, StarterConfig(), true); err != nil {
		t.Fatal(err)
	}
}

func TestSaveChmodsExistingConfigTo0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, StarterConfig(), true); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
}

func cloneConfig(c Config) Config {
	b, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}
	var out Config
	if err := yaml.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

func TestStarterConfigIncludesProfiles(t *testing.T) {
	b, err := yaml.Marshal(StarterConfig())
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "profiles: {}") {
		t.Fatalf("starter missing profiles: {}:\n%s", content)
	}
}
