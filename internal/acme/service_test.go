package acme

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/neko233-com/acme233/internal/config"
)

func TestCertificateNeedsRenew(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "cert")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	certPEM := createCertificatePEM(t, time.Now().Add(20*24*time.Hour))
	if err := os.WriteFile(filepath.Join(outputDir, "fullchain.pem"), certPEM, 0o600); err != nil {
		t.Fatalf("write certificate: %v", err)
	}

	needsRenew, _, err := certificateNeedsRenew(config.CertificateSpec{
		Name:            "example",
		OutputDir:       outputDir,
		RenewBeforeDays: 30,
	})
	if err != nil {
		t.Fatalf("certificateNeedsRenew: %v", err)
	}
	if !needsRenew {
		t.Fatal("expected certificate to require renewal")
	}
}

func TestWriteCertificateUsesConfiguredOutputFiles(t *testing.T) {
	dir := t.TempDir()
	cert := config.CertificateSpec{
		Name:      "nginx-gui",
		Domains:   []string{"example.com"},
		OutputDir: dir,
		OutputFiles: config.CertificateFiles{
			CertFile:      "server.crt",
			KeyFile:       "private/server.key",
			PublicKeyFile: "server.pub",
			FullChainFile: "server.fullchain.crt",
			ChainFile:     "server.chain.crt",
			MetadataFile:  "server.meta.json",
		},
	}
	cfg := &config.Config{
		CA:  config.CAConfig{DirectoryURL: "https://example.com/directory"},
		DNS: config.DNSConfig{Provider: "cloudflare"},
	}
	resource := &certificate.Resource{
		Domain:            "example.com",
		Certificate:       createCertificatePEM(t, time.Now().Add(90*24*time.Hour)),
		IssuerCertificate: createCertificatePEM(t, time.Now().Add(365*24*time.Hour)),
		PrivateKey:        []byte("-----BEGIN PRIVATE KEY-----\nprivate\n-----END PRIVATE KEY-----\n"),
	}

	runtime := certificateRuntime{
		cert: cert,
		dns:  config.DNSConfig{Provider: "cloudflare"},
	}
	if err := writeCertificate(runtime, cfg, resource); err != nil {
		t.Fatalf("writeCertificate: %v", err)
	}

	paths := cert.Paths()
	assertPathContains(t, paths.CertFile, "BEGIN CERTIFICATE")
	assertPathContains(t, paths.KeyFile, "PRIVATE KEY")
	assertPathContains(t, paths.PublicKeyFile, "PUBLIC KEY")
	assertPathContains(t, paths.FullChainFile, "BEGIN CERTIFICATE")
	assertPathContains(t, paths.ChainFile, "BEGIN CERTIFICATE")
	assertPathContains(t, paths.MetadataFile, "nginx-gui")
	assertPathContains(t, filepath.Join(dir, "README.en.md"), "server.fullchain.crt")
	assertPathContains(t, filepath.Join(dir, "README.zh-CN.md"), "server.key")
	if _, err := os.Stat(filepath.Join(dir, "fullchain.pem")); !os.IsNotExist(err) {
		t.Fatalf("default fullchain.pem should not be written when custom output file is configured")
	}
}

func createCertificatePEM(t *testing.T, notAfter time.Time) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		DNSNames:     []string{"example.com"},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func assertPathContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("file %s does not contain %q", path, want)
	}
}
