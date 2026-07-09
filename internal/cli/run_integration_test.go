package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunNonTTY_NoProject_FailFast(t *testing.T) {
	origTTY := isTTY
	isTTY = func() bool { return false }
	defer func() { isTTY = origTTY }()

	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  company:\n    pi:\n      env: {}\nprojects: []\n"
	os.WriteFile(cfg, []byte(configText), 0o600)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(dir)

	_, err := execute("--config", cfg, "run", "pi", "--", "--version")
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !strings.Contains(err.Error(), "denied") || !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("expected denial with terminal hint, got: %v", err)
	}
}

func TestRunNonTTY_ProjectMissingAgent_FailFast(t *testing.T) {
	origTTY := isTTY
	isTTY = func() bool { return false }
	defer func() { isTTY = origTTY }()

	dir := t.TempDir()
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nagents:\n  claude:\n    command: /bin/echo\nprofiles:\n  company:\n    claude:\n      env: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      claude: company\n"
	os.WriteFile(cfg, []byte(configText), 0o600)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(proj)

	_, err := execute("--config", cfg, "run", "pi", "--", "--version")
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denial, got: %v", err)
	}
}

func TestRunNonTTY_MissingTopLevelProfile_FailFast(t *testing.T) {
	origTTY := isTTY
	isTTY = func() bool { return false }
	defer func() { isTTY = origTTY }()

	dir := t.TempDir()
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: ghost\n"
	os.WriteFile(cfg, []byte(configText), 0o600)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(proj)

	_, err := execute("--config", cfg, "run", "pi", "--", "--version")
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denial, got: %v", err)
	}
}

func TestRunNonTTY_MissingProfileAgent_FailFast(t *testing.T) {
	origTTY := isTTY
	isTTY = func() bool { return false }
	defer func() { isTTY = origTTY }()

	dir := t.TempDir()
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  company: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n"
	os.WriteFile(cfg, []byte(configText), 0o600)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(proj)

	_, err := execute("--config", cfg, "run", "pi", "--", "--version")
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denial, got: %v", err)
	}
}

func TestRunNonTTY_MissingAgentCommand_FailFast(t *testing.T) {
	origTTY := isTTY
	isTTY = func() bool { return false }
	defer func() { isTTY = origTTY }()

	dir := t.TempDir()
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nagents: {}\nprofiles:\n  company:\n    pi:\n      env: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n"
	os.WriteFile(cfg, []byte(configText), 0o600)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(proj)

	_, err := execute("--config", cfg, "run", "pi", "--", "--version")
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denial, got: %v", err)
	}
}

func TestRunAlreadyConfigured_LaunchesWithoutSetup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-agent")
	outFile := filepath.Join(dir, "out")
	script := "#!/bin/sh\necho args:$@ > \"" + outFile + "\"\n"
	os.WriteFile(fake, []byte(script), 0o755)

	proj := filepath.Join(dir, "project")
	os.MkdirAll(proj, 0o755)
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nui:\n  startup_banner: true\nagents:\n  pi:\n    command: " + fake + "\nprofiles:\n  company:\n    pi:\n      env:\n        ANTHROPIC_API_KEY: key\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n"
	os.WriteFile(cfg, []byte(configText), 0o600)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(proj)

	out, err := execute("--config", cfg, "run", "pi", "--", "--version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[sub-switch] pi -> profile company") {
		t.Fatalf("expected banner, got: %s", out)
	}

	b, _ := os.ReadFile(outFile)
	if !strings.Contains(string(b), "args:--version") {
		t.Fatalf("args not forwarded: %s", b)
	}
}

func TestRunQuietStillSuppressesBanner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-agent")
	outFile := filepath.Join(dir, "out")
	script := "#!/bin/sh\necho args:$@ > \"" + outFile + "\"\n"
	os.WriteFile(fake, []byte(script), 0o755)

	proj := filepath.Join(dir, "project")
	os.MkdirAll(proj, 0o755)
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nui:\n  startup_banner: true\nagents:\n  pi:\n    command: " + fake + "\nprofiles:\n  company:\n    pi:\n      env: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n"
	os.WriteFile(cfg, []byte(configText), 0o600)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(proj)

	out, err := execute("--config", cfg, "run", "pi", "--quiet", "--", "--version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[sub-switch]") {
		t.Fatalf("quiet should suppress banner: %s", out)
	}
}

func TestWhich_StricterModel(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "work")
	os.MkdirAll(proj, 0o755)

	tests := []struct {
		name       string
		config     string
		wantDenied bool
	}{
		{
			name: "ready",
			config: "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  company:\n    pi:\n      env: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n",
		},
		{
			name:       "missing top-level profile",
			config:     "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: ghost\n",
			wantDenied: true,
		},
		{
			name:       "missing profile-agent entry",
			config:     "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  company: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n",
			wantDenied: true,
		},
		{
			name:       "no matching project",
			config:     "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  company:\n    pi:\n      env: {}\nprojects: []\n",
			wantDenied: true,
		},
		{
			name:       "missing agent command",
			config:     "default: deny\nagents: {}\nprofiles:\n  company:\n    pi:\n      env: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n",
			wantDenied: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := filepath.Join(dir, tt.name+"-config.yaml")
			os.WriteFile(cfg, []byte(tt.config), 0o600)

			cwd, _ := os.Getwd()
			defer os.Chdir(cwd)
			os.Chdir(proj)

			out, err := execute("--config", cfg, "which", "pi")
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantDenied {
				if !strings.Contains(out, "denied") {
					t.Fatalf("expected denied, got: %s", out)
				}
			} else {
				if !strings.Contains(out, "profile company") {
					t.Fatalf("expected selection, got: %s", out)
				}
			}
		})
	}
}
