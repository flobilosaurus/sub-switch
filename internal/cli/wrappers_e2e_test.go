package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWrapperE2EHappyPath(t *testing.T) {
	skipWrapperE2EOnWindows(t)

	root := t.TempDir()
	home := filepath.Join(root, "home")
	xdgConfig := filepath.Join(root, "xdg-config")
	project := filepath.Join(root, "project")
	wrapperDir := filepath.Join(root, "wrappers")
	recordsDir := filepath.Join(root, "records")
	realAgentsDir := filepath.Join(root, "real-agents")
	recordFile := filepath.Join(recordsDir, "pi.env")
	fakeAgent := filepath.Join(realAgentsDir, "fake-pi")
	for _, dir := range []string{home, filepath.Join(xdgConfig, "sub-switch"), project, wrapperDir, recordsDir, realAgentsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	subSwitch := buildTestSubSwitch(t)
	out, err := runCmd(t, root, testEnv(home, xdgConfig), subSwitch, "install-wrappers", "--dir", wrapperDir)
	if err != nil {
		t.Fatalf("install wrappers failed: %v\n%s", err, out)
	}

	writeFakeAgent(t, fakeAgent, recordFile)
	writeConfig(t, filepath.Join(xdgConfig, "sub-switch", "config.yaml"), project, fakeAgent, "company")

	out, err = runCmd(t, project, testEnv(home, xdgConfig), filepath.Join(wrapperDir, "pi"), "--version", "--json")
	if err != nil {
		t.Fatalf("wrapper failed: %v\n%s", err, out)
	}

	record := readRecord(t, recordFile)
	if !strings.Contains(record, "args:--version --json\n") {
		t.Fatalf("forwarded args not recorded in order:\n%s", record)
	}
	wantCWD, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(record, "cwd:"+wantCWD+"\n") {
		t.Fatalf("cwd mismatch, want %s:\n%s", wantCWD, record)
	}
	assertRecordContains(t, record, "XDG_CONFIG_HOME:"+filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "pi", "config")+"\n")
	assertRecordContains(t, record, "XDG_CACHE_HOME:"+filepath.Join(home, ".cache", "sub-switch", "profiles", "company", "pi", "cache")+"\n")
	assertRecordContains(t, record, "XDG_DATA_HOME:"+filepath.Join(home, ".local", "share", "sub-switch", "profiles", "company", "pi", "data")+"\n")
}

func TestWrapperE2EDeniedFromUnmatchedDirectory(t *testing.T) {
	skipWrapperE2EOnWindows(t)

	root := t.TempDir()
	home := filepath.Join(root, "home")
	xdgConfig := filepath.Join(root, "xdg-config")
	project := filepath.Join(root, "project")
	unknown := filepath.Join(root, "unknown")
	wrapperDir := filepath.Join(root, "wrappers")
	recordsDir := filepath.Join(root, "records")
	realAgentsDir := filepath.Join(root, "real-agents")
	recordFile := filepath.Join(recordsDir, "pi.env")
	fakeAgent := filepath.Join(realAgentsDir, "fake-pi")
	for _, dir := range []string{home, filepath.Join(xdgConfig, "sub-switch"), project, unknown, wrapperDir, recordsDir, realAgentsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	subSwitch := buildTestSubSwitch(t)
	out, err := runCmd(t, root, testEnv(home, xdgConfig), subSwitch, "install-wrappers", "--dir", wrapperDir)
	if err != nil {
		t.Fatalf("install wrappers failed: %v\n%s", err, out)
	}

	writeFakeAgent(t, fakeAgent, recordFile)
	writeConfig(t, filepath.Join(xdgConfig, "sub-switch", "config.yaml"), project, fakeAgent, "company")

	out, err = runCmd(t, unknown, testEnv(home, xdgConfig), filepath.Join(wrapperDir, "pi"), "--version")
	if err == nil {
		t.Fatalf("wrapper unexpectedly succeeded:\n%s", out)
	}
	if !strings.Contains(out, "[sub-switch] denied") || !strings.Contains(out, "no project rule matches") {
		t.Fatalf("denial output missing expected text:\n%s", out)
	}
	if _, statErr := os.Stat(recordFile); !os.IsNotExist(statErr) {
		t.Fatalf("fake agent should not have been invoked, stat err: %v", statErr)
	}
}

func TestWrapperE2EManagedWrapperRecursionGuard(t *testing.T) {
	skipWrapperE2EOnWindows(t)

	root := t.TempDir()
	home := filepath.Join(root, "home")
	xdgConfig := filepath.Join(root, "xdg-config")
	project := filepath.Join(root, "project")
	wrapperDir := filepath.Join(root, "wrappers")
	recordsDir := filepath.Join(root, "records")
	realAgentsDir := filepath.Join(root, "real-agents")
	recordFile := filepath.Join(recordsDir, "pi.env")
	fakeAgent := filepath.Join(realAgentsDir, "fake-pi")
	for _, dir := range []string{home, filepath.Join(xdgConfig, "sub-switch"), project, wrapperDir, recordsDir, realAgentsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	subSwitch := buildTestSubSwitch(t)
	out, err := runCmd(t, root, testEnv(home, xdgConfig), subSwitch, "install-wrappers", "--dir", wrapperDir)
	if err != nil {
		t.Fatalf("install wrappers failed: %v\n%s", err, out)
	}

	writeFakeAgent(t, fakeAgent, recordFile)
	writeConfig(t, filepath.Join(xdgConfig, "sub-switch", "config.yaml"), project, filepath.Join(wrapperDir, "pi"), "company")

	out, err = runCmd(t, project, testEnv(home, xdgConfig), filepath.Join(wrapperDir, "pi"), "--version")
	if err == nil {
		t.Fatalf("wrapper unexpectedly succeeded:\n%s", out)
	}
	if !strings.Contains(out, "configured command points to a managed sub-switch wrapper") {
		t.Fatalf("recursion guard output missing expected text:\n%s", out)
	}
	if _, statErr := os.Stat(recordFile); !os.IsNotExist(statErr) {
		t.Fatalf("fake agent should not have been invoked, stat err: %v", statErr)
	}
}

func skipWrapperE2EOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell wrappers")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func buildTestSubSwitch(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "sub-switch-test")
	cmd := exec.Command("go", "build", "-o", exe, "./cmd/sub-switch")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return exe
}

func runCmd(t *testing.T, dir string, env []string, exe string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), ctx.Err()
	}
	return string(out), err
}

func testEnv(home, xdgConfig string) []string {
	env := append([]string{}, os.Environ()...)
	env = appendWithoutPrefixes(env, "HOME=", "XDG_CONFIG_HOME=", "XDG_CACHE_HOME=", "XDG_DATA_HOME=")
	return append(env, "HOME="+home, "XDG_CONFIG_HOME="+xdgConfig)
}

func appendWithoutPrefixes(env []string, prefixes ...string) []string {
	out := env[:0]
	for _, kv := range env {
		keep := true
		for _, prefix := range prefixes {
			if strings.HasPrefix(kv, prefix) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, kv)
		}
	}
	return out
}

func writeFakeAgent(t *testing.T, path, recordFile string) {
	t.Helper()
	script := fmt.Sprintf("#!/bin/sh\n{\n  printf 'args:%%s\\n' \"$*\"\n  printf 'cwd:%%s\\n' \"$(pwd -P)\"\n  printf 'XDG_CONFIG_HOME:%%s\\n' \"$XDG_CONFIG_HOME\"\n  printf 'XDG_CACHE_HOME:%%s\\n' \"$XDG_CACHE_HOME\"\n  printf 'XDG_DATA_HOME:%%s\\n' \"$XDG_DATA_HOME\"\n} > %q\n", recordFile)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeConfig(t *testing.T, cfg, projectPath, realAgentPath, profile string) {
	t.Helper()
	text := fmt.Sprintf("default: deny\nui:\n  startup_banner: true\nagents:\n  pi:\n    command: %q\nprojects:\n  - path: %q\n    profiles:\n      pi: %q\n", realAgentPath, projectPath, profile)
	if err := os.WriteFile(cfg, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readRecord(t *testing.T, recordFile string) string {
	t.Helper()
	b, err := os.ReadFile(recordFile)
	if err != nil {
		t.Fatalf("reading fake-agent record: %v", err)
	}
	return string(b)
}

func assertRecordContains(t *testing.T, record, want string) {
	t.Helper()
	if !strings.Contains(record, want) {
		t.Fatalf("record missing %q:\n%s", want, record)
	}
}
