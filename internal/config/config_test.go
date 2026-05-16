package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaultsAndEnvExpansion(t *testing.T) {
	t.Setenv("ACME_EMAIL", "ops@example.com")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `account:
  email: ${ACME_EMAIL}
  accept_tos: true
dns:
  provider: cloudflare
certificates:
  - name: example
    domains:
      - example.com
`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

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
	configPath := filepath.Join(dir, "config.integration.yaml")
	baseConfig := `account:
  email: ${ACME_EMAIL}
  accept_tos: true
dns:
  provider: cloudflare
certificates:
  - name: example
    domains:
      - example.com
`
	localOverride := `{
  "dns": {
    "provider": "alidns",
    "env": {
      "ALICLOUD_ACCESS_KEY": "override-key"
    }
  },
  "certificates": [
    {
      "name": "example",
      "domains": ["test.neko233.com", "*.test.neko233.com"]
    }
  ]
}`

	if err := os.WriteFile(configPath, []byte(baseConfig), 0o644); err != nil {
		t.Fatalf("write base config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".local.json"), []byte(localOverride), 0o600); err != nil {
		t.Fatalf("write local override: %v", err)
	}

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
	if got := cfg.Certificates[0].Domains[0]; got != "test.neko233.com" {
		t.Fatalf("expected local certificate override, got %q", got)
	}
}
