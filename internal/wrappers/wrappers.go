package wrappers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var SupportedAgents = []string{"pi", "claude", "codex", "opencode"}

const Marker = "# managed by sub-switch; do not edit"

type Result struct {
	Err    error
	Agent  string
	Path   string
	Action string
}

func Content(subSwitchPath, agent string) string {
	return fmt.Sprintf("#!/bin/sh\n%s\nexec %q run %s -- \"$@\"\n", Marker, subSwitchPath, agent)
}
func IsManagedFile(path string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), Marker)
}

func Install(dir, subSwitchPath string, force bool) ([]Result, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if subSwitchPath == "" {
		subSwitchPath = "sub-switch"
	}
	if abs, err := filepath.Abs(subSwitchPath); err == nil {
		subSwitchPath = abs
	}
	results := make([]Result, 0, len(SupportedAgents))
	for _, a := range SupportedAgents {
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
