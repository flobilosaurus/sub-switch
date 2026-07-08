package profile

import (
	"os"
	"path/filepath"
)

type Env struct{ ConfigHome, CacheHome, DataHome string }

func Build(home, profileName, agent string) Env {
	baseShare := filepath.Join(home, ".local", "share", "sub-switch", "profiles", profileName, agent)
	return Env{ConfigHome: filepath.Join(baseShare, "config"), CacheHome: filepath.Join(home, ".cache", "sub-switch", "profiles", profileName, agent, "cache"), DataHome: filepath.Join(baseShare, "data")}
}

func BuildForCurrentUser(profileName, agent string) (Env, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return Env{}, err
	}
	return Build(h, profileName, agent), nil
}
func (e Env) Ensure() error {
	for _, d := range []string{e.ConfigHome, e.CacheHome, e.DataHome} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			return err
		}
	}
	return nil
}
func (e Env) Merge(base []string) []string {
	out := make([]string, 0, len(base)+3)
	for _, kv := range base {
		if len(kv) >= 16 && (kv[:16] == "XDG_CONFIG_HOME=") {
			continue
		}
		if len(kv) >= 15 && (kv[:15] == "XDG_CACHE_HOME=") {
			continue
		}
		if len(kv) >= 14 && (kv[:14] == "XDG_DATA_HOME=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, "XDG_CONFIG_HOME="+e.ConfigHome, "XDG_CACHE_HOME="+e.CacheHome, "XDG_DATA_HOME="+e.DataHome)
}
