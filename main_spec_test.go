package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"acme-go/internal/testspec"
)

type commandSpec struct {
	Name              string   `json:"name"`
	Args              []string `json:"args"`
	RequireConfig     bool     `json:"require_config"`
	WantErrorContains string   `json:"want_error_contains"`
}

func TestCommandDispatchSpecs(t *testing.T) {
	var specs []commandSpec
	if err := testspec.LoadJSON(filepath.Join("specs", "command_dispatch.json"), &specs); err != nil {
		t.Fatalf("load command specs: %v", err)
	}

	for _, spec := range specs {
		t.Run(spec.Name, func(t *testing.T) {
			args := append([]string(nil), spec.Args...)
			if spec.RequireConfig {
				configPath := writeCommandConfig(t)
				hasConfig := false
				for _, arg := range args {
					if arg == "-config" {
						hasConfig = true
						break
					}
				}
				if !hasConfig {
					args = append(args, "-config", configPath)
				}
			}

			err := run(args)
			if spec.WantErrorContains == "" {
				if err != nil {
					t.Fatalf("run(%v): %v", args, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("run(%v): expected error containing %q", args, spec.WantErrorContains)
			}
			if !strings.Contains(err.Error(), spec.WantErrorContains) {
				t.Fatalf("run(%v): got error %q, want substring %q", args, err.Error(), spec.WantErrorContains)
			}
		})
	}
}

func writeCommandConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := "account:\n  email: ops@example.com\n  accept_tos: true\ndns:\n  provider: cloudflare\ncertificates:\n  - name: example\n    domains:\n      - example.com\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}
