package acme

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"acme-go/internal/config"
	"acme-go/internal/dnsprovider"
)

type CertificateInfo struct {
	Name        string     `json:"name"`
	Domains     []string   `json:"domains"`
	OutputDir   string     `json:"output_dir"`
	Provider    string     `json:"provider"`
	Status      string     `json:"status"`
	NotAfter    *time.Time `json:"not_after,omitempty"`
	RenewBefore int        `json:"renew_before_days"`
	Metadata    *Metadata  `json:"metadata,omitempty"`
}

func List(cfg *config.Config, out io.Writer) error {
	infos := make([]CertificateInfo, 0, len(cfg.Certificates))
	for _, cert := range cfg.Certificates {
		info, err := inspectCertificate(cfg, cert)
		if err != nil {
			return err
		}
		infos = append(infos, info)
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(infos)
}

func Info(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}
	info, err := inspectCertificate(cfg, targets[0])
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

func Revoke(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}
	restoreEnv, err := applyEnv(cfg.DNS)
	if err != nil {
		return err
	}
	defer restoreEnv()

	user, err := LoadOrCreateUser(cfg.Account.Email, cfg.Account.KeyPath)
	if err != nil {
		return err
	}

	for _, cert := range targets {
		client, err := buildClient(user, cfg, cert)
		if err != nil {
			return err
		}
		if err := ensureRegistration(client, user, cfg.Account.AcceptTOS); err != nil {
			return err
		}
		leaf, err := os.ReadFile(filepath.Join(cert.OutputDir, "cert.pem"))
		if err != nil {
			return fmt.Errorf("read certificate for revoke: %w", err)
		}
		if err := client.Certificate.Revoke(leaf); err != nil {
			return fmt.Errorf("revoke %s: %w", cert.Name, err)
		}
		if err := updateRevocationMetadata(cert); err != nil {
			return err
		}
		if err := runRevokeHooks(cfg, cert, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s: revoked\n", cert.Name)
	}
	return nil
}

func Install(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}
	for _, cert := range targets {
		if err := validateLocalCertificateMaterial(cert); err != nil {
			return err
		}
		if err := installExisting(cfg, cert, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s: installed\n", cert.Name)
	}
	return nil
}

func Deploy(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}
	for _, cert := range targets {
		if err := validateLocalCertificateMaterial(cert); err != nil {
			return err
		}
		if err := deployExisting(cfg, cert, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s: deployed\n", cert.Name)
	}
	return nil
}

func Providers(out io.Writer) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(dnsprovider.Supported())
}

func inspectCertificate(cfg *config.Config, cert config.CertificateSpec) (CertificateInfo, error) {
	info := CertificateInfo{
		Name:        cert.Name,
		Domains:     append([]string(nil), cert.Domains...),
		OutputDir:   cert.OutputDir,
		Provider:    cfg.DNS.Provider,
		Status:      "missing",
		RenewBefore: cert.RenewBeforeDays,
	}
	metadata, err := readMetadata(cert)
	if err == nil {
		info.Metadata = metadata
	}
	expiry, err := readCertificateExpiry(filepath.Join(cert.OutputDir, "fullchain.pem"))
	if err != nil {
		return info, nil
	}
	info.NotAfter = &expiry
	needsRenew, _, err := certificateNeedsRenew(cert)
	if err != nil {
		return info, nil
	}
	if metadata != nil && metadata.RevokedAt != nil {
		info.Status = "revoked"
		return info, nil
	}
	if needsRenew {
		info.Status = "renew-due"
		return info, nil
	}
	info.Status = "valid"
	return info, nil
}
