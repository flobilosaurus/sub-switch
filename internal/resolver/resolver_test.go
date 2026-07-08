package resolver

import (
	"path/filepath"
	"testing"

	"github.com/florian-balling/sub-switch/internal/config"
)

func TestResolveLongestPrefixAndDenials(t *testing.T) {
	root := t.TempDir(); a := filepath.Join(root,"a"); ab := filepath.Join(a,"b")
	c := config.Config{Default:"deny", Projects: []config.ProjectRule{{Path:a, Profiles:map[string]string{"pi":"a"}}, {Path:ab, Profiles:map[string]string{"pi":"ab"}}}}
	sel,err := Resolve(c, filepath.Join(ab,"child"), "pi"); if err != nil { t.Fatal(err) }
	if sel.Profile != "ab" { t.Fatalf("want longest profile ab, got %#v", sel) }
	sel,err = Resolve(c, root, "pi"); if err != nil || !sel.Denied { t.Fatalf("want no match denial: %#v %v", sel, err) }
	sel,err = Resolve(c, a, "claude"); if err != nil || !sel.Denied { t.Fatalf("want missing profile denial: %#v %v", sel, err) }
}
