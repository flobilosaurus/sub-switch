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
	cmd := NewRootCommand(); var out bytes.Buffer; cmd.SetOut(&out); cmd.SetErr(&out); cmd.SetArgs(args); err := cmd.Execute(); return out.String(), err
}

func TestInitAndWhichCommands(t *testing.T) {
	dir := t.TempDir(); cfg := filepath.Join(dir,"config.yaml")
	if _,err := execute("--config", cfg, "init"); err != nil { t.Fatal(err) }
	if _,err := execute("--config", cfg, "init"); err == nil { t.Fatal("expected overwrite refusal") }
	if _,err := execute("--config", cfg, "init", "--force"); err != nil { t.Fatal(err) }
	cwd, _ := os.Getwd(); defer os.Chdir(cwd)
	proj := filepath.Join(dir,"work","example"); if err := os.MkdirAll(proj,0o755); err != nil { t.Fatal(err) }
	b,err := os.ReadFile(cfg); if err != nil { t.Fatal(err) }
	content := strings.ReplaceAll(string(b), "~/work/example", proj)
	if err := os.WriteFile(cfg, []byte(content),0o600); err != nil { t.Fatal(err) }
	if err := os.Chdir(proj); err != nil { t.Fatal(err) }
	out,err := execute("--config", cfg, "which", "pi"); if err != nil { t.Fatal(err) }
	if !strings.Contains(out,"profile example") { t.Fatalf("unexpected output: %s", out) }
}

func TestRunCommandQuietForwardsArgsAndEnv(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip("shell script") }
	dir := t.TempDir(); fake := filepath.Join(dir,"fake-agent"); outFile := filepath.Join(dir,"out")
	script := "#!/bin/sh\necho args:$@ > \""+outFile+"\"\necho config:$XDG_CONFIG_HOME >> \""+outFile+"\"\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil { t.Fatal(err) }
	proj := filepath.Join(dir,"project"); if err := os.MkdirAll(proj,0o755); err != nil { t.Fatal(err) }
	cfg := filepath.Join(dir,"config.yaml")
	configText := "default: deny\nui:\n  startup_banner: true\nagents:\n  pi:\n    command: "+fake+"\nprojects:\n  - path: "+proj+"\n    profiles:\n      pi: company\n"
	if err := os.WriteFile(cfg, []byte(configText),0o600); err != nil { t.Fatal(err) }
	cwd,_ := os.Getwd(); defer os.Chdir(cwd); if err := os.Chdir(proj); err != nil { t.Fatal(err) }
	out,err := execute("--config", cfg, "run", "pi", "--quiet", "--", "--version"); if err != nil { t.Fatal(err) }
	if strings.Contains(out,"[sub-switch]") { t.Fatalf("quiet printed banner: %s", out) }
	b,err := os.ReadFile(outFile); if err != nil { t.Fatal(err) }
	if !strings.Contains(string(b),"args:--version") || !strings.Contains(string(b),"profiles/company/pi/config") { t.Fatalf("bad fake output: %s", b) }
}
