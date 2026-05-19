package update

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCheckVersionSummarizesNewerReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/neko233-com/acme-go/releases":
			if err := json.NewEncoder(w).Encode([]githubRelease{
				{TagName: "v0.2.0", Name: "0.2.0", Body: "- add upgrade command\n- add version diff", HTMLURL: "https://example.com/v0.2.0"},
				{TagName: "v0.1.1", Name: "0.1.1", Body: "- dns provider aliases", HTMLURL: "https://example.com/v0.1.1"},
			}); err != nil {
				t.Fatalf("encode releases: %v", err)
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}
	info, err := client.CheckVersion("0.1.0")
	if err != nil {
		t.Fatalf("CheckVersion: %v", err)
	}
	if info.UpToDate {
		t.Fatal("expected update to be available")
	}
	if info.LatestVersion != "v0.2.0" {
		t.Fatalf("latest version: got %q", info.LatestVersion)
	}
	if len(info.Changes) != 2 {
		t.Fatalf("changes: got %d want 2", len(info.Changes))
	}
	if info.Changes[0].Summary != "add upgrade command" {
		t.Fatalf("unexpected summary %q", info.Changes[0].Summary)
	}
}

func TestCheckVersionHandlesOfflineGracefully(t *testing.T) {
	client := Client{BaseURL: "http://127.0.0.1:1", HTTPClient: &http.Client{}}
	info, err := client.CheckVersion("0.1.0")
	if err != nil {
		t.Fatalf("CheckVersion: %v", err)
	}
	if info.Warning == "" {
		t.Fatal("expected warning for offline check")
	}
}

func TestSelectAssetMatchesCurrentNaming(t *testing.T) {
	release := githubRelease{
		TagName: "v0.2.0",
		Assets: []githubReleaseAsset{
			{Name: "acme-go_linux_amd64.tar.gz"},
			{Name: "acme-go_windows_amd64.zip"},
		},
	}
	asset, err := selectAsset(release, "windows", "amd64")
	if err != nil {
		t.Fatalf("selectAsset: %v", err)
	}
	if asset.Name != "acme-go_windows_amd64.zip" {
		t.Fatalf("asset: got %q", asset.Name)
	}
}

func TestWindowsUpgradeScriptContainsPaths(t *testing.T) {
	script := windowsUpgradeScript(`C:\apps\acme-go.exe`, `C:\temp\acme-go.exe`, `C:\apps\acme-go.exe.bak`)
	for _, token := range []string{"TARGET=C:\\apps\\acme-go.exe", "SOURCE=C:\\temp\\acme-go.exe", "BACKUP=C:\\apps\\acme-go.exe.bak", ":waitloop"} {
		if !strings.Contains(script, token) {
			t.Fatalf("script missing %q", token)
		}
	}
}

func TestAutoUpdateEnabledDefaultsToTrue(t *testing.T) {
	t.Setenv(autoUpdateEnv, "")
	if !AutoUpdateEnabled() {
		t.Fatal("expected auto update to default to enabled")
	}
	t.Setenv(autoUpdateEnv, "false")
	if AutoUpdateEnabled() {
		t.Fatal("expected auto update to be disabled by env")
	}
}

func TestMaybeAutoUpdateSkipsFreshCache(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		t.Fatalf("unexpected network call to %s", r.URL.Path)
	}))
	defer server.Close()

	now := time.Now().UTC()
	cachePath := filepath.Join(t.TempDir(), "auto-update.json")
	if err := writeAutoUpdateState(cachePath, autoUpdateState{CheckedAt: now}); err != nil {
		t.Fatalf("writeAutoUpdateState: %v", err)
	}

	client := Client{BaseURL: server.URL, HTTPClient: server.Client()}
	var out bytes.Buffer
	err := client.MaybeAutoUpdate(AutoUpdateOptions{
		CurrentVersion: "0.1.0",
		ExecutablePath: filepath.Join(t.TempDir(), "acme-go"),
		CachePath:      cachePath,
		Out:            &out,
		Now:            func() time.Time { return now.Add(time.Hour) },
	})
	if err != nil {
		t.Fatalf("MaybeAutoUpdate: %v", err)
	}
	if hits != 0 {
		t.Fatalf("unexpected network hits: %d", hits)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output, got %q", out.String())
	}
}
