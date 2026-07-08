package launcher

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/florian-balling/sub-switch/internal/wrappers"
)

func TestRunUsesProcessStandardFilesWhenSpecFilesAreNil(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	p := filepath.Join(t.TempDir(), "agent")
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho stdout\necho stderr >&2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Run(CommandSpec{Command: p}); err != nil {
		t.Fatalf("Run() with nil stdio returned error: %v", err)
	}
}

func TestValidateCommandRefusesManagedWrapper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell perms")
	}
	p := filepath.Join(t.TempDir(), "pi")
	if err := os.WriteFile(p, []byte(wrappers.Content("/bin/sub-switch", "pi")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCommand(p); err == nil {
		t.Fatal("expected recursion error")
	}
}
