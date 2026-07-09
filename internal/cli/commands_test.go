package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func execute(args ...string) (string, error) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestInitCreatesEmptyStarterConfigAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	if _, err := execute("--config", cfg, "init"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "agents: {}") || !strings.Contains(content, "profiles: {}") || !strings.Contains(content, "projects: []") {
		t.Fatalf("starter config should not include initial agents or projects and should include empty profiles:\n%s", content)
	}
	if _, err := execute("--config", cfg, "init"); err == nil {
		t.Fatal("expected overwrite refusal")
	}
	if _, err := execute("--config", cfg, "init", "--force"); err != nil {
		t.Fatal(err)
	}
}

func TestWhichCommand(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work", "example")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  example:\n    pi:\n      env: {}\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: example\n"
	if err := os.WriteFile(cfg, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	cwd, getwdErr := os.Getwd()
	if getwdErr != nil {
		t.Fatal(getwdErr)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(proj); err != nil {
		t.Fatal(err)
	}
	out, err := execute("--config", cfg, "which", "pi")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "profile example") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestAddProjectAddsCurrentFolderAndUpdatesExistingMapping(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work", "example")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\n  claude:\n    command: /bin/echo\nprofiles:\n  work:\n    pi:\n      env:\n        ANTHROPIC_API_KEY: keep-me\nprojects: []\n"
	if err := os.WriteFile(cfg, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	cwd, getwdErr := os.Getwd()
	if getwdErr != nil {
		t.Fatal(getwdErr)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(proj); err != nil {
		t.Fatal(err)
	}
	resolvedProj, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	out, err := execute("--config", cfg, "add-project", "pi=work", "claude=work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "added project\t"+resolvedProj) {
		t.Fatalf("unexpected output: %s", out)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "path: "+resolvedProj) || !strings.Contains(content, "pi: work") || !strings.Contains(content, "claude: work") || !strings.Contains(content, "ANTHROPIC_API_KEY: keep-me") {
		t.Fatalf("config was not updated with project mappings or preserved profiles:\n%s", content)
	}
	if _, err := execute("--config", cfg, "add-project", "pi=personal"); err == nil {
		t.Fatal("expected conflicting mapping to be refused")
	}
	if _, err := execute("--config", cfg, "add-project", "pi=personal", "--force"); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "pi: personal") {
		t.Fatalf("forced update did not replace profile:\n%s", b)
	}
}

func TestAddProjectConfiguresNewAgentFromPath(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	proj := filepath.Join(dir, "work", "example")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "claude")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprojects: []\n"
	if err := os.WriteFile(cfg, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	cwd, getwdErr := os.Getwd()
	if getwdErr != nil {
		t.Fatal(getwdErr)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(proj); err != nil {
		t.Fatal(err)
	}
	out, err := execute("--config", cfg, "add-project", "claude=work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "configured\tclaude\t"+fake) {
		t.Fatalf("expected configured agent output, got %s", out)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "claude:") || !strings.Contains(content, "command: "+fake) || !strings.Contains(content, "claude: work") {
		t.Fatalf("config was not updated with new agent and project mapping:\n%s", content)
	}
}

func TestAddProjectRefusesNewAgentMissingFromPath(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	t.Setenv("PATH", dir)
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprojects: []\n"
	if err := os.WriteFile(cfg, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := execute("--config", cfg, "add-project", "claude=work")
	if err == nil || !strings.Contains(err.Error(), `agent "claude" is not configured and could not be found on PATH`) {
		t.Fatalf("expected missing PATH agent error, got %v", err)
	}
}

func TestInstallWrappersAddsAgentAndInstallsConfiguredAgents(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	wrapperDir := filepath.Join(dir, "wrappers")
	gemini := filepath.Join(dir, "gemini")
	if err := os.WriteFile(gemini, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprofiles:\n  work:\n    pi:\n      env:\n        ANTHROPIC_API_KEY: keep-me\nprojects: []\n"
	if err := os.WriteFile(cfg, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := execute("--config", cfg, "install-wrappers", "gemini", "--dir", wrapperDir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "configured\tgemini") {
		t.Fatalf("expected gemini configuration in output: %s", out)
	}
	if _, err := os.Stat(filepath.Join(wrapperDir, "gemini")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(wrapperDir, "pi")); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "gemini:") || !strings.Contains(string(b), "command: "+gemini) || !strings.Contains(string(b), "ANTHROPIC_API_KEY: keep-me") {
		t.Fatalf("config was not updated with gemini command or preserved profiles:\n%s", b)
	}
}

func TestRunCommandQuietForwardsArgsAndEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-agent")
	outFile := filepath.Join(dir, "out")
	script := "#!/bin/sh\n{\n" +
		"echo args:$@\n" +
		"echo config:$XDG_CONFIG_HOME\n" +
		"echo pi_dir:$PI_CODING_AGENT_DIR\n" +
		"echo anthropic:${ANTHROPIC_API_KEY-unset}\n" +
		"echo openai:${OPENAI_API_KEY-unset}\n" +
		"echo keep:$KEEP_ME\n" +
		"} > \"" + outFile + "\"\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(dir, "project")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nui:\n  startup_banner: true\nagents:\n  pi:\n    command: " + fake + "\nprofiles:\n  company:\n    pi:\n      env:\n        ANTHROPIC_API_KEY: company-key\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n"
	if err := os.WriteFile(cfg, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "global")
	t.Setenv("OPENAI_API_KEY", "global")
	t.Setenv("KEEP_ME", "yes")
	cwd, getwdErr := os.Getwd()
	if getwdErr != nil {
		t.Fatal(getwdErr)
	}
	defer os.Chdir(cwd)
	if err := os.Chdir(proj); err != nil {
		t.Fatal(err)
	}
	out, err := execute("--config", cfg, "run", "pi", "--quiet", "--", "--version")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "[sub-switch]") {
		t.Fatalf("quiet printed banner: %s", out)
	}
	b, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(b)
	if !strings.Contains(content, "args:--version") || !strings.Contains(content, "profiles/company/pi/config") || !strings.Contains(content, "pi_dir:") || !strings.Contains(content, "anthropic:company-key") || !strings.Contains(content, "openai:unset") || !strings.Contains(content, "keep:yes") {
		t.Fatalf("bad fake output: %s", b)
	}
}
