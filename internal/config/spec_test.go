package config

import (
	"os"
	"path/filepath"
	"testing"

	"acme-go/internal/testspec"
)

type configSpec struct {
	Name          string            `json:"name"`
	Env           map[string]string `json:"env"`
	BaseConfig    string            `json:"base_config"`
	LocalOverride string            `json:"local_override"`
	Expect        struct {
		Email           string            `json:"email"`
		Provider        string            `json:"provider"`
		FirstDomain     string            `json:"first_domain"`
		RenewBeforeDays int               `json:"renew_before_days"`
		Challenge       string            `json:"challenge"`
		Bundle          bool              `json:"bundle"`
		Env             map[string]string `json:"env"`
	} `json:"expect"`
}

func TestLoadSpecs(t *testing.T) {
	var specs []configSpec
	if err := testspec.LoadJSON(filepath.Join("specs", "config_cases.json"), &specs); err != nil {
		t.Fatalf("load config specs: %v", err)
	}

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			for key, value := range spec.Env {
				t.Setenv(key, value)
			}

			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(spec.BaseConfig), 0o644); err != nil {
				t.Fatalf("write base config: %v", err)
			}
			if spec.LocalOverride != "" {
				if err := os.WriteFile(filepath.Join(dir, ".local.json"), []byte(spec.LocalOverride), 0o600); err != nil {
					t.Fatalf("write local override: %v", err)
				}
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}

			if cfg.Account.Email != spec.Expect.Email {
				t.Fatalf("email: got %q want %q", cfg.Account.Email, spec.Expect.Email)
			}
			if cfg.DNS.Provider != spec.Expect.Provider {
				t.Fatalf("provider: got %q want %q", cfg.DNS.Provider, spec.Expect.Provider)
			}
			if cfg.Certificates[0].Domains[0] != spec.Expect.FirstDomain {
				t.Fatalf("first domain: got %q want %q", cfg.Certificates[0].Domains[0], spec.Expect.FirstDomain)
			}
			if cfg.Certificates[0].RenewBeforeDays != spec.Expect.RenewBeforeDays {
				t.Fatalf("renew window: got %d want %d", cfg.Certificates[0].RenewBeforeDays, spec.Expect.RenewBeforeDays)
			}
			if cfg.Certificates[0].Challenge != spec.Expect.Challenge {
				t.Fatalf("challenge: got %q want %q", cfg.Certificates[0].Challenge, spec.Expect.Challenge)
			}
			if cfg.Certificates[0].Bundle == nil || *cfg.Certificates[0].Bundle != spec.Expect.Bundle {
				t.Fatalf("bundle: got %v want %v", cfg.Certificates[0].Bundle, spec.Expect.Bundle)
			}
			for key, value := range spec.Expect.Env {
				if cfg.DNS.Env[key] != value {
					t.Fatalf("env[%s]: got %q want %q", key, cfg.DNS.Env[key], value)
				}
			}
		})
	}
}
