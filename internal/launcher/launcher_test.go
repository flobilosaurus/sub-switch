package launcher

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/florian-balling/sub-switch/internal/wrappers"
)

func TestValidateCommandRefusesManagedWrapper(t *testing.T) {
	if runtime.GOOS == "windows" { t.Skip("shell perms") }
	p := filepath.Join(t.TempDir(),"pi"); if err := os.WriteFile(p, []byte(wrappers.Content("/bin/sub-switch","pi")), 0o755); err != nil { t.Fatal(err) }
	if err := ValidateCommand(p); err == nil { t.Fatal("expected recursion error") }
}
