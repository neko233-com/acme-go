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
	"testing"
	"time"

	"acme-go/internal/config"
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
