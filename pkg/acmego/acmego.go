package acmego

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/neko233-com/acme-go/internal/acme"
	"github.com/neko233-com/acme-go/internal/config"
	"github.com/neko233-com/acme-go/internal/doc"
	"github.com/neko233-com/acme-go/internal/update"
)

type Config = config.Config
type CAConfig = config.CAConfig
type AccountConfig = config.AccountConfig
type AutomationConfig = config.AutomationConfig
type DNSConfig = config.DNSConfig
type CertificateSpec = config.CertificateSpec
type CertificateFiles = config.CertificateFiles
type CertificatePaths = config.CertificatePaths
type DNSCredentials = config.DNSCredentials
type InstallConfig = config.InstallConfig
type DeployTarget = config.DeployTarget
type HookConfig = config.HookConfig

type Mode = acme.Mode

const (
	ModeIssue Mode = acme.ModeIssue
	ModeRenew Mode = acme.ModeRenew
)

type Options = acme.Options
type Result = acme.Result
type AutoRenewOptions = acme.AutoRenewOptions
type Metadata = acme.Metadata
type ConfigSummary = acme.ConfigSummary
type CertificatePathInfo = acme.CertificatePathInfo
type ReleaseDiff = update.ReleaseDiff
type VersionInfo = update.VersionInfo

// Request is the compact library-facing API for embedding ACME issuance in a Go
// service. Fill Email, Provider, provider credentials, and Domains; the helper
// builds the full runtime config for DNS-01 certificate issuance.
type Request struct {
	Email                      string
	AccountKeyPath             string
	CADirectoryURL             string
	Name                       string
	Domains                    []string
	Provider                   string
	AccountMode                string
	Region                     string
	Credentials                DNSCredentials
	Env                        map[string]string
	OutputDir                  string
	OutputFiles                CertificateFiles
	KeyType                    string
	RenewBeforeDays            int
	DisableCNAMESupport        bool
	DisableCompletePropagation bool
	RecursiveNameservers       []string
	Force                      bool
}

type CertificateResult struct {
	Result Result
	Name   string
	Paths  CertificatePaths
}

func Load(path string) (*Config, error) {
	return config.Load(path)
}

func BuildConfig(request Request) (*Config, error) {
	if len(request.Domains) == 0 {
		return nil, fmt.Errorf("at least one domain is required")
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = certificateNameFromDomain(request.Domains[0])
	}
	outputDir := strings.TrimSpace(request.OutputDir)
	if outputDir == "" {
		outputDir = filepath.Join("certs", name)
	}
	keyType := strings.TrimSpace(request.KeyType)
	if keyType == "" {
		keyType = "ec256"
	}
	renewBeforeDays := request.RenewBeforeDays
	if renewBeforeDays == 0 {
		renewBeforeDays = 30
	}
	caDirectoryURL := strings.TrimSpace(request.CADirectoryURL)
	if caDirectoryURL == "" {
		caDirectoryURL = "https://acme-v02.api.letsencrypt.org/directory"
	}
	accountKeyPath := strings.TrimSpace(request.AccountKeyPath)
	if accountKeyPath == "" {
		accountKeyPath = filepath.Join(outputDir, ".acme", "account.pem")
	}
	maxRetryAttempts := 5

	cfg := &Config{
		CA: CAConfig{DirectoryURL: caDirectoryURL},
		Account: AccountConfig{
			Email:     request.Email,
			KeyPath:   accountKeyPath,
			AcceptTOS: true,
		},
		Automation: AutomationConfig{
			RenewInterval:    "24h",
			RetryBackoff:     "1m",
			MaxRetryBackoff:  "15m",
			MaxRetryAttempts: &maxRetryAttempts,
		},
		DNS: DNSConfig{
			Provider:                   request.Provider,
			AccountMode:                request.AccountMode,
			Region:                     request.Region,
			Credentials:                request.Credentials,
			Env:                        copyStringMap(request.Env),
			DisableCNAMESupport:        request.DisableCNAMESupport,
			DisableCompletePropagation: request.DisableCompletePropagation,
			RecursiveNameservers:       append([]string(nil), request.RecursiveNameservers...),
		},
		Certificates: []CertificateSpec{
			{
				Name:            name,
				Domains:         append([]string(nil), request.Domains...),
				OutputDir:       outputDir,
				OutputFiles:     request.OutputFiles,
				KeyType:         keyType,
				RenewBeforeDays: renewBeforeDays,
				Challenge:       "dns-01",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func IssueCertificate(request Request, out io.Writer) (CertificateResult, error) {
	cfg, err := BuildConfig(request)
	if err != nil {
		return CertificateResult{}, err
	}
	cert := cfg.Certificates[0]
	result, err := Issue(cfg, cert.Name, request.Force, out)
	if err != nil {
		return CertificateResult{}, err
	}
	return CertificateResult{Result: result, Name: cert.Name, Paths: cert.Paths()}, nil
}

func RenewCertificate(request Request, out io.Writer) (CertificateResult, error) {
	cfg, err := BuildConfig(request)
	if err != nil {
		return CertificateResult{}, err
	}
	cert := cfg.Certificates[0]
	result, err := Renew(cfg, cert.Name, request.Force, out)
	if err != nil {
		return CertificateResult{}, err
	}
	return CertificateResult{Result: result, Name: cert.Name, Paths: cert.Paths()}, nil
}

func certificateNameFromDomain(domain string) string {
	name := strings.TrimSpace(strings.TrimPrefix(domain, "*."))
	name = strings.NewReplacer("*", "_", ":", "_", "/", "_", "\\", "_", " ", "_").Replace(name)
	if name == "" {
		return "certificate"
	}
	return name
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	copyOf := make(map[string]string, len(values))
	for key, value := range values {
		copyOf[key] = value
	}
	return copyOf
}

func Plan(cfg *Config, name string, out io.Writer) error {
	return acme.Plan(cfg, name, out)
}

func Validate(cfg *Config, out io.Writer) error {
	return acme.Validate(cfg, out)
}

func Paths(cfg *Config, name string, out io.Writer) error {
	return acme.Paths(cfg, name, out)
}

func Run(cfg *Config, options Options) (Result, error) {
	return acme.Run(cfg, options)
}

func Issue(cfg *Config, name string, force bool, out io.Writer) (Result, error) {
	return acme.Run(cfg, Options{Name: name, Force: force, Mode: ModeIssue, Out: out})
}

func Renew(cfg *Config, name string, force bool, out io.Writer) (Result, error) {
	return acme.Run(cfg, Options{Name: name, Force: force, Mode: ModeRenew, Out: out})
}

func AutoRenewLoop(ctx context.Context, cfg *Config, options AutoRenewOptions) error {
	return acme.AutoRenewLoop(ctx, cfg, options)
}

func List(cfg *Config, out io.Writer) error {
	return acme.List(cfg, out)
}

func Info(cfg *Config, name string, out io.Writer) error {
	return acme.Info(cfg, name, out)
}

func Revoke(cfg *Config, name string, out io.Writer) error {
	return acme.Revoke(cfg, name, out)
}

func Install(cfg *Config, name string, out io.Writer) error {
	return acme.Install(cfg, name, out)
}

func Deploy(cfg *Config, name string, out io.Writer) error {
	return acme.Deploy(cfg, name, out)
}

func Providers(out io.Writer) error {
	return acme.Providers(out)
}

func CheckVersion(currentVersion string) (VersionInfo, error) {
	return update.CheckVersion(currentVersion)
}

func Upgrade(currentVersion, executablePath string, out io.Writer) error {
	return update.Upgrade(currentVersion, executablePath, out)
}

func MaybeAutoUpdate(currentVersion, executablePath string, out io.Writer) error {
	return update.MaybeAutoUpdate(currentVersion, executablePath, out)
}

func ResolveGuidePath(workingDir, executablePath string) (string, error) {
	return doc.ResolveGuidePath(workingDir, executablePath)
}

func OpenGuide(workingDir, executablePath string) (string, error) {
	return doc.OpenGuide(workingDir, executablePath)
}
