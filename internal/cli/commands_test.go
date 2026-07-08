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
	if !strings.Contains(content, "agents: {}") || !strings.Contains(content, "projects: []") {
		t.Fatalf("starter config should not include initial agents or projects:\n%s", content)
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
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: example\n"
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

func TestInstallWrappersAddsAgentAndInstallsConfiguredAgents(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	wrapperDir := filepath.Join(dir, "wrappers")
	gemini := filepath.Join(dir, "gemini")
	if err := os.WriteFile(gemini, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	configText := "default: deny\nagents:\n  pi:\n    command: /bin/echo\nprojects: []\n"
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
	if !strings.Contains(string(b), "gemini:") || !strings.Contains(string(b), "command: "+gemini) {
		t.Fatalf("config was not updated with gemini command:\n%s", b)
	}
}

func TestRunCommandQuietForwardsArgsAndEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-agent")
	outFile := filepath.Join(dir, "out")
	script := "#!/bin/sh\necho args:$@ > \"" + outFile + "\"\necho config:$XDG_CONFIG_HOME >> \"" + outFile + "\"\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(dir, "project")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.yaml")
	configText := "default: deny\nui:\n  startup_banner: true\nagents:\n  pi:\n    command: " + fake + "\nprojects:\n  - path: " + proj + "\n    profiles:\n      pi: company\n"
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
	if !strings.Contains(string(b), "args:--version") || !strings.Contains(string(b), "profiles/company/pi/config") {
		t.Fatalf("bad fake output: %s", b)
	}
}
