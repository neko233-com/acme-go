package acme

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"acme-go/internal/config"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	lego "github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/dns"
	"github.com/go-acme/lego/v4/registration"
)

type Mode string

const (
	ModeIssue Mode = "issue"
	ModeRenew Mode = "renew"
)

type Options struct {
	Name  string
	Force bool
	Mode  Mode
	Out   io.Writer
}

type Result struct {
	Changed int
	Skipped int
}

func Plan(cfg *config.Config, name string, out io.Writer) error {
	targets, err := selectCertificates(cfg.Certificates, name)
	if err != nil {
		return err
	}

	for _, cert := range targets {
		needsRenew, expiry, err := certificateNeedsRenew(cert)
		if err != nil {
			fmt.Fprintf(out, "%s: issue (%v)\n", cert.Name, err)
			continue
		}
		if needsRenew {
			fmt.Fprintf(out, "%s: issue (expires %s)\n", cert.Name, expiry.Format(time.RFC3339))
			continue
		}
		fmt.Fprintf(out, "%s: skip (expires %s)\n", cert.Name, expiry.Format(time.RFC3339))
	}
	return nil
}

func Run(cfg *config.Config, options Options) (Result, error) {
	if options.Out == nil {
		options.Out = io.Discard
	}

	targets, err := selectCertificates(cfg.Certificates, options.Name)
	if err != nil {
		return Result{}, err
	}

	restoreEnv, err := applyEnv(cfg.DNS.Env)
	if err != nil {
		return Result{}, err
	}
	defer restoreEnv()

	user, err := LoadOrCreateUser(cfg.Account.Email, cfg.Account.KeyPath)
	if err != nil {
		return Result{}, err
	}

	result := Result{}
	for _, cert := range targets {
		needsRenew, expiry, err := certificateNeedsRenew(cert)
		if err == nil && !needsRenew && !options.Force {
			fmt.Fprintf(options.Out, "%s: skip, certificate valid until %s\n", cert.Name, expiry.Format(time.RFC3339))
			result.Skipped++
			continue
		}

		client, err := buildClient(user, cfg, cert)
		if err != nil {
			return result, err
		}
		if err := ensureRegistration(client, user, cfg.Account.AcceptTOS); err != nil {
			return result, err
		}

		request := certificate.ObtainRequest{
			Domains: cert.Domains,
			Bundle:  cert.Bundle,
		}
		resource, err := client.Certificate.Obtain(request)
		if err != nil {
			return result, fmt.Errorf("issue %s: %w", cert.Name, err)
		}
		if err := writeCertificate(cert, cfg.CA.DirectoryURL, resource); err != nil {
			return result, err
		}
		fmt.Fprintf(options.Out, "%s: wrote certificate to %s\n", cert.Name, cert.OutputDir)
		result.Changed++
	}

	return result, nil
}

func selectCertificates(certificates []config.CertificateSpec, name string) ([]config.CertificateSpec, error) {
	if name == "" {
		copyOf := append([]config.CertificateSpec(nil), certificates...)
		sort.Slice(copyOf, func(i, j int) bool { return copyOf[i].Name < copyOf[j].Name })
		return copyOf, nil
	}
	for _, cert := range certificates {
		if cert.Name == name {
			return []config.CertificateSpec{cert}, nil
		}
	}
	return nil, fmt.Errorf("certificate %q not found", name)
}

func applyEnv(values map[string]string) (func(), error) {
	type restore struct {
		value   string
		present bool
	}
	originals := map[string]restore{}
	for key, value := range values {
		current, present := os.LookupEnv(key)
		originals[key] = restore{value: current, present: present}
		if err := os.Setenv(key, value); err != nil {
			return nil, fmt.Errorf("set env %s: %w", key, err)
		}
	}
	return func() {
		for key, item := range originals {
			if item.present {
				_ = os.Setenv(key, item.value)
				continue
			}
			_ = os.Unsetenv(key)
		}
	}, nil
}

func buildClient(user *User, cfg *config.Config, cert config.CertificateSpec) (*lego.Client, error) {
	legoConfig := lego.NewConfig(user)
	legoConfig.CADirURL = cfg.CA.DirectoryURL
	legoConfig.Certificate.KeyType = mapKeyType(cert.KeyType)

	client, err := lego.NewClient(legoConfig)
	if err != nil {
		return nil, fmt.Errorf("create ACME client: %w", err)
	}

	provider, err := dns.NewDNSChallengeProviderByName(strings.ToUpper(cfg.DNS.Provider))
	if err != nil {
		provider, err = dns.NewDNSChallengeProviderByName(cfg.DNS.Provider)
		if err != nil {
			return nil, fmt.Errorf("build DNS provider %q: %w", cfg.DNS.Provider, err)
		}
	}
	if err := client.Challenge.SetDNS01Provider(provider); err != nil {
		return nil, fmt.Errorf("attach DNS provider: %w", err)
	}
	return client, nil
}

func ensureRegistration(client *lego.Client, user *User, acceptTOS bool) error {
	if user.Registration != nil {
		return nil
	}

	registrationResource, err := client.Registration.ResolveAccountByKey()
	if err != nil {
		registrationResource, err = client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: acceptTOS})
		if err != nil {
			return fmt.Errorf("register account: %w", err)
		}
	}
	user.Registration = registrationResource
	if err := user.Save(); err != nil {
		return err
	}
	return nil
}

func mapKeyType(value string) certcrypto.KeyType {
	switch strings.ToLower(value) {
	case "ec384":
		return certcrypto.EC384
	case "rsa2048":
		return certcrypto.RSA2048
	case "rsa4096":
		return certcrypto.RSA4096
	case "rsa8192":
		return certcrypto.RSA8192
	default:
		return certcrypto.EC256
	}
}

func certificateNeedsRenew(cert config.CertificateSpec) (bool, time.Time, error) {
	expiry, err := readCertificateExpiry(filepath.Join(cert.OutputDir, "fullchain.pem"))
	if err != nil {
		return true, time.Time{}, err
	}
	renewAt := time.Now().Add(time.Duration(cert.RenewBeforeDays) * 24 * time.Hour)
	return !expiry.After(renewAt), expiry, nil
}

func readCertificateExpiry(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return time.Time{}, fmt.Errorf("no certificate found in %s", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse certificate %s: %w", path, err)
	}
	return cert.NotAfter, nil
}

func writeCertificate(cert config.CertificateSpec, directoryURL string, resource *certificate.Resource) error {
	if err := os.MkdirAll(cert.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create certificate dir: %w", err)
	}

	leaf, chain, err := splitCertificate(resource.Certificate)
	if err != nil {
		return err
	}
	if len(chain) == 0 && len(resource.IssuerCertificate) > 0 {
		chain = resource.IssuerCertificate
	}
	fullChain := append([]byte{}, leaf...)
	fullChain = append(fullChain, chain...)

	files := map[string][]byte{
		"cert.pem":      leaf,
		"issuer.pem":    chain,
		"fullchain.pem": fullChain,
		"privkey.pem":   resource.PrivateKey,
	}
	for name, content := range files {
		if len(content) == 0 {
			continue
		}
		if err := os.WriteFile(filepath.Join(cert.OutputDir, name), content, 0o600); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	expiry, err := readCertificateExpiry(filepath.Join(cert.OutputDir, "fullchain.pem"))
	if err != nil {
		return err
	}

	metadata, err := json.MarshalIndent(struct {
		Name        string    `json:"name"`
		Domains     []string  `json:"domains"`
		Directory   string    `json:"directory_url"`
		IssuedAt    time.Time `json:"issued_at"`
		NotAfter    time.Time `json:"not_after"`
		RenewBefore int       `json:"renew_before_days"`
	}{
		Name:        cert.Name,
		Domains:     cert.Domains,
		Directory:   directoryURL,
		IssuedAt:    time.Now().UTC(),
		NotAfter:    expiry.UTC(),
		RenewBefore: cert.RenewBeforeDays,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(cert.OutputDir, "metadata.json"), metadata, 0o600); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	return nil
}

func splitCertificate(data []byte) ([]byte, []byte, error) {
	var leaf bytes.Buffer
	var chain bytes.Buffer
	index := 0
	for len(data) > 0 {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			if index == 0 {
				if err := pem.Encode(&leaf, block); err != nil {
					return nil, nil, fmt.Errorf("encode leaf certificate: %w", err)
				}
			} else {
				if err := pem.Encode(&chain, block); err != nil {
					return nil, nil, fmt.Errorf("encode chain certificate: %w", err)
				}
			}
			index++
		}
		data = rest
	}
	if leaf.Len() == 0 {
		return nil, nil, fmt.Errorf("ACME response did not contain a leaf certificate")
	}
	return leaf.Bytes(), chain.Bytes(), nil
}
