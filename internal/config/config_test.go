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
