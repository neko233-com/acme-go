package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	owner          = "neko233-com"
	repo           = "acme-go"
	defaultBaseURL = "https://api.github.com"
	autoUpdateEnv  = "ACME_GO_AUTO_UPDATE"
)

const autoUpdateCooldown = 12 * time.Hour

type ReleaseDiff struct {
	Version string `json:"version"`
	Name    string `json:"name,omitempty"`
	Summary string `json:"summary,omitempty"`
	URL     string `json:"url,omitempty"`
}

type VersionInfo struct {
	CurrentVersion string        `json:"current_version"`
	LatestVersion  string        `json:"latest_version,omitempty"`
	UpToDate       bool          `json:"up_to_date"`
	ReleaseURL     string        `json:"release_url,omitempty"`
	CheckedAt      time.Time     `json:"checked_at"`
	Changes        []ReleaseDiff `json:"changes,omitempty"`
	Warning        string        `json:"warning,omitempty"`
}

type githubRelease struct {
	TagName     string               `json:"tag_name"`
	Name        string               `json:"name"`
	Body        string               `json:"body"`
	HTMLURL     string               `json:"html_url"`
	Draft       bool                 `json:"draft"`
	Prerelease  bool                 `json:"prerelease"`
	Assets      []githubReleaseAsset `json:"assets"`
	PublishedAt time.Time            `json:"published_at"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type AutoUpdateOptions struct {
	CurrentVersion string
	ExecutablePath string
	CachePath      string
	Force          bool
	Out            io.Writer
	Now            func() time.Time
}

type autoUpdateState struct {
	CheckedAt      time.Time `json:"checked_at"`
	CurrentVersion string    `json:"current_version,omitempty"`
	LatestVersion  string    `json:"latest_version,omitempty"`
	Warning        string    `json:"warning,omitempty"`
}

func NewClient() Client {
	return Client{
		BaseURL:    defaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func CheckVersion(currentVersion string) (VersionInfo, error) {
	return NewClient().CheckVersion(currentVersion)
}

func (c Client) CheckVersion(currentVersion string) (VersionInfo, error) {
	info := VersionInfo{
		CurrentVersion: normalizeVersion(currentVersion),
		CheckedAt:      time.Now().UTC(),
	}
	releases, err := c.fetchReleases()
	if err != nil {
		info.Warning = err.Error()
		return info, nil
	}
	latest, ok := latestStableRelease(releases)
	if !ok {
		info.Warning = "no stable GitHub release found"
		return info, nil
	}
	info.LatestVersion = latest.TagName
	info.ReleaseURL = latest.HTMLURL
	if info.CurrentVersion == "dev" {
		info.UpToDate = false
		info.Changes = summarizeChanges(releases, "")
		return info, nil
	}
	info.UpToDate = semver.Compare(canonicalSemver(info.CurrentVersion), canonicalSemver(latest.TagName)) >= 0
	if !info.UpToDate {
		info.Changes = summarizeChanges(releases, info.CurrentVersion)
	}
	return info, nil
}

func Upgrade(currentVersion string, executablePath string, out io.Writer) error {
	return NewClient().Upgrade(currentVersion, executablePath, out)
}

func MaybeAutoUpdate(currentVersion string, executablePath string, out io.Writer) error {
	return NewClient().MaybeAutoUpdate(AutoUpdateOptions{
		CurrentVersion: currentVersion,
		ExecutablePath: executablePath,
		Out:            out,
	})
}

func (c Client) Upgrade(currentVersion string, executablePath string, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	info, err := c.CheckVersion(currentVersion)
	if err != nil {
		return err
	}
	if info.Warning != "" && info.LatestVersion == "" {
		return fmt.Errorf("check latest version: %s", info.Warning)
	}
	if info.UpToDate {
		fmt.Fprintf(out, "already on latest version %s\n", info.CurrentVersion)
		return nil
	}
	if info.LatestVersion == "" {
		return fmt.Errorf("latest version not available")
	}
	return c.upgradeToVersion(currentVersion, executablePath, info, out)
}

func (c Client) MaybeAutoUpdate(options AutoUpdateOptions) error {
	if options.Out == nil {
		options.Out = io.Discard
	}
	if !AutoUpdateEnabled() {
		return nil
	}
	currentVersion := normalizeVersion(options.CurrentVersion)
	if currentVersion == "dev" {
		return nil
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	cachePath, err := resolveAutoUpdateCachePath(options.CachePath, options.ExecutablePath)
	if err != nil {
		return fmt.Errorf("resolve auto update cache path: %w", err)
	}
	if !options.Force {
		state, err := readAutoUpdateState(cachePath)
		if err == nil && options.Now().Sub(state.CheckedAt) < autoUpdateCooldown {
			return nil
		}
	}

	info, err := c.CheckVersion(currentVersion)
	if err != nil {
		return err
	}
	state := autoUpdateState{
		CheckedAt:      options.Now().UTC(),
		CurrentVersion: currentVersion,
		LatestVersion:  info.LatestVersion,
		Warning:        info.Warning,
	}
	if err := writeAutoUpdateState(cachePath, state); err != nil {
		fmt.Fprintf(options.Out, "warning: persist auto update state: %v\n", err)
	}
	if info.Warning != "" && info.LatestVersion == "" {
		fmt.Fprintf(options.Out, "warning: auto update check skipped: %s\n", info.Warning)
		return nil
	}
	if info.UpToDate || info.LatestVersion == "" {
		return nil
	}
	fmt.Fprintf(options.Out, "auto update: upgrading from %s to %s\n", currentVersion, info.LatestVersion)
	if err := c.upgradeToVersion(currentVersion, options.ExecutablePath, info, options.Out); err != nil {
		return err
	}
	state.CurrentVersion = info.LatestVersion
	state.Warning = ""
	if err := writeAutoUpdateState(cachePath, state); err != nil {
		fmt.Fprintf(options.Out, "warning: persist auto update state: %v\n", err)
	}
	return nil
}

func (c Client) upgradeToVersion(currentVersion string, executablePath string, info VersionInfo, out io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	release, err := c.fetchReleaseByTag(info.LatestVersion)
	if err != nil {
		return err
	}
	asset, err := selectAsset(release, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "acme-go-upgrade-")
	if err != nil {
		return fmt.Errorf("create upgrade temp dir: %w", err)
	}
	archivePath := filepath.Join(tempDir, asset.Name)
	fmt.Fprintf(out, "downloading %s\n", asset.Name)
	if err := c.downloadFile(archivePath, asset.BrowserDownloadURL); err != nil {
		return err
	}
	newBinaryPath, err := extractBinary(archivePath, tempDir)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "upgrading from %s to %s\n", normalizeVersion(currentVersion), info.LatestVersion)
	if runtime.GOOS == "windows" {
		return scheduleWindowsReplacement(executablePath, newBinaryPath, out)
	}
	return replaceBinary(executablePath, newBinaryPath)
}

func (c Client) fetchReleases() ([]githubRelease, error) {
	endpoint := strings.TrimRight(c.baseURL(), "/") + "/repos/" + owner + "/" + repo + "/releases?per_page=10"
	var releases []githubRelease
	if err := c.getJSON(endpoint, &releases); err != nil {
		return nil, fmt.Errorf("fetch releases: %w", err)
	}
	return releases, nil
}

func (c Client) fetchReleaseByTag(tag string) (githubRelease, error) {
	endpoint := strings.TrimRight(c.baseURL(), "/") + "/repos/" + owner + "/" + repo + "/releases/tags/" + tag
	var release githubRelease
	if err := c.getJSON(endpoint, &release); err != nil {
		return githubRelease{}, fmt.Errorf("fetch release %s: %w", tag, err)
	}
	return release, nil
}

func (c Client) getJSON(url string, target any) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", repo+"-updater")
	if token := githubToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.httpClient().Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("unexpected status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func (c Client) downloadFile(path, url string) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", repo+"-updater")
	if token := githubToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.httpClient().Do(request)
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("download asset status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create asset file: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, response.Body); err != nil {
		return fmt.Errorf("write asset file: %w", err)
	}
	return nil
}

func latestStableRelease(releases []githubRelease) (githubRelease, bool) {
	for _, release := range releases {
		if !release.Draft && !release.Prerelease && semver.IsValid(canonicalSemver(release.TagName)) {
			return release, true
		}
	}
	return githubRelease{}, false
}

func summarizeChanges(releases []githubRelease, currentVersion string) []ReleaseDiff {
	current := canonicalSemver(currentVersion)
	changes := make([]ReleaseDiff, 0)
	for _, release := range releases {
		if release.Draft || release.Prerelease || !semver.IsValid(canonicalSemver(release.TagName)) {
			continue
		}
		if current != "" && semver.Compare(canonicalSemver(release.TagName), current) <= 0 {
			continue
		}
		changes = append(changes, ReleaseDiff{
			Version: release.TagName,
			Name:    release.Name,
			Summary: summarizeReleaseBody(release.Body),
			URL:     release.HTMLURL,
		})
	}
	return changes
}

func summarizeReleaseBody(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "-"))
		if line != "" {
			if len(line) > 160 {
				return line[:160]
			}
			return line
		}
	}
	return ""
}

func selectAsset(release githubRelease, goos, goarch string) (githubReleaseAsset, error) {
	wanted := assetName(goos, goarch)
	for _, asset := range release.Assets {
		if asset.Name == wanted {
			return asset, nil
		}
	}
	return githubReleaseAsset{}, fmt.Errorf("release %s does not contain asset %s", release.TagName, wanted)
}

func assetName(goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("acme-go_%s_%s.zip", goos, goarch)
	}
	return fmt.Sprintf("acme-go_%s_%s.tar.gz", goos, goarch)
}

func extractBinary(archivePath, tempDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractBinaryFromZip(archivePath, tempDir)
	}
	return extractBinaryFromTarGz(archivePath, tempDir)
}

func extractBinaryFromZip(archivePath, tempDir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}
	defer reader.Close()
	binaryName := expectedBinaryName(runtime.GOOS)
	for _, file := range reader.File {
		if filepath.Base(file.Name) != binaryName {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("open zip file: %w", err)
		}
		defer rc.Close()
		outPath := filepath.Join(tempDir, binaryName)
		if err := writeExtractedFile(outPath, rc, file.Mode()); err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("binary %s not found in zip", binaryName)
}

func extractBinaryFromTarGz(archivePath, tempDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open tar.gz: %w", err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	binaryName := expectedBinaryName(runtime.GOOS)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read tar entry: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}
		outPath := filepath.Join(tempDir, binaryName)
		if err := writeExtractedFile(outPath, tarReader, os.FileMode(header.Mode)); err != nil {
			return "", err
		}
		return outPath, nil
	}
	return "", fmt.Errorf("binary %s not found in tar.gz", binaryName)
}

func writeExtractedFile(path string, reader io.Reader, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("create extracted binary: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("write extracted binary: %w", err)
	}
	if mode != 0 {
		if err := os.Chmod(path, mode|0o755); err != nil {
			return fmt.Errorf("chmod extracted binary: %w", err)
		}
	}
	return nil
}

func replaceBinary(targetPath, newBinaryPath string) error {
	backupPath := targetPath + ".bak"
	_ = os.Remove(backupPath)
	if err := os.Rename(targetPath, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(newBinaryPath, targetPath); err != nil {
		_ = os.Rename(backupPath, targetPath)
		return fmt.Errorf("replace binary: %w", err)
	}
	_ = os.Remove(backupPath)
	return nil
}

func scheduleWindowsReplacement(targetPath, newBinaryPath string, out io.Writer) error {
	scriptPath := filepath.Join(filepath.Dir(newBinaryPath), "upgrade.cmd")
	backupPath := targetPath + ".bak"
	script := windowsUpgradeScript(targetPath, newBinaryPath, backupPath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return fmt.Errorf("write upgrade script: %w", err)
	}
	command := exec.Command("cmd", "/C", "start", "", "/MIN", "cmd", "/C", scriptPath)
	if err := command.Start(); err != nil {
		return fmt.Errorf("start upgrade script: %w", err)
	}
	fmt.Fprintln(out, "upgrade scheduled, current process must exit before replacement completes")
	return nil
}

func windowsUpgradeScript(targetPath, newBinaryPath, backupPath string) string {
	quote := func(value string) string {
		return strings.ReplaceAll(value, `"`, `"`)
	}
	return "@echo off\r\n" +
		"setlocal\r\n" +
		fmt.Sprintf("set \"TARGET=%s\"\r\n", quote(targetPath)) +
		fmt.Sprintf("set \"SOURCE=%s\"\r\n", quote(newBinaryPath)) +
		fmt.Sprintf("set \"BACKUP=%s\"\r\n", quote(backupPath)) +
		":waitloop\r\n" +
		"move /Y \"%TARGET%\" \"%BACKUP%\" >nul 2>nul\r\n" +
		"if errorlevel 1 (\r\n" +
		"  ping 127.0.0.1 -n 2 >nul\r\n" +
		"  goto waitloop\r\n" +
		")\r\n" +
		"move /Y \"%SOURCE%\" \"%TARGET%\" >nul\r\n" +
		"del /Q \"%BACKUP%\" >nul 2>nul\r\n" +
		"del /Q \"%~f0\" >nul 2>nul\r\n"
}

func expectedBinaryName(goos string) string {
	if goos == "windows" {
		return "acme-go.exe"
	}
	return "acme-go"
}

func normalizeVersion(version string) string {
	value := strings.TrimSpace(version)
	if value == "" {
		return "dev"
	}
	if value == "dev" {
		return value
	}
	if strings.HasPrefix(value, "v") {
		return value
	}
	return "v" + value
}

func canonicalSemver(version string) string {
	if version == "" || version == "dev" {
		return ""
	}
	value := normalizeVersion(version)
	if semver.IsValid(value) {
		return value
	}
	return ""
}

func (c Client) baseURL() string {
	if strings.TrimSpace(c.BaseURL) == "" {
		return defaultBaseURL
	}
	return c.BaseURL
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	defaultClient := http.Client{Timeout: 30 * time.Second}
	return &defaultClient
}

func githubToken() string {
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GH_TOKEN"))
}

func AutoUpdateEnabled() bool {
	value, ok := os.LookupEnv(autoUpdateEnv)
	if !ok {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "1", "true", "yes", "on", "enable", "enabled":
		return true
	case "0", "false", "no", "off", "disable", "disabled":
		return false
	default:
		return true
	}
}

func resolveAutoUpdateCachePath(cachePath, executablePath string) (string, error) {
	if strings.TrimSpace(cachePath) != "" {
		return cachePath, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err == nil && strings.TrimSpace(cacheDir) != "" {
		return filepath.Join(cacheDir, repo, "auto-update.json"), nil
	}
	if strings.TrimSpace(executablePath) == "" {
		return "", fmt.Errorf("empty executable path")
	}
	return filepath.Join(filepath.Dir(executablePath), ".acme-go-auto-update.json"), nil
}

func readAutoUpdateState(path string) (autoUpdateState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return autoUpdateState{}, err
	}
	var state autoUpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		return autoUpdateState{}, fmt.Errorf("parse auto update state: %w", err)
	}
	return state, nil
}

func writeAutoUpdateState(path string, state autoUpdateState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create auto update cache dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode auto update state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write auto update state: %w", err)
	}
	return nil
}
