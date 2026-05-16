package acme

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"acme-go/internal/config"
)

func TestRunSuccessActionsRunsInstallDeployAndHooks(t *testing.T) {
	outputDir := t.TempDir()
	writeCertificateFixtures(t, outputDir)

	installHook := envPrintCommand("INSTALL:", "ACME_CERT_NAME")
	deployHook := envPrintCommand("DEPLOY:", "ACME_CERT_NAME")
	commandHook := envPrintCommand("TARGET:", "ACME_DEPLOY_TARGET")
	var out bytes.Buffer

	cfg := &config.Config{
		CA:  config.CAConfig{DirectoryURL: "https://example.com/directory"},
		DNS: config.DNSConfig{Provider: "cloudflare"},
	}
	cert := config.CertificateSpec{
		Name:      "example",
		Domains:   []string{"example.com"},
		OutputDir: outputDir,
		Challenge: "webroot",
		Install: config.InstallConfig{
			KeyFile:       filepath.Join(outputDir, "installed", "privkey.pem"),
			FullChainFile: filepath.Join(outputDir, "installed", "fullchain.pem"),
		},
		Deploy: []config.DeployTarget{
			{
				Name:      "copy-edge",
				Type:      "copy",
				Directory: filepath.Join(outputDir, "deploy-copy"),
			},
			{
				Name:    "command-edge",
				Type:    "command",
				Command: commandHook,
			},
		},
		Hooks: config.HookConfig{
			PostInstall: []string{installHook},
			PostDeploy:  []string{deployHook},
		},
	}

	if err := runSuccessActions(cfg, cert, ModeIssue, &out); err != nil {
		t.Fatalf("runSuccessActions: %v", err)
	}

	assertFileContains(t, filepath.Join(outputDir, "installed", "privkey.pem"), "PRIVATE KEY")
	assertFileContains(t, filepath.Join(outputDir, "installed", "fullchain.pem"), "BEGIN CERTIFICATE")
	assertFileContains(t, filepath.Join(outputDir, "deploy-copy", "fullchain.pem"), "BEGIN CERTIFICATE")
	if !strings.Contains(out.String(), "INSTALL:") || !strings.Contains(out.String(), "example") {
		t.Fatalf("post-install output missing marker: %q", out.String())
	}
	if !strings.Contains(out.String(), "DEPLOY:") || !strings.Contains(out.String(), "example") {
		t.Fatalf("post-deploy output missing marker: %q", out.String())
	}
	if !strings.Contains(out.String(), "TARGET:") || !strings.Contains(out.String(), "command-edge") {
		t.Fatalf("deploy target output missing marker: %q", out.String())
	}
}

func TestRunPreHooksUsesRenewCommands(t *testing.T) {
	outputDir := t.TempDir()
	hookCommand := envPrintCommand("PRERENEW:", "ACME_CERT_NAME")
	var out bytes.Buffer

	cert := config.CertificateSpec{
		Name:      "renew-example",
		Domains:   []string{"example.com"},
		OutputDir: outputDir,
		Hooks: config.HookConfig{
			PreRenew: []string{hookCommand},
		},
	}

	if err := runPreHooks(cert, ModeRenew, newDeployContext(&config.Config{}, cert), &out); err != nil {
		t.Fatalf("runPreHooks: %v", err)
	}
	if !strings.Contains(out.String(), "PRERENEW:") || !strings.Contains(out.String(), "renew-example") {
		t.Fatalf("pre-renew output missing marker: %q", out.String())
	}
}

func writeCertificateFixtures(t *testing.T, outputDir string) {
	t.Helper()
	fixtures := map[string]string{
		"cert.pem":      "-----BEGIN CERTIFICATE-----\nleaf\n-----END CERTIFICATE-----\n",
		"fullchain.pem": "-----BEGIN CERTIFICATE-----\nfullchain\n-----END CERTIFICATE-----\n",
		"issuer.pem":    "-----BEGIN CERTIFICATE-----\nissuer\n-----END CERTIFICATE-----\n",
		"privkey.pem":   "-----BEGIN PRIVATE KEY-----\nprivate\n-----END PRIVATE KEY-----\n",
	}
	for name, content := range fixtures {
		if err := os.WriteFile(filepath.Join(outputDir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write fixture %s: %v", name, err)
		}
	}
}

func envPrintCommand(prefix, envName string) string {
	if runtime.GOOS == "windows" {
		return `powershell -NoProfile -Command "Write-Output ('` + prefix + `' + $env:` + envName + `)"`
	}
	return `printf '` + prefix + `%s' "$` + envName + `"`
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("file %s does not contain %q; got %q", path, want, string(data))
	}
}
