package subswitch_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallScriptDownloadsLatestReleaseAsset(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl is required for install.sh integration test")
	}

	installDir := t.TempDir()
	assetName := fmt.Sprintf("sub-switch-%s-%s", runtime.GOOS, runtime.GOARCH)
	binary := "#!/bin/sh\necho sub-switch test binary \"$@\"\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest/download/"+assetName {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(binary))
	}))
	defer server.Close()

	cmd := exec.Command("sh", "install.sh")
	cmd.Env = append(os.Environ(),
		"INSTALL_DIR="+installDir,
		"SUB_SWITCH_BASE_URL="+server.URL,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	installedPath := filepath.Join(installDir, "sub-switch")
	contents, err := os.ReadFile(installedPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != binary {
		t.Fatalf("unexpected installed content:\n%s", contents)
	}

	info, err := os.Stat(installedPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("unexpected install mode: %v", info.Mode().Perm())
	}
	if !strings.Contains(string(out), "Installed sub-switch to "+installedPath) {
		t.Fatalf("expected success message in output:\n%s", out)
	}
}

func TestInstallScriptDownloadsPinnedVersionAsset(t *testing.T) {
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl is required for install.sh integration test")
	}

	installDir := t.TempDir()
	assetName := fmt.Sprintf("sub-switch-%s-%s", runtime.GOOS, runtime.GOARCH)
	version := "v9.8.7"
	binary := "#!/bin/sh\necho pinned sub-switch\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download/"+version+"/"+assetName {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(binary))
	}))
	defer server.Close()

	cmd := exec.Command("sh", "install.sh")
	cmd.Env = append(os.Environ(),
		"INSTALL_DIR="+installDir,
		"SUB_SWITCH_BASE_URL="+server.URL,
		"SUB_SWITCH_VERSION="+version,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}

	contents, err := os.ReadFile(filepath.Join(installDir, "sub-switch"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != binary {
		t.Fatalf("unexpected installed content:\n%s", contents)
	}
}
