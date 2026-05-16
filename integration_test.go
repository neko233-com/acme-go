//go:build integration

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"acme-go/internal/acme"
	"acme-go/internal/config"
)

func TestAliDNSStagingIssueFlow(t *testing.T) {
	if _, err := os.Stat(".local.json"); err != nil {
		if os.IsNotExist(err) {
			t.Skip(".local.json not found")
		}
		t.Fatalf("stat .local.json: %v", err)
	}

	cfg, err := config.Load("config.integration.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	tempDir := t.TempDir()
	cfg.Account.KeyPath = filepath.Join(tempDir, "account.pem")
	for index := range cfg.Certificates {
		cfg.Certificates[index].OutputDir = filepath.Join(tempDir, cfg.Certificates[index].Name)
	}

	var issueOutput bytes.Buffer
	issueResult, err := acme.Run(cfg, acme.Options{
		Name:  "neko233-staging",
		Force: true,
		Mode:  acme.ModeIssue,
		Out:   &issueOutput,
	})
	if err != nil {
		t.Fatalf("issue certificate: %v\n%s", err, issueOutput.String())
	}
	if issueResult.Changed != 1 {
		t.Fatalf("expected one certificate to be issued, got %+v\n%s", issueResult, issueOutput.String())
	}
	if _, err := os.Stat(filepath.Join(tempDir, "neko233-staging", "fullchain.pem")); err != nil {
		t.Fatalf("expected fullchain.pem: %v", err)
	}

	var renewOutput bytes.Buffer
	renewResult, err := acme.Run(cfg, acme.Options{
		Name: "neko233-staging",
		Mode: acme.ModeRenew,
		Out:  &renewOutput,
	})
	if err != nil {
		t.Fatalf("renew certificate: %v\n%s", err, renewOutput.String())
	}
	if renewResult.Skipped != 1 {
		t.Fatalf("expected renew to skip a fresh certificate, got %+v\n%s", renewResult, renewOutput.String())
	}
}