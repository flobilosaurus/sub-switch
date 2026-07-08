package cli

import "testing"

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()
	if cmd.Use != "sub-switch" {
		t.Fatalf("unexpected use: %s", cmd.Use)
	}
	for _, name := range []string{"init", "which", "run", "install-wrappers", "doctor"} {
		if _, _, err := cmd.Find([]string{name}); err != nil {
			t.Fatalf("missing command %s: %v", name, err)
		}
	}
}
