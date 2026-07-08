package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	dir := t.TempDir(); path := filepath.Join(dir,"config.yaml")
	if err := os.WriteFile(path, []byte("agents:\n  pi:\n    command: /bin/echo\nprojects:\n  - path: .\n    profiles:\n      pi: test\n"), 0o600); err != nil { t.Fatal(err) }
	c,_,err := Load(path); if err != nil { t.Fatal(err) }
	if c.Default != "deny" || !c.UI.StartupBanner { t.Fatalf("defaults not applied: %#v", c) }
}

func TestLoadRejectsInvalidDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(),"config.yaml")
	if err := os.WriteFile(path, []byte("default: allow\n"), 0o600); err != nil { t.Fatal(err) }
	if _,_,err := Load(path); err == nil { t.Fatal("expected error") }
}

func TestSaveRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(),"config.yaml")
	if err := Save(path, StarterConfig(), false); err != nil { t.Fatal(err) }
	if err := Save(path, StarterConfig(), false); err == nil { t.Fatal("expected overwrite refusal") }
	if err := Save(path, StarterConfig(), true); err != nil { t.Fatal(err) }
}
