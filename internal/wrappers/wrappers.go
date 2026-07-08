package wrappers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const Marker = "# managed by sub-switch; do not edit"

type Result struct {
	Err    error
	Agent  string
	Path   string
	Action string
}

func Content(subSwitchPath, agent string) string {
	return fmt.Sprintf("#!/bin/sh\n%s\nexec %s run %s -- \"$@\"\n", Marker, shellQuote(subSwitchPath), shellQuote(agent))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func IsManagedFile(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), Marker)
}

func Install(dir, subSwitchPath string, agents []string, force bool) ([]Result, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if subSwitchPath == "" {
		subSwitchPath = "sub-switch"
	}
	if abs, err := filepath.Abs(subSwitchPath); err == nil {
		subSwitchPath = abs
	}
	agents = sortedUniqueAgents(agents)
	results := make([]Result, 0, len(agents))
	for _, a := range agents {
		p := filepath.Join(dir, a)
		content := []byte(Content(subSwitchPath, a))
		action := "created"
		if _, err := os.Stat(p); err == nil {
			if !force && !IsManagedFile(p) {
				results = append(results, Result{Agent: a, Path: p, Action: "refused", Err: fmt.Errorf("refusing to overwrite unrelated file")})
				continue
			}
			action = "updated"
		} else if !os.IsNotExist(err) {
			results = append(results, Result{Agent: a, Path: p, Action: "error", Err: err})
			continue
		}
		if err := os.WriteFile(p, content, 0o755); err != nil {
			results = append(results, Result{Agent: a, Path: p, Action: "error", Err: err})
			continue
		}
		results = append(results, Result{Agent: a, Path: p, Action: action})
	}
	return results, nil
}

func sortedUniqueAgents(agents []string) []string {
	seen := make(map[string]struct{}, len(agents))
	out := make([]string, 0, len(agents))
	for _, agent := range agents {
		if _, ok := seen[agent]; ok {
			continue
		}
		seen[agent] = struct{}{}
		out = append(out, agent)
	}
	sort.Strings(out)
	return out
}
