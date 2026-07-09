package resolver

import (
	"path/filepath"
	"testing"

	"github.com/florian-balling/sub-switch/internal/config"
)

func TestAnalyzeRunReady(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "work")
	c := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company": {"pi": config.AgentProfileConfig{Env: map[string]string{"KEY": "val"}}},
		},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}
	a, err := AnalyzeRun(c, filepath.Join(proj, "child"), "pi")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Ready() {
		t.Fatalf("expected ready, got status=%d reason=%q", a.Status, a.Reason)
	}
	if a.Profile != "company" || a.ProjectPath != proj {
		t.Fatalf("unexpected result: %+v", a)
	}
}

func TestAnalyzeRunNoProject(t *testing.T) {
	root := t.TempDir()
	c := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company": {"pi": config.AgentProfileConfig{}},
		},
	}
	a, err := AnalyzeRun(c, root, "pi")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != RunNoProject {
		t.Fatalf("expected RunNoProject, got %d: %s", a.Status, a.Reason)
	}
}

func TestAnalyzeRunProjectMissingAgent(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "work")
	c := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{"company": {"pi": config.AgentProfileConfig{}}},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}
	a, err := AnalyzeRun(c, proj, "claude")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != RunProjectMissingAgent {
		t.Fatalf("expected RunProjectMissingAgent, got %d: %s", a.Status, a.Reason)
	}
}

func TestAnalyzeRunMissingProfile(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "work")
	c := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}
	a, err := AnalyzeRun(c, proj, "pi")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != RunMissingProfile {
		t.Fatalf("expected RunMissingProfile, got %d: %s", a.Status, a.Reason)
	}
	if a.Profile != "company" {
		t.Fatalf("expected profile=company, got %q", a.Profile)
	}
}

func TestAnalyzeRunMissingProfileAgent(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "work")
	c := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{"company": {}},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}
	a, err := AnalyzeRun(c, proj, "pi")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != RunMissingProfileAgent {
		t.Fatalf("expected RunMissingProfileAgent, got %d: %s", a.Status, a.Reason)
	}
}

func TestAnalyzeRunMissingAgentCommand(t *testing.T) {
	root := t.TempDir()
	proj := filepath.Join(root, "work")
	c := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{},
		Profiles: map[string]config.ProfileConfig{"company": {"pi": config.AgentProfileConfig{}}},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}
	a, err := AnalyzeRun(c, proj, "pi")
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != RunMissingAgentCommand {
		t.Fatalf("expected RunMissingAgentCommand, got %d: %s", a.Status, a.Reason)
	}
}

func TestAnalyzeRunLongestPrefixUnchanged(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	ab := filepath.Join(a, "b")
	c := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"profile-a":  {"pi": config.AgentProfileConfig{}},
			"profile-ab": {"pi": config.AgentProfileConfig{}},
		},
		Projects: []config.ProjectRule{
			{Path: a, Profiles: map[string]string{"pi": "profile-a"}},
			{Path: ab, Profiles: map[string]string{"pi": "profile-ab"}},
		},
	}
	result, err := AnalyzeRun(c, filepath.Join(ab, "child"), "pi")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Ready() || result.Profile != "profile-ab" {
		t.Fatalf("expected longest prefix profile-ab, got %+v", result)
	}
}
