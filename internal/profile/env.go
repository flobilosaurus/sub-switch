package profile

import (
	"os"
	"path/filepath"
	"strings"
)

type Env struct {
	ProfileEnv map[string]string
	AgentDirs  []string
	AgentEnv   map[string]string
	BaseShare  string
	ConfigHome string
	CacheHome  string
	DataHome   string
}

var exactScrubNames = map[string]struct{}{
	"ANTHROPIC_API_KEY":              {},
	"ANTHROPIC_AUTH_TOKEN":           {},
	"CLAUDE_CODE_OAUTH_TOKEN":        {},
	"OPENAI_API_KEY":                 {},
	"CODEX_API_KEY":                  {},
	"CODEX_ACCESS_TOKEN":             {},
	"GITHUB_TOKEN":                   {},
	"GITLAB_TOKEN":                   {},
	"GITLAB_PAT":                     {},
	"GH_TOKEN":                       {},
	"OPENCODE_API_KEY":               {},
	"PI_CODING_AGENT_DIR":            {},
	"PI_CODING_AGENT_SESSION_DIR":    {},
	"CLAUDE_CONFIG_DIR":              {},
	"CODEX_HOME":                     {},
	"OPENCODE_CONFIG":                {},
	"OPENCODE_CONFIG_DIR":            {},
	"GOOGLE_APPLICATION_CREDENTIALS": {},
	"CLOUDSDK_CONFIG":                {},
	"CLAUDE_CODE_USE_BEDROCK":        {},
	"CLAUDE_CODE_USE_VERTEX":         {},
	"CLAUDE_CODE_USE_FOUNDRY":        {},
}

var scrubPrefixes = []string{"AWS_", "AZURE_", "ARM_", "GOOGLE_", "GCLOUD_", "GCP_"}
var scrubSuffixes = []string{"_API_KEY", "_AUTH_TOKEN", "_ACCESS_TOKEN", "_BEARER_TOKEN"}

func Build(home, profileName, agent string) Env {
	return BuildWithEnv(home, profileName, agent, nil)
}

func BuildWithEnv(home, profileName, agent string, profileEnv map[string]string) Env {
	baseShare := filepath.Join(home, ".local", "share", "sub-switch", "profiles", profileName, agent)
	e := Env{
		ProfileEnv: copyMap(profileEnv),
		AgentEnv:   map[string]string{},
		BaseShare:  baseShare,
		ConfigHome: filepath.Join(baseShare, "config"),
		CacheHome:  filepath.Join(home, ".cache", "sub-switch", "profiles", profileName, agent, "cache"),
		DataHome:   filepath.Join(baseShare, "data"),
	}
	switch agent {
	case "pi":
		e.AgentEnv["PI_CODING_AGENT_DIR"] = filepath.Join(baseShare, "pi-agent")
		e.AgentEnv["PI_CODING_AGENT_SESSION_DIR"] = filepath.Join(baseShare, "pi-sessions")
	case "codex":
		e.AgentEnv["CODEX_HOME"] = filepath.Join(baseShare, "codex")
	case "claude":
		e.AgentEnv["CLAUDE_CONFIG_DIR"] = filepath.Join(baseShare, "claude-config")
	}
	for _, dir := range e.AgentEnv {
		e.AgentDirs = append(e.AgentDirs, dir)
	}
	return e
}

func BuildForCurrentUser(profileName, agent string) (Env, error) {
	return BuildForCurrentUserWithEnv(profileName, agent, nil)
}

func BuildForCurrentUserWithEnv(profileName, agent string, profileEnv map[string]string) (Env, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return Env{}, err
	}
	return BuildWithEnv(h, profileName, agent, profileEnv), nil
}

func (e Env) Ensure() error {
	dirs := []string{e.ConfigHome, e.CacheHome, e.DataHome}
	dirs = append(dirs, e.AgentDirs...)
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func (e Env) Merge(base []string) []string {
	out := make([]string, 0, len(base)+3+len(e.AgentEnv)+len(e.ProfileEnv))
	for _, kv := range base {
		name, _, ok := strings.Cut(kv, "=")
		if !ok || shouldScrub(name) {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "XDG_CONFIG_HOME="+e.ConfigHome, "XDG_CACHE_HOME="+e.CacheHome, "XDG_DATA_HOME="+e.DataHome)
	for _, name := range sortedKeys(e.AgentEnv) {
		out = append(out, name+"="+e.AgentEnv[name])
	}
	for _, name := range sortedKeys(e.ProfileEnv) {
		out = append(out, name+"="+e.ProfileEnv[name])
	}
	return out
}

func shouldScrub(name string) bool {
	if name == "XDG_CONFIG_HOME" || name == "XDG_CACHE_HOME" || name == "XDG_DATA_HOME" {
		return true
	}
	if _, ok := exactScrubNames[name]; ok {
		return true
	}
	for _, prefix := range scrubPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	for _, suffix := range scrubSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func copyMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
