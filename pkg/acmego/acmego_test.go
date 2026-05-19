package acmego_test

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/neko233-com/acme-go/pkg/acmego"
)

func TestPublicAPIExposesConfigAndHelpers(t *testing.T) {
	dir := t.TempDir()
	cfg := &acmego.Config{
		CA: acmego.CAConfig{DirectoryURL: "https://example.com/directory"},
		Account: acmego.AccountConfig{
			Email:     "ops@example.com",
			AcceptTOS: true,
		},
		DNS: acmego.DNSConfig{Provider: "cloudflare"},
		Certificates: []acmego.CertificateSpec{
			{
				Name:      "example",
				Domains:   []string{"example.com"},
				OutputDir: dir,
				OutputFiles: acmego.CertificateFiles{
					FullChainFile: "nginx.crt",
					KeyFile:       "nginx.key",
					PublicKeyFile: "nginx.pub",
					MetadataFile:  "nginx.meta.json",
				},
				RenewBeforeDays: 30,
				Challenge:       "dns-01",
			},
		},
	}

	paths := cfg.Certificates[0].Paths()
	if paths.FullChainFile != filepath.Join(dir, "nginx.crt") {
		t.Fatalf("fullchain path: got %q", paths.FullChainFile)
	}
	if paths.PublicKeyFile != filepath.Join(dir, "nginx.pub") {
		t.Fatalf("public key path: got %q", paths.PublicKeyFile)
	}
	if _, err := acmego.Run(cfg, acmego.Options{Name: "missing", Mode: acmego.ModeRenew, Out: io.Discard}); err == nil {
		t.Fatal("expected missing certificate error")
	}
}
