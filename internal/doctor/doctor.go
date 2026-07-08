package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/florian-balling/sub-switch/internal/config"
	"github.com/florian-balling/sub-switch/internal/launcher"
	"github.com/florian-balling/sub-switch/internal/wrappers"
)

type Severity string

const (
	OK    Severity = "ok"
	Warn  Severity = "warn"
	Error Severity = "error"
)

type CheckResult struct {
	Severity Severity
	Message  string
}

func Run(configPath, wrapperDir string) []CheckResult {
	var out []CheckResult
	c, path, err := config.Load(configPath)
	if err != nil {
		return []CheckResult{{Error, fmt.Sprintf("config failed: %v", err)}}
	}
	out = append(out, CheckResult{OK, "config loaded: " + path})
	for agent, ac := range c.Agents {
		if err := launcher.ValidateCommand(ac.Command); err != nil {
			out = append(out, CheckResult{Error, fmt.Sprintf("configured command for %s invalid: %v", agent, err)})
		} else {
			out = append(out, CheckResult{OK, fmt.Sprintf("%s command exists: %s", agent, ac.Command)})
		}
		if wrapperDir != "" {
			wp := filepath.Join(wrapperDir, agent)
			if _, err := os.Stat(wp); err != nil {
				out = append(out, CheckResult{Warn, fmt.Sprintf("missing wrapper for %s: %s", agent, wp)})
			} else if !wrappers.IsManagedFile(wp) {
				out = append(out, CheckResult{Error, fmt.Sprintf("wrapper for %s is not managed: %s", agent, wp)})
			} else {
				out = append(out, CheckResult{OK, fmt.Sprintf("wrapper exists for %s: %s", agent, wp)})
			}
			if active, err := exec.LookPath(agent); err == nil && filepath.Clean(active) != filepath.Clean(wp) {
				out = append(out, CheckResult{Warn, fmt.Sprintf("active %s resolves to %s, expected wrapper in %s", agent, active, wrapperDir)})
			}
		}
	}
	return out
}
func HasError(rs []CheckResult) bool {
	for _, r := range rs {
		if r.Severity == Error {
			return true
		}
	}
	return false
}
