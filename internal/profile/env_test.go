package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildEnsureMerge(t *testing.T) {
	home := t.TempDir()
	e := Build(home, "company", "pi")
	if e.ConfigHome != filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "pi", "config") {
		t.Fatalf("bad config home: %s", e.ConfigHome)
	}
	if err := e.Ensure(); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{e.ConfigHome, e.CacheHome, e.DataHome, e.AgentEnv["PI_CODING_AGENT_DIR"], e.AgentEnv["PI_CODING_AGENT_SESSION_DIR"]} {
		if st, err := os.Stat(d); err != nil || !st.IsDir() {
			t.Fatalf("missing dir %s", d)
		}
	}
	merged := e.Merge([]string{"A=B", "XDG_CONFIG_HOME=old"})
	if envMap(merged)["XDG_CONFIG_HOME"] == "old" {
		t.Fatal("old env preserved")
	}
}

func TestMergeScrubsAndInjects(t *testing.T) {
	home := t.TempDir()
	e := BuildWithEnv(home, "company", "pi", map[string]string{
		"ANTHROPIC_API_KEY": "company-key",
		"NORMAL":            "profile",
		"EMPTY":             "",
	})
	merged := envMap(e.Merge([]string{
		"NORMAL=parent",
		"KEEP=1",
		"XDG_CACHE_HOME=old",
		"ANTHROPIC_API_KEY=global",
		"OPENAI_API_KEY=global",
		"AWS_PROFILE=prod",
		"AWS_SECRET_ACCESS_KEY=secret",
		"FOO_API_KEY=foo",
		"PI_CODING_AGENT_DIR=old",
		"PI_CODING_AGENT_SESSION_DIR=old",
		"OPENCODE_CONFIG=/tmp/opencode.json",
	}))
	if merged["KEEP"] != "1" {
		t.Fatal("unrelated env not preserved")
	}
	for _, name := range []string{"OPENAI_API_KEY", "AWS_PROFILE", "AWS_SECRET_ACCESS_KEY", "FOO_API_KEY", "OPENCODE_CONFIG"} {
		if _, ok := merged[name]; ok {
			t.Fatalf("%s was not scrubbed", name)
		}
	}
	if merged["ANTHROPIC_API_KEY"] != "company-key" || merged["NORMAL"] != "profile" {
		t.Fatalf("profile env did not override/inject: %#v", merged)
	}
	if got, ok := merged["EMPTY"]; !ok || got != "" {
		t.Fatalf("empty profile env missing: %#v", merged)
	}
	if merged["PI_CODING_AGENT_DIR"] == "old" || !strings.HasSuffix(merged["PI_CODING_AGENT_DIR"], filepath.Join("pi", "pi-agent")) {
		t.Fatalf("pi dir not replaced: %s", merged["PI_CODING_AGENT_DIR"])
	}
}

func TestSupportedAgentSpecificVars(t *testing.T) {
	home := t.TempDir()
	tests := []struct {
		agent string
		want  map[string]string
	}{
		{"pi", map[string]string{"PI_CODING_AGENT_DIR": filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "pi", "pi-agent"), "PI_CODING_AGENT_SESSION_DIR": filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "pi", "pi-sessions")}},
		{"codex", map[string]string{"CODEX_HOME": filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "codex", "codex")}},
		{"claude", map[string]string{"CLAUDE_CONFIG_DIR": filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "claude", "claude-config")}},
		{"opencode", map[string]string{}},
		{"gemini", map[string]string{}},
	}
	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			e := BuildWithEnv(home, "company", tt.agent, map[string]string{"TOKEN": "x"})
			merged := envMap(e.Merge([]string{"OPENCODE_CONFIG=old", "CODEX_HOME=old"}))
			for name, want := range tt.want {
				if got := merged[name]; got != want {
					t.Fatalf("%s = %q, want %q", name, got, want)
				}
			}
			if tt.agent != "codex" {
				if got := merged["CODEX_HOME"]; got != "" {
					t.Fatalf("CODEX_HOME should be scrubbed for %s, got %q", tt.agent, got)
				}
			}
			if _, ok := merged["OPENCODE_CONFIG"]; ok {
				t.Fatal("OPENCODE_CONFIG should be scrubbed")
			}
			if merged["TOKEN"] != "x" || merged["XDG_CONFIG_HOME"] == "" || merged["XDG_CACHE_HOME"] == "" || merged["XDG_DATA_HOME"] == "" {
				t.Fatalf("generic env missing: %#v", merged)
			}
		})
	}
}

func envMap(env []string) map[string]string {
	out := map[string]string{}
	for _, kv := range env {
		name, value, _ := strings.Cut(kv, "=")
		out[name] = value
	}
	return out
}
