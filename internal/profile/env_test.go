package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildEnsureMerge(t *testing.T) {
	home := t.TempDir()
	e := Build(home, "company", "pi")
	if e.ConfigHome != filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "pi", "config") {
		t.Fatalf("bad config home: %s", e.ConfigHome)
	}
	if err := e.Ensure(); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{e.ConfigHome, e.CacheHome, e.DataHome} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			t.Fatalf("missing dir %s", d)
		}
	}
	merged := e.Merge([]string{"A=B", "XDG_CONFIG_HOME=old"})
	for _, kv := range merged {
		if kv == "XDG_CONFIG_HOME=old" {
			t.Fatal("old env preserved")
		}
	}
}
