package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/florian-balling/sub-switch/internal/resolver"
)

// fakePrompter implements runPrompter for tests.
type fakePrompter struct {
	confirms []bool
	selects  []string
	inputs   []string
	confirmI int
	selectI  int
	inputI   int
	abortAt  int // -1 means never abort, otherwise abort at this total call count
	callN    int
}

func newFakePrompter(confirms []bool, selects []string, inputs []string) *fakePrompter {
	return &fakePrompter{confirms: confirms, selects: selects, inputs: inputs, abortAt: -1}
}

func newAbortingPrompter(abortAfter int) *fakePrompter {
	return &fakePrompter{abortAt: abortAfter}
}

func (f *fakePrompter) Confirm(title, description string, defaultValue bool) (bool, error) {
	f.callN++
	if f.abortAt >= 0 && f.callN > f.abortAt {
		return false, ErrSetupAborted
	}
	if f.confirmI >= len(f.confirms) {
		return false, fmt.Errorf("unexpected Confirm call #%d: %s", f.confirmI, title)
	}
	v := f.confirms[f.confirmI]
	f.confirmI++
	return v, nil
}

func (f *fakePrompter) Select(title string, options []string) (string, error) {
	f.callN++
	if f.abortAt >= 0 && f.callN > f.abortAt {
		return "", ErrSetupAborted
	}
	if f.selectI >= len(f.selects) {
		return "", fmt.Errorf("unexpected Select call #%d: %s", f.selectI, title)
	}
	v := f.selects[f.selectI]
	f.selectI++
	return v, nil
}

func (f *fakePrompter) Input(title string, validate func(string) error) (string, error) {
	f.callN++
	if f.abortAt >= 0 && f.callN > f.abortAt {
		return "", ErrSetupAborted
	}
	if f.inputI >= len(f.inputs) {
		return "", fmt.Errorf("unexpected Input call #%d: %s", f.inputI, title)
	}
	v := f.inputs[f.inputI]
	f.inputI++
	if validate != nil {
		if err := validate(v); err != nil {
			return "", err
		}
	}
	return v, nil
}

func testPrinter() func(string, ...interface{}) {
	return func(format string, a ...interface{}) {
		// discard output in tests
	}
}

func capturedPrinter() (func(string, ...interface{}), *[]string) {
	var lines []string
	return func(format string, a ...interface{}) {
		lines = append(lines, fmt.Sprintf(format, a...))
	}, &lines
}

func TestSetupNoProject_ExistingProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "myproject")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company": {"pi": config.AgentProfileConfig{Env: map[string]string{"KEY": "keep"}}},
			"personal": {},
		},
		Projects: []config.ProjectRule{},
	}

	analysis := resolver.RunAnalysis{
		Agent:  "pi",
		CWD:    proj,
		Status: resolver.RunNoProject,
		Reason: "no project rule matches " + proj,
	}

	// Select "company", confirm add
	prompter := newFakePrompter([]bool{true}, []string{"company"}, nil)
	result, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}

	// Check saved config
	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// Project rule added
	found := false
	for _, p := range loaded.Projects {
		if filepath.Clean(p.Path) == filepath.Clean(proj) && p.Profiles["pi"] == "company" {
			found = true
		}
	}
	if !found {
		t.Fatalf("project rule not found in saved config: %+v", loaded.Projects)
	}
	// Existing env preserved
	if loaded.Profiles["company"]["pi"].Env["KEY"] != "keep" {
		t.Fatal("existing profile env not preserved")
	}
	// Profile-agent entry exists
	if _, ok := result.Profiles["company"]["pi"]; !ok {
		t.Fatal("profile-agent entry not created")
	}

	// Verify file mode is 0600
	st, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", st.Mode().Perm())
	}
}

func TestSetupNoProject_NewProfile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "myproject")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{"existing": {}},
		Projects: []config.ProjectRule{},
	}

	analysis := resolver.RunAnalysis{
		Agent:  "pi",
		CWD:    proj,
		Status: resolver.RunNoProject,
	}

	// Select "create new", input "newprof", confirm add
	prompter := newFakePrompter([]bool{true}, []string{createNewProfileOption}, []string{"newprof"})
	_, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}

	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Profiles["newprof"]; !ok {
		t.Fatal("new profile not created in top-level profiles")
	}
	if _, ok := loaded.Profiles["newprof"]["pi"]; !ok {
		t.Fatal("profile-agent entry not created for new profile")
	}
}

func TestSetupProjectMissingAgent_AddsCurrentFolderRule(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	parent := filepath.Join(dir, "parent")
	child := filepath.Join(parent, "child")
	os.MkdirAll(child, 0o755)

	cfg := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}, "claude": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company": {"claude": config.AgentProfileConfig{}},
		},
		Projects: []config.ProjectRule{
			{Path: parent, Profiles: map[string]string{"claude": "company"}},
		},
	}

	analysis := resolver.RunAnalysis{
		Agent:        "pi",
		CWD:          child,
		ProjectPath:  parent,
		ProjectIndex: 0,
		Status:       resolver.RunProjectMissingAgent,
	}

	// Select "company", confirm add
	prompter := newFakePrompter([]bool{true}, []string{"company"}, nil)
	result, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}

	// Parent rule untouched
	parentFound := false
	for _, p := range result.Projects {
		if filepath.Clean(p.Path) == filepath.Clean(parent) {
			parentFound = true
			if _, has := p.Profiles["pi"]; has {
				t.Fatal("parent rule should not be modified with pi mapping")
			}
		}
	}
	if !parentFound {
		t.Fatal("parent rule disappeared")
	}

	// Child rule created
	childFound := false
	for _, p := range result.Projects {
		if filepath.Clean(p.Path) == filepath.Clean(child) && p.Profiles["pi"] == "company" {
			childFound = true
		}
	}
	if !childFound {
		t.Fatalf("child rule not created: %+v", result.Projects)
	}
}

func TestSetupMissingProfile_CreatesProfileAndAgent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}

	analysis := resolver.RunAnalysis{
		Agent:        "pi",
		CWD:          proj,
		ProjectPath:  proj,
		Profile:      "company",
		ProjectIndex: 0,
		Status:       resolver.RunMissingProfile,
	}

	prompter := newFakePrompter([]bool{true}, nil, nil)
	result, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Profiles["company"]; !ok {
		t.Fatal("profile not created")
	}
	if _, ok := result.Profiles["company"]["pi"]; !ok {
		t.Fatal("profile-agent entry not created")
	}
}

func TestSetupMissingProfileAgent_CreatesEntry(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company": {"claude": config.AgentProfileConfig{Env: map[string]string{"KEY": "keep"}}},
		},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}

	analysis := resolver.RunAnalysis{
		Agent:        "pi",
		CWD:          proj,
		ProjectPath:  proj,
		Profile:      "company",
		ProjectIndex: 0,
		Status:       resolver.RunMissingProfileAgent,
	}

	prompter := newFakePrompter([]bool{true}, nil, nil)
	result, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := result.Profiles["company"]["pi"]; !ok {
		t.Fatal("profile-agent entry not created")
	}
	// Existing agent preserved
	if result.Profiles["company"]["claude"].Env["KEY"] != "keep" {
		t.Fatal("existing agent env lost")
	}
}

func TestSetupMissingAgentCommand(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	// Create fake agent on PATH
	fake := filepath.Join(dir, "myagent")
	os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{},
		Profiles: map[string]config.ProfileConfig{
			"company": {"myagent": config.AgentProfileConfig{}},
		},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"myagent": "company"}}},
	}

	analysis := resolver.RunAnalysis{
		Agent:        "myagent",
		CWD:          proj,
		ProjectPath:  proj,
		Profile:      "company",
		ProjectIndex: 0,
		Status:       resolver.RunMissingAgentCommand,
	}

	prompter := newFakePrompter([]bool{true}, nil, nil)
	result, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}
	if result.Agents["myagent"].Command == "" {
		t.Fatal("agent command not configured")
	}
}

func TestSetupMissingAgentCommand_NotOnPATH(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	t.Setenv("PATH", dir) // empty dir

	cfg := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{},
		Profiles: map[string]config.ProfileConfig{"company": {"ghost": config.AgentProfileConfig{}}},
		Projects: []config.ProjectRule{{Path: dir, Profiles: map[string]string{"ghost": "company"}}},
	}

	analysis := resolver.RunAnalysis{
		Agent:  "ghost",
		CWD:    dir,
		Status: resolver.RunMissingAgentCommand,
	}

	prompter := newFakePrompter(nil, nil, nil)
	_, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err == nil {
		t.Fatal("expected error for missing PATH agent")
	}
	// Config file should not be created
	if _, statErr := os.Stat(cfgPath); statErr == nil {
		t.Fatal("config file should not have been saved on error")
	}
}

func TestSetupMissingAgentCommand_ManagedWrapper(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	// Create a fake managed wrapper
	fake := filepath.Join(dir, "wrapagent")
	os.WriteFile(fake, []byte("#!/bin/sh\n# managed by sub-switch – do not edit\nexec sub-switch run wrapagent -- \"$@\"\n"), 0o755)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{},
		Profiles: map[string]config.ProfileConfig{"company": {"wrapagent": config.AgentProfileConfig{}}},
		Projects: []config.ProjectRule{{Path: dir, Profiles: map[string]string{"wrapagent": "company"}}},
	}

	analysis := resolver.RunAnalysis{
		Agent:  "wrapagent",
		CWD:    dir,
		Status: resolver.RunMissingAgentCommand,
	}

	prompter := newFakePrompter(nil, nil, nil)
	_, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err == nil {
		t.Fatal("expected error for managed wrapper")
	}
	if _, statErr := os.Stat(cfgPath); statErr == nil {
		t.Fatal("config file should not have been saved")
	}
}

func TestSetupAbort_NoSave(t *testing.T) {
	tests := []struct {
		name     string
		status   resolver.RunStatus
		abortAt  int
	}{
		{"no project abort at select", resolver.RunNoProject, 1},
		{"no project abort at confirm", resolver.RunNoProject, 2},
		{"missing profile abort", resolver.RunMissingProfile, 1},
		{"missing profile-agent abort", resolver.RunMissingProfileAgent, 1},
		{"missing command abort at confirm", resolver.RunMissingAgentCommand, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "config.yaml")

			// Create a fake agent so MissingAgentCommand can find it
			fake := filepath.Join(dir, "pi")
			os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755)
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

			cfg := config.Config{
				Default:  "deny",
				Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
				Profiles: map[string]config.ProfileConfig{"company": {"pi": config.AgentProfileConfig{}}},
				Projects: []config.ProjectRule{{Path: dir, Profiles: map[string]string{"pi": "company"}}},
			}

			// Adjust config for the specific status
			switch tt.status {
			case resolver.RunNoProject:
				cfg.Projects = nil
			case resolver.RunMissingProfile:
				cfg.Profiles = map[string]config.ProfileConfig{}
			case resolver.RunMissingProfileAgent:
				cfg.Profiles = map[string]config.ProfileConfig{"company": {}}
			case resolver.RunMissingAgentCommand:
				cfg.Agents = map[string]config.AgentConfig{}
			}

			analysis := resolver.RunAnalysis{
				Agent:   "pi",
				CWD:     dir,
				Profile: "company",
				Status:  tt.status,
			}

			prompter := newAbortingPrompter(tt.abortAt)
			_, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
			if err == nil {
				t.Fatal("expected setup abort error")
			}
			// Verify no config file saved
			if _, statErr := os.Stat(cfgPath); statErr == nil {
				t.Fatal("config file should not exist after abort")
			}
		})
	}
}

func TestSetupConfirmReject_NoSave(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{"company": {}},
		Projects: []config.ProjectRule{},
	}

	analysis := resolver.RunAnalysis{
		Agent:  "pi",
		CWD:    proj,
		Status: resolver.RunNoProject,
	}

	// Select "company", then reject confirm
	prompter := newFakePrompter([]bool{false}, []string{"company"}, nil)
	_, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err == nil || !strings.Contains(err.Error(), "aborted") {
		t.Fatalf("expected abort error, got %v", err)
	}
	if _, statErr := os.Stat(cfgPath); statErr == nil {
		t.Fatal("config file should not exist after rejection")
	}
}

func TestSetupExactRuleConflict(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)
	absProj, _ := filepath.Abs(proj)
	absProj = filepath.Clean(absProj)

	cfg := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company":  {"pi": config.AgentProfileConfig{}},
			"personal": {"pi": config.AgentProfileConfig{}},
		},
		Projects: []config.ProjectRule{
			{Path: absProj, Profiles: map[string]string{"pi": "company"}},
		},
	}

	analysis := resolver.RunAnalysis{
		Agent:        "pi",
		CWD:          absProj,
		ProjectPath:  absProj,
		ProjectIndex: 0,
		Status:       resolver.RunProjectMissingAgent,
	}

	// Select "personal" (different from "company"), confirm replacement, confirm add
	prompter := newFakePrompter([]bool{true, true}, []string{"personal"}, nil)
	result, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}
	// Should have updated the mapping
	for _, p := range result.Projects {
		if filepath.Clean(p.Path) == absProj {
			if p.Profiles["pi"] != "personal" {
				t.Fatalf("expected pi=personal, got pi=%s", p.Profiles["pi"])
			}
		}
	}
}

func TestSetupSaved0600(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company": {},
		},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}

	analysis := resolver.RunAnalysis{
		Agent:        "pi",
		CWD:          proj,
		ProjectPath:  proj,
		Profile:      "company",
		ProjectIndex: 0,
		Status:       resolver.RunMissingProfileAgent,
	}

	prompter := newFakePrompter([]bool{true}, nil, nil)
	_, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}

	st, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", st.Mode().Perm())
	}
}

func TestSetupPreservesExistingEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default: "deny",
		Agents:  map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{
			"company": {
				"claude": config.AgentProfileConfig{Env: map[string]string{"PRESERVE": "me"}},
			},
		},
		Projects: []config.ProjectRule{{Path: proj, Profiles: map[string]string{"pi": "company"}}},
	}

	analysis := resolver.RunAnalysis{
		Agent:        "pi",
		CWD:          proj,
		ProjectPath:  proj,
		Profile:      "company",
		ProjectIndex: 0,
		Status:       resolver.RunMissingProfileAgent,
	}

	prompter := newFakePrompter([]bool{true}, nil, nil)
	result, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}
	if result.Profiles["company"]["claude"].Env["PRESERVE"] != "me" {
		t.Fatal("existing env values not preserved")
	}

	// Also check persisted file
	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Profiles["company"]["claude"].Env["PRESERVE"] != "me" {
		t.Fatal("existing env values not preserved in saved file")
	}
}

func TestSetupNoProfilesCreatesNew(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	cfg := config.Config{
		Default:  "deny",
		Agents:   map[string]config.AgentConfig{"pi": {Command: "/bin/echo"}},
		Profiles: map[string]config.ProfileConfig{},
		Projects: []config.ProjectRule{},
	}

	analysis := resolver.RunAnalysis{
		Agent:  "pi",
		CWD:    proj,
		Status: resolver.RunNoProject,
	}

	// Input new profile name, confirm add
	prompter := newFakePrompter([]bool{true}, nil, []string{"fresh"})
	_, err := runSetup(prompter, &cfg, cfgPath, analysis, testPrinter())
	if err != nil {
		t.Fatal(err)
	}

	loaded, _, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := loaded.Profiles["fresh"]; !ok {
		t.Fatal("new profile not created")
	}
	if _, ok := loaded.Profiles["fresh"]["pi"]; !ok {
		t.Fatal("profile-agent entry not created for new profile")
	}
}
