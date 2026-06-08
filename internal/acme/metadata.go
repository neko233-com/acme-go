package acme

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/neko233-com/acme233/internal/config"

	"github.com/go-acme/lego/v4/certificate"
)

type Metadata struct {
	Name           string     `json:"name"`
	Domains        []string   `json:"domains"`
	DirectoryURL   string     `json:"directory_url"`
	Provider       string     `json:"provider"`
	Challenge      string     `json:"challenge"`
	IssuedAt       time.Time  `json:"issued_at"`
	NotAfter       time.Time  `json:"not_after"`
	RenewBefore    int        `json:"renew_before_days"`
	PreferredChain string     `json:"preferred_chain,omitempty"`
	MustStaple     bool       `json:"must_staple,omitempty"`
	CSRPath        string     `json:"csr_path,omitempty"`
	CertURL        string     `json:"cert_url,omitempty"`
	CertStableURL  string     `json:"cert_stable_url,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

// Metadata intentionally stores the resolved provider rather than provider_ref
// so historical records remain readable even when named provider blocks change.
func writeMetadata(runtime certificateRuntime, cfg *config.Config, resource *certificate.Resource, expiry time.Time) error {
	metadata, err := json.MarshalIndent(Metadata{
		Name:           runtime.cert.Name,
		Domains:        runtime.cert.Domains,
		DirectoryURL:   cfg.CA.DirectoryURL,
		Provider:       runtime.dns.Provider,
		Challenge:      runtime.cert.Challenge,
		IssuedAt:       time.Now().UTC(),
		NotAfter:       expiry.UTC(),
		RenewBefore:    runtime.cert.RenewBeforeDays,
		PreferredChain: runtime.cert.PreferredChain,
		MustStaple:     runtime.cert.MustStaple,
		CSRPath:        runtime.cert.CSRPath,
		CertURL:        resource.CertURL,
		CertStableURL:  resource.CertStableURL,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath(runtime.cert), metadata, 0o600); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func readMetadata(cert config.CertificateSpec) (*Metadata, error) {
	data, err := os.ReadFile(metadataPath(cert))
	if err != nil {
		return nil, err
	}
	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	return &metadata, nil
}

func updateRevocationMetadata(cert config.CertificateSpec) error {
	metadata, err := readMetadata(cert)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	now := time.Now().UTC()
	metadata.RevokedAt = &now
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath(cert), data, 0o600); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func metadataPath(cert config.CertificateSpec) string {
	return cert.Paths().MetadataFile
}
