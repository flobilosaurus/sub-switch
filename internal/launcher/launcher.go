package launcher

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/florian-balling/sub-switch/internal/wrappers"
)

type CommandSpec struct { Command string; Args, Env []string; CWD string; Stdout, Stderr *os.File; Stdin *os.File }

func ValidateCommand(path string) error { st, err := os.Stat(path); if err != nil { return fmt.Errorf("configured command does not exist: %s", path) }; if st.IsDir() { return fmt.Errorf("configured command is a directory: %s", path) }; if wrappers.IsManagedFile(path) { return fmt.Errorf("configured command points to a managed sub-switch wrapper; this would recurse: %s", path) }; return nil }
func Run(s CommandSpec) error { if err := ValidateCommand(s.Command); err != nil { return err }; cmd := exec.Command(s.Command, s.Args...); cmd.Env=s.Env; cmd.Dir=s.CWD; cmd.Stdout=s.Stdout; if cmd.Stdout==nil { cmd.Stdout=os.Stdout }; cmd.Stderr=s.Stderr; if cmd.Stderr==nil { cmd.Stderr=os.Stderr }; cmd.Stdin=s.Stdin; if cmd.Stdin==nil { cmd.Stdin=os.Stdin }; return cmd.Run() }
