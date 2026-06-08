package acme

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	lego "github.com/go-acme/lego/v4/lego"
	"github.com/neko233-com/acme233/internal/config"
	"github.com/neko233-com/acme233/internal/dnsprovider"
)

type CertificateInfo struct {
	Name        string                  `json:"name"`
	Domains     []string                `json:"domains"`
	OutputDir   string                  `json:"output_dir"`
	Paths       config.CertificatePaths `json:"paths"`
	Provider    string                  `json:"provider"`
	Status      string                  `json:"status"`
	NotAfter    *time.Time              `json:"not_after,omitempty"`
	RenewBefore int                     `json:"renew_before_days"`
	Metadata    *Metadata               `json:"metadata,omitempty"`
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

	user, err := LoadOrCreateUser(cfg.Account.Email, cfg.Account.KeyPath)
	if err != nil {
		return err
	}

	for _, cert := range targets {
		runtime, err := resolveCertificateRuntime(cfg, cert)
		if err != nil {
			return err
		}
		var client *lego.Client
		err = runtime.withChallengeEnv(func() error {
			var buildErr error
			client, buildErr = buildClient(user, cfg, runtime)
			return buildErr
		})
		if err != nil {
			return err
		}
		if err := ensureRegistration(client, user, cfg.Account.AcceptTOS); err != nil {
			return err
		}
		leaf, err := os.ReadFile(runtime.cert.Paths().CertFile)
		if err != nil {
			return fmt.Errorf("read certificate for revoke: %w", err)
		}
		if err := client.Certificate.Revoke(leaf); err != nil {
			return fmt.Errorf("revoke %s: %w", runtime.cert.Name, err)
		}
		if err := updateRevocationMetadata(runtime.cert); err != nil {
			return err
		}
		if err := runRevokeHooks(runtime, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "%s: revoked\n", runtime.cert.Name)
	}
	return nil
}

func Install(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}
	for _, cert := range targets {
		runtime, err := resolveCertificateRuntime(cfg, cert)
		if err != nil {
			return err
		}
		if err := validateLocalCertificateMaterial(cert); err != nil {
			return err
		}
		if err := installExisting(runtime, out); err != nil {
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
		runtime, err := resolveCertificateRuntime(cfg, cert)
		if err != nil {
			return err
		}
		if err := validateLocalCertificateMaterial(cert); err != nil {
			return err
		}
		if err := deployExisting(runtime, out); err != nil {
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

type ConfigSummary struct {
	CertificateCount int      `json:"certificate_count"`
	Certificates     []string `json:"certificates"`
}

type CertificatePathInfo struct {
	Name      string                  `json:"name"`
	OutputDir string                  `json:"output_dir"`
	Paths     config.CertificatePaths `json:"paths"`
}

func Validate(cfg *config.Config, out io.Writer) error {
	names := make([]string, 0, len(cfg.Certificates))
	for _, cert := range cfg.Certificates {
		names = append(names, cert.Name)
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(ConfigSummary{
		CertificateCount: len(cfg.Certificates),
		Certificates:     names,
	})
}

func Paths(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}
	infos := make([]CertificatePathInfo, 0, len(targets))
	for _, cert := range targets {
		infos = append(infos, CertificatePathInfo{
			Name:      cert.Name,
			OutputDir: cert.OutputDir,
			Paths:     cert.Paths(),
		})
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(infos)
}

func inspectCertificate(cfg *config.Config, cert config.CertificateSpec) (CertificateInfo, error) {
	runtime, err := resolveCertificateRuntime(cfg, cert)
	if err != nil {
		return CertificateInfo{}, err
	}
	info := CertificateInfo{
		Name:        runtime.cert.Name,
		Domains:     append([]string(nil), runtime.cert.Domains...),
		OutputDir:   runtime.cert.OutputDir,
		Paths:       runtime.cert.Paths(),
		Provider:    runtime.dns.Provider,
		Status:      "missing",
		RenewBefore: runtime.cert.RenewBeforeDays,
	}
	metadata, err := readMetadata(runtime.cert)
	if err == nil {
		info.Metadata = metadata
	}
	expiry, err := readCertificateExpiry(runtime.cert.Paths().FullChainFile)
	if err != nil {
		return info, nil
	}
	info.NotAfter = &expiry
	needsRenew, _, err := certificateNeedsRenew(runtime.cert)
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
