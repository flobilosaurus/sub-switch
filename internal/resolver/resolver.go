package resolver

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/florian-balling/sub-switch/internal/config"
)

type Selection struct {
	Agent       string
	CWD         string
	ProjectPath string
	Profile     string
	Reason      string
	Denied      bool
}

func Resolve(c config.Config, cwd, agent string) (Selection, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return Selection{}, err
	}
	abs = filepath.Clean(abs)
	var best *config.ProjectRule
	bestLen := -1
	for i := range c.Projects {
		p := filepath.Clean(c.Projects[i].Path)
		if matches(abs, p) && len(p) > bestLen {
			best = &c.Projects[i]
			bestLen = len(p)
		}
	}
	if best == nil {
		return Selection{Agent: agent, CWD: abs, Denied: true, Reason: fmt.Sprintf("no project rule matches %s", abs)}, nil
	}
	prof := best.Profiles[agent]
	if prof == "" {
		return Selection{Agent: agent, CWD: abs, ProjectPath: best.Path, Denied: true, Reason: fmt.Sprintf("project %s has no profile for %s", best.Path, agent)}, nil
	}
	return Selection{Agent: agent, CWD: abs, ProjectPath: best.Path, Profile: prof}, nil
}

func matches(cwd, root string) bool {
	rel, err := filepath.Rel(root, cwd)
	return err == nil && (rel == "." || (!strings.HasPrefix(rel, "..") && rel != ".."))
}
