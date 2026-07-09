package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProfileNames(t *testing.T) {
	c := Config{Profiles: map[string]ProfileConfig{
		"beta":  {},
		"alpha": {},
		"gamma": {},
	}}
	names := c.ProfileNames()
	if len(names) != 3 || names[0] != "alpha" || names[1] != "beta" || names[2] != "gamma" {
		t.Fatalf("expected sorted [alpha beta gamma], got %v", names)
	}
}

func TestProfileNamesEmpty(t *testing.T) {
	c := Config{}
	names := c.ProfileNames()
	if len(names) != 0 {
		t.Fatalf("expected empty, got %v", names)
	}
}

func TestEnsureProfileCreatesNew(t *testing.T) {
	c := Config{Profiles: map[string]ProfileConfig{
		"existing": {"pi": AgentProfileConfig{Env: map[string]string{"KEY": "val"}}},
	}}
	if err := c.EnsureProfile("newone"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Profiles["newone"]; !ok {
		t.Fatal("new profile not created")
	}
	// Existing profile preserved.
	if c.Profiles["existing"]["pi"].Env["KEY"] != "val" {
		t.Fatal("existing profile env lost")
	}
}

func TestEnsureProfilePreservesExisting(t *testing.T) {
	c := Config{Profiles: map[string]ProfileConfig{
		"existing": {"pi": AgentProfileConfig{Env: map[string]string{"KEY": "val"}}},
	}}
	if err := c.EnsureProfile("existing"); err != nil {
		t.Fatal(err)
	}
	if c.Profiles["existing"]["pi"].Env["KEY"] != "val" {
		t.Fatal("existing profile env lost after EnsureProfile")
	}
}

func TestEnsureProfileRejectsInvalidName(t *testing.T) {
	c := Config{Profiles: map[string]ProfileConfig{}}
	if err := c.EnsureProfile("bad/name"); err == nil {
		t.Fatal("expected error for invalid profile name")
	}
}

func TestEnsureProfileAgentCreatesNew(t *testing.T) {
	c := Config{Profiles: map[string]ProfileConfig{
		"company": {"pi": AgentProfileConfig{Env: map[string]string{"KEY": "val"}}},
	}}
	if err := c.EnsureProfileAgent("company", "claude"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Profiles["company"]["claude"]; !ok {
		t.Fatal("new agent entry not created")
	}
	// Existing agent entry preserved.
	if c.Profiles["company"]["pi"].Env["KEY"] != "val" {
		t.Fatal("existing agent env lost")
	}
}

func TestEnsureProfileAgentPreservesExisting(t *testing.T) {
	c := Config{Profiles: map[string]ProfileConfig{
		"company": {"pi": AgentProfileConfig{Env: map[string]string{"KEY": "val"}}},
	}}
	if err := c.EnsureProfileAgent("company", "pi"); err != nil {
		t.Fatal(err)
	}
	if c.Profiles["company"]["pi"].Env["KEY"] != "val" {
		t.Fatal("existing env lost after EnsureProfileAgent")
	}
}

func TestEnsureProfileAgentCreatesProfileIfMissing(t *testing.T) {
	c := Config{Profiles: map[string]ProfileConfig{}}
	if err := c.EnsureProfileAgent("newprof", "pi"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Profiles["newprof"]["pi"]; !ok {
		t.Fatal("profile and agent entry not created")
	}
}

func TestSetProjectMappingAddsNewRule(t *testing.T) {
	dir := t.TempDir()
	c := Config{Projects: []ProjectRule{}}
	created, err := c.SetProjectMapping(dir, "pi", "company")
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected created=true for new rule")
	}
	if len(c.Projects) != 1 || c.Projects[0].Profiles["pi"] != "company" {
		t.Fatalf("unexpected projects: %+v", c.Projects)
	}
}

func TestSetProjectMappingUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	absDir, _ := filepath.Abs(dir)
	absDir = filepath.Clean(absDir)
	c := Config{Projects: []ProjectRule{
		{Path: absDir, Profiles: map[string]string{"claude": "work"}},
	}}
	created, err := c.SetProjectMapping(dir, "pi", "company")
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("expected created=false for existing rule")
	}
	if len(c.Projects) != 1 {
		t.Fatalf("expected 1 project rule, got %d", len(c.Projects))
	}
	if c.Projects[0].Profiles["pi"] != "company" || c.Projects[0].Profiles["claude"] != "work" {
		t.Fatalf("unexpected profiles: %+v", c.Projects[0].Profiles)
	}
}

func TestConfigureAgentFromPATH(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "myagent")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	c := Config{Agents: map[string]AgentConfig{}}
	cmd, err := c.ConfigureAgentFromPATH("myagent")
	if err != nil {
		t.Fatal(err)
	}
	if cmd == "" {
		t.Fatal("expected non-empty command path")
	}
	if c.Agents["myagent"].Command != cmd {
		t.Fatalf("agent command not set: %+v", c.Agents)
	}
}

func TestConfigureAgentFromPATHMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	c := Config{Agents: map[string]AgentConfig{}}
	_, err := c.ConfigureAgentFromPATH("nonexistent-agent")
	if err == nil {
		t.Fatal("expected error for missing agent on PATH")
	}
}
