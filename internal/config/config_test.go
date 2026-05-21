package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testConfigFixture = "config_acme.test.json"
const testLocalConfigFixture = "config_acme.test.local.json"

func TestLoadAppliesDefaultsAndEnvExpansion(t *testing.T) {
	t.Setenv("ACME_EMAIL", "ops@example.com")

	dir := t.TempDir()
	path := copyFixture(t, dir, testConfigFixture, "config_acme.json", 0o644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Account.Email != "ops@example.com" {
		t.Fatalf("unexpected email %q", cfg.Account.Email)
	}
	if cfg.Account.KeyPath == "" {
		t.Fatal("expected default key path")
	}
	if cfg.CA.DirectoryURL != defaultCADirectoryURL {
		t.Fatalf("unexpected CA directory URL %q", cfg.CA.DirectoryURL)
	}
	if cfg.Automation.RenewInterval != defaultAutomationRenewInterval {
		t.Fatalf("unexpected default renew interval %q", cfg.Automation.RenewInterval)
	}
	if cfg.Automation.RetryBackoff != defaultAutomationRetryBackoff {
		t.Fatalf("unexpected default retry backoff %q", cfg.Automation.RetryBackoff)
	}
	if cfg.Automation.MaxRetryBackoff != defaultAutomationMaxRetryBackoff {
		t.Fatalf("unexpected default max retry backoff %q", cfg.Automation.MaxRetryBackoff)
	}
	if cfg.Automation.MaxRetryAttempts == nil || *cfg.Automation.MaxRetryAttempts != defaultAutomationMaxRetryAttempts {
		t.Fatalf("unexpected default max retry attempts %v", cfg.Automation.MaxRetryAttempts)
	}
	if cfg.Certificates[0].OutputDir == "" {
		t.Fatal("expected default output dir")
	}
	if cfg.Certificates[0].RenewBeforeDays != 30 {
		t.Fatalf("unexpected renew window %d", cfg.Certificates[0].RenewBeforeDays)
	}
}

func TestLoadPrefersLocalJSONOverride(t *testing.T) {
	t.Setenv("ACME_EMAIL", "ops@example.com")

	dir := t.TempDir()
	configPath := copyFixture(t, dir, testConfigFixture, "config_acme.json", 0o644)
	copyFixture(t, dir, testLocalConfigFixture, "config_acme.local.json", 0o600)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DNS.Provider != "alidns" {
		t.Fatalf("expected alidns provider, got %q", cfg.DNS.Provider)
	}
	if cfg.DNS.Env["ALICLOUD_ACCESS_KEY"] != "override-key" {
		t.Fatalf("expected override access key, got %q", cfg.DNS.Env["ALICLOUD_ACCESS_KEY"])
	}
	if got := cfg.Certificates[0].Domains[0]; got != "_.neko2public.online" {
		t.Fatalf("expected local certificate override, got %q", got)
	}
}

func TestLoadAllowsDisablingRetriesExplicitly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config_acme.json")
	cfgFile := loadFixtureConfig(t, testConfigFixture)
	cfgFile.Account.Email = "ops@example.com"
	zeroRetryAttempts := 0
	cfgFile.Automation.MaxRetryAttempts = &zeroRetryAttempts
	content, err := json.MarshalIndent(cfgFile, "", "  ")
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Automation.MaxRetryAttempts == nil || *cfg.Automation.MaxRetryAttempts != 0 {
		t.Fatalf("unexpected max retry attempts: %v", cfg.Automation.MaxRetryAttempts)
	}
}

func copyFixture(t *testing.T, dir, sourceName, targetName string, perm os.FileMode) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", sourceName))
	if err != nil {
		t.Fatalf("read fixture %s: %v", sourceName, err)
	}
	targetPath := filepath.Join(dir, targetName)
	if err := os.WriteFile(targetPath, data, perm); err != nil {
		t.Fatalf("write fixture %s: %v", targetName, err)
	}
	return targetPath
}

func loadFixtureConfig(t *testing.T, sourceName string) Config {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", sourceName))
	if err != nil {
		t.Fatalf("read fixture %s: %v", sourceName, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse fixture %s: %v", sourceName, err)
	}
	return cfg
}

func TestCertificatePathsUseConfiguredOutputFiles(t *testing.T) {
	dir := t.TempDir()
	cert := CertificateSpec{
		OutputDir: dir,
		OutputFiles: CertificateFiles{
			CertFile:      "nginx.crt",
			KeyFile:       "private/nginx.key",
			PublicKeyFile: "nginx.pub",
			FullChainFile: filepath.Join(dir, "public", "fullchain.crt"),
			ChainFile:     "ca-chain.crt",
			MetadataFile:  "nginx.meta.json",
		},
	}

	paths := cert.Paths()
	if paths.CertFile != filepath.Join(dir, "nginx.crt") {
		t.Fatalf("cert path: got %q", paths.CertFile)
	}
	if paths.KeyFile != filepath.Join(dir, "private", "nginx.key") {
		t.Fatalf("key path: got %q", paths.KeyFile)
	}
	if paths.PublicKeyFile != filepath.Join(dir, "nginx.pub") {
		t.Fatalf("public key path: got %q", paths.PublicKeyFile)
	}
	if paths.FullChainFile != filepath.Join(dir, "public", "fullchain.crt") {
		t.Fatalf("fullchain path: got %q", paths.FullChainFile)
	}
	if paths.ChainFile != filepath.Join(dir, "ca-chain.crt") {
		t.Fatalf("chain path: got %q", paths.ChainFile)
	}
	if paths.MetadataFile != filepath.Join(dir, "nginx.meta.json") {
		t.Fatalf("metadata path: got %q", paths.MetadataFile)
	}
}

func TestEffectiveDNSMergesGlobalNamedAndCertificateOverrides(t *testing.T) {
	cfg := Config{
		DNS: DNSConfig{
			Provider:    "cloudflare",
			Env:         map[string]string{"GLOBAL_ONLY": "true"},
			Credentials: DNSCredentials{APIToken: "global-token"},
		},
		DNSProviders: map[string]DNSConfig{
			"aliyun-main": {
				Provider:    "alicloud-cn",
				Credentials: DNSCredentials{AccessKey: "provider-ak", SecretKey: "provider-sk"},
			},
		},
	}

	effective, err := cfg.EffectiveDNS(CertificateSpec{
		Name:    "example",
		Domains: []string{"example.com"},
		DNS: DNSConfig{
			ProviderRef: "aliyun-main",
			Env:         map[string]string{"CERT_ONLY": "true"},
		},
	})
	if err != nil {
		t.Fatalf("EffectiveDNS: %v", err)
	}
	if effective.Provider != "alicloud-cn" {
		t.Fatalf("provider: got %q want %q", effective.Provider, "alicloud-cn")
	}
	if effective.Env["GLOBAL_ONLY"] != "true" {
		t.Fatalf("expected GLOBAL_ONLY env to be preserved")
	}
	if effective.Env["CERT_ONLY"] != "true" {
		t.Fatalf("expected CERT_ONLY env to be merged")
	}
	if effective.Env["ALICLOUD_ACCESS_KEY"] != "provider-ak" {
		t.Fatalf("access key env: got %q", effective.Env["ALICLOUD_ACCESS_KEY"])
	}
	if effective.Env["ALICLOUD_SECRET_KEY"] != "provider-sk" {
		t.Fatalf("secret key env: got %q", effective.Env["ALICLOUD_SECRET_KEY"])
	}
}

func TestAutomationRenewIntervalDuration(t *testing.T) {
	interval, err := (AutomationConfig{RenewInterval: "12h"}).RenewIntervalDuration()
	if err != nil {
		t.Fatalf("RenewIntervalDuration: %v", err)
	}
	if interval != 12*time.Hour {
		t.Fatalf("interval: got %s want %s", interval, 12*time.Hour)
	}

	if _, err := (AutomationConfig{RenewInterval: "bad-duration"}).RenewIntervalDuration(); err == nil {
		t.Fatal("expected invalid duration to fail")
	}
}

func TestAutomationRetryDurations(t *testing.T) {
	backoff, err := (AutomationConfig{RetryBackoff: "30s"}).RetryBackoffDuration()
	if err != nil {
		t.Fatalf("RetryBackoffDuration: %v", err)
	}
	if backoff != 30*time.Second {
		t.Fatalf("retry_backoff: got %s want %s", backoff, 30*time.Second)
	}

	maxBackoff, err := (AutomationConfig{MaxRetryBackoff: "10m"}).MaxRetryBackoffDuration()
	if err != nil {
		t.Fatalf("MaxRetryBackoffDuration: %v", err)
	}
	if maxBackoff != 10*time.Minute {
		t.Fatalf("max_retry_backoff: got %s want %s", maxBackoff, 10*time.Minute)
	}
}
