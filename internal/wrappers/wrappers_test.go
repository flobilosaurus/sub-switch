package wrappers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWrappers(t *testing.T) {
	dir := t.TempDir()
	agents := []string{"gemini", "pi"}
	res, err := Install(dir, "/bin/sub-switch", agents, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != len(agents) {
		t.Fatalf("got %d", len(res))
	}
	for _, a := range agents {
		p := filepath.Join(dir, a)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), Marker) || !strings.Contains(string(b), "\"$@\"") || !strings.Contains(string(b), "run '"+a+"'") {
			t.Fatalf("bad wrapper %s", a)
		}
		st, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode()&0o111 == 0 {
			t.Fatalf("not executable")
		}
	}
}

func TestInstallRefusesAndForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "pi")
	if err := os.WriteFile(p, []byte("custom"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Install(dir, "/bin/sub-switch", []string{"pi"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].Action != "refused" {
		t.Fatalf("want refused: %#v", res[0])
	}
	if _, err := Install(dir, "/bin/sub-switch", []string{"pi"}, true); err != nil {
		t.Fatal(err)
	}
	if !IsManagedFile(p) {
		t.Fatal("force did not overwrite")
	}
}
