package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/acme-go/internal/testspec"
)

type configSpec struct {
	Name          string            `json:"name"`
	Env           map[string]string `json:"env"`
	BaseConfig    string            `json:"base_config"`
	LocalOverride string            `json:"local_override"`
	Expect        struct {
		Email                string            `json:"email"`
		Provider             string            `json:"provider"`
		FirstDomain          string            `json:"first_domain"`
		RenewBeforeDays      int               `json:"renew_before_days"`
		Challenge            string            `json:"challenge"`
		Bundle               bool              `json:"bundle"`
		WebrootPath          string            `json:"webroot_path"`
		InstallFullChainFile string            `json:"install_fullchain_file"`
		DeployCount          int               `json:"deploy_count"`
		PostInstallHooks     int               `json:"post_install_hooks"`
		PostDeployHooks      int               `json:"post_deploy_hooks"`
		Env                  map[string]string `json:"env"`
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
			configPath := filepath.Join(dir, "config_acme.json")
			if err := os.WriteFile(configPath, []byte(spec.BaseConfig), 0o644); err != nil {
				t.Fatalf("write base config: %v", err)
			}
			if spec.LocalOverride != "" {
				if err := os.WriteFile(filepath.Join(dir, "config_acme.local.json"), []byte(spec.LocalOverride), 0o600); err != nil {
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
			effectiveDNS, err := cfg.EffectiveDNS(cfg.Certificates[0])
			if err != nil {
				t.Fatalf("effective dns: %v", err)
			}
			if effectiveDNS.Provider != spec.Expect.Provider {
				t.Fatalf("provider: got %q want %q", effectiveDNS.Provider, spec.Expect.Provider)
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
			if spec.Expect.WebrootPath != "" && cfg.Certificates[0].WebrootPath != spec.Expect.WebrootPath {
				t.Fatalf("webroot_path: got %q want %q", cfg.Certificates[0].WebrootPath, spec.Expect.WebrootPath)
			}
			if spec.Expect.InstallFullChainFile != "" && cfg.Certificates[0].Install.FullChainFile != spec.Expect.InstallFullChainFile {
				t.Fatalf("install.fullchain_file: got %q want %q", cfg.Certificates[0].Install.FullChainFile, spec.Expect.InstallFullChainFile)
			}
			if spec.Expect.DeployCount != 0 && len(cfg.Certificates[0].Deploy) != spec.Expect.DeployCount {
				t.Fatalf("deploy count: got %d want %d", len(cfg.Certificates[0].Deploy), spec.Expect.DeployCount)
			}
			if spec.Expect.PostInstallHooks != 0 && len(cfg.Certificates[0].Hooks.PostInstall) != spec.Expect.PostInstallHooks {
				t.Fatalf("post_install hook count: got %d want %d", len(cfg.Certificates[0].Hooks.PostInstall), spec.Expect.PostInstallHooks)
			}
			if spec.Expect.PostDeployHooks != 0 && len(cfg.Certificates[0].Hooks.PostDeploy) != spec.Expect.PostDeployHooks {
				t.Fatalf("post_deploy hook count: got %d want %d", len(cfg.Certificates[0].Hooks.PostDeploy), spec.Expect.PostDeployHooks)
			}
			for key, value := range spec.Expect.Env {
				if effectiveDNS.Env[key] != value {
					t.Fatalf("env[%s]: got %q want %q", key, effectiveDNS.Env[key], value)
				}
			}
		})
	}
}
