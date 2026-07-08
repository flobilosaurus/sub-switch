package launcher

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/florian-balling/sub-switch/internal/wrappers"
)

type CommandSpec struct {
	Stdout  *os.File
	Stderr  *os.File
	Stdin   *os.File
	Command string
	CWD     string
	Args    []string
	Env     []string
}

func ValidateCommand(path string) error {
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("configured command does not exist: %s", path)
	}
	if st.IsDir() {
		return fmt.Errorf("configured command is a directory: %s", path)
	}
	if wrappers.IsManagedFile(path) {
		return fmt.Errorf("configured command points to a managed sub-switch wrapper; this would recurse: %s", path)
	}
	return nil
}
func Run(s CommandSpec) error {
	if err := ValidateCommand(s.Command); err != nil {
		return err
	}
	cmd := exec.Command(s.Command, s.Args...)
	cmd.Env = s.Env
	cmd.Dir = s.CWD
	if s.Stdout != nil {
		cmd.Stdout = s.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	if s.Stderr != nil {
		cmd.Stderr = s.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}
	if s.Stdin != nil {
		cmd.Stdin = s.Stdin
	} else {
		cmd.Stdin = os.Stdin
	}
	return cmd.Run()
}
