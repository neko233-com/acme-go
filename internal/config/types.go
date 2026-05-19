package config

import "path/filepath"

const defaultCADirectoryURL = "https://acme-v02.api.letsencrypt.org/directory"
const defaultAutomationRenewInterval = "24h"
const defaultAutomationRetryBackoff = "1m"
const defaultAutomationMaxRetryBackoff = "15m"
const defaultAutomationMaxRetryAttempts = 5

// Config keeps the declarative surface stable while allowing the runtime to
// flatten it into per-certificate execution state.
//
// Layering order for DNS settings is:
// 1. top-level dns shared defaults
// 2. dns_providers[name] reusable provider/account blocks
// 3. certificates[].dns final per-certificate override
type Config struct {
	CA           CAConfig             `yaml:"ca" json:"ca"`
	Account      AccountConfig        `yaml:"account" json:"account"`
	Automation   AutomationConfig     `yaml:"automation" json:"automation"`
	DNS          DNSConfig            `yaml:"dns" json:"dns"`
	DNSProviders map[string]DNSConfig `yaml:"dns_providers" json:"dns_providers"`
	Certificates []CertificateSpec    `yaml:"certificates" json:"certificates"`
}

// AutomationConfig controls long-running workflow behavior such as periodic
// renewal loops.
type AutomationConfig struct {
	// RenewInterval is a Go duration string such as 30m, 6h, or 24h.
	// Default is 24h so renew-loop can be started with minimal config.
	RenewInterval string `yaml:"renew_interval" json:"renew_interval"`
	// RetryBackoff is the initial wait duration before the first retry after a
	// scheduled renew cycle fails.
	RetryBackoff string `yaml:"retry_backoff" json:"retry_backoff"`
	// MaxRetryBackoff caps the exponential retry backoff.
	MaxRetryBackoff string `yaml:"max_retry_backoff" json:"max_retry_backoff"`
	// MaxRetryAttempts limits how many retries happen after one scheduled cycle
	// fails. Zero disables retries for that cycle.
	MaxRetryAttempts *int `yaml:"max_retry_attempts" json:"max_retry_attempts"`
	// FailureCommands run after each failed attempt. They receive ACME_LOOP_*
	// environment variables so callers can send alerts to external systems.
	FailureCommands []string `yaml:"failure_commands" json:"failure_commands"`
}

type CAConfig struct {
	DirectoryURL string `yaml:"directory_url" json:"directory_url"`
}

type AccountConfig struct {
	Email     string `yaml:"email" json:"email"`
	KeyPath   string `yaml:"key_path" json:"key_path"`
	AcceptTOS bool   `yaml:"accept_tos" json:"accept_tos"`
}

type DNSConfig struct {
	// ProviderRef selects a named entry from Config.DNSProviders.
	// It is primarily meant for certificates so a batch config can reuse one
	// provider definition across many certificate entries.
	ProviderRef                string            `yaml:"provider_ref" json:"provider_ref"`
	Provider                   string            `yaml:"provider" json:"provider"`
	AccountMode                string            `yaml:"account_mode" json:"account_mode"`
	Region                     string            `yaml:"region" json:"region"`
	Credentials                DNSCredentials    `yaml:"credentials" json:"credentials"`
	Env                        map[string]string `yaml:"env" json:"env"`
	DisableCNAMESupport        bool              `yaml:"disable_cname_support" json:"disable_cname_support"`
	DisableCompletePropagation bool              `yaml:"disable_complete_propagation" json:"disable_complete_propagation"`
	RecursiveNameservers       []string          `yaml:"recursive_nameservers" json:"recursive_nameservers"`
}

type DNSCredentials struct {
	AccessKey          string `yaml:"access_key" json:"access_key"`
	SecretKey          string `yaml:"secret_key" json:"secret_key"`
	SecretID           string `yaml:"secret_id" json:"secret_id"`
	SecretToken        string `yaml:"secret_token" json:"secret_token"`
	Username           string `yaml:"username" json:"username"`
	APIToken           string `yaml:"api_token" json:"api_token"`
	APIKey             string `yaml:"api_key" json:"api_key"`
	APIEmail           string `yaml:"api_email" json:"api_email"`
	ProjectID          string `yaml:"project_id" json:"project_id"`
	ServiceAccountFile string `yaml:"service_account_file" json:"service_account_file"`
	CompartmentID      string `yaml:"compartment_id" json:"compartment_id"`
	TenancyID          string `yaml:"tenancy_id" json:"tenancy_id"`
	UserID             string `yaml:"user_id" json:"user_id"`
	Fingerprint        string `yaml:"fingerprint" json:"fingerprint"`
	PrivateKeyFile     string `yaml:"private_key_file" json:"private_key_file"`
	PrivateKeyPass     string `yaml:"private_key_pass" json:"private_key_pass"`
	SubscriptionID     string `yaml:"subscription_id" json:"subscription_id"`
	TenantID           string `yaml:"tenant_id" json:"tenant_id"`
	ClientID           string `yaml:"client_id" json:"client_id"`
	ClientSecret       string `yaml:"client_secret" json:"client_secret"`
	ResourceGroup      string `yaml:"resource_group" json:"resource_group"`
}

type CertificateSpec struct {
	Name    string   `yaml:"name" json:"name"`
	Domains []string `yaml:"domains" json:"domains"`
	// DNS allows a certificate to pick a named provider or override parts of it
	// without duplicating the rest of the provider definition.
	DNS             DNSConfig        `yaml:"dns" json:"dns"`
	OutputDir       string           `yaml:"output_dir" json:"output_dir"`
	OutputFiles     CertificateFiles `yaml:"output_files" json:"output_files"`
	KeyType         string           `yaml:"key_type" json:"key_type"`
	Bundle          *bool            `yaml:"bundle" json:"bundle"`
	MustStaple      bool             `yaml:"must_staple" json:"must_staple"`
	PreferredChain  string           `yaml:"preferred_chain" json:"preferred_chain"`
	CSRPath         string           `yaml:"csr_path" json:"csr_path"`
	WebrootPath     string           `yaml:"webroot_path" json:"webroot_path"`
	HTTPBind        string           `yaml:"http_bind" json:"http_bind"`
	HTTPPort        string           `yaml:"http_port" json:"http_port"`
	TLSALPNBind     string           `yaml:"tlsalpn_bind" json:"tlsalpn_bind"`
	TLSALPNPort     string           `yaml:"tlsalpn_port" json:"tlsalpn_port"`
	RenewBeforeDays int              `yaml:"renew_before_days" json:"renew_before_days"`
	Challenge       string           `yaml:"challenge" json:"challenge"`
	Install         InstallConfig    `yaml:"install" json:"install"`
	Deploy          []DeployTarget   `yaml:"deploy" json:"deploy"`
	Hooks           HookConfig       `yaml:"hooks" json:"hooks"`
}

type CertificateFiles struct {
	CertFile      string `yaml:"cert_file" json:"cert_file"`
	KeyFile       string `yaml:"key_file" json:"key_file"`
	PublicKeyFile string `yaml:"public_key_file" json:"public_key_file"`
	FullChainFile string `yaml:"fullchain_file" json:"fullchain_file"`
	ChainFile     string `yaml:"chain_file" json:"chain_file"`
	MetadataFile  string `yaml:"metadata_file" json:"metadata_file"`
}

type CertificatePaths struct {
	CertFile      string `json:"cert_file"`
	KeyFile       string `json:"key_file"`
	PublicKeyFile string `json:"public_key_file"`
	FullChainFile string `json:"fullchain_file"`
	ChainFile     string `json:"chain_file"`
	MetadataFile  string `json:"metadata_file"`
}

func (c CertificateSpec) Paths() CertificatePaths {
	return CertificatePaths{
		CertFile:      resolveCertificatePath(c.OutputDir, c.OutputFiles.CertFile, "cert.pem"),
		KeyFile:       resolveCertificatePath(c.OutputDir, c.OutputFiles.KeyFile, "privkey.pem"),
		PublicKeyFile: resolveCertificatePath(c.OutputDir, c.OutputFiles.PublicKeyFile, "pubkey.pem"),
		FullChainFile: resolveCertificatePath(c.OutputDir, c.OutputFiles.FullChainFile, "fullchain.pem"),
		ChainFile:     resolveCertificatePath(c.OutputDir, c.OutputFiles.ChainFile, "issuer.pem"),
		MetadataFile:  resolveCertificatePath(c.OutputDir, c.OutputFiles.MetadataFile, "metadata.json"),
	}
}

func resolveCertificatePath(outputDir, configured, fallback string) string {
	name := configured
	if name == "" {
		name = fallback
	}
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(outputDir, name)
}

type InstallConfig struct {
	CertFile      string `yaml:"cert_file" json:"cert_file"`
	KeyFile       string `yaml:"key_file" json:"key_file"`
	PublicKeyFile string `yaml:"public_key_file" json:"public_key_file"`
	FullChainFile string `yaml:"fullchain_file" json:"fullchain_file"`
	ChainFile     string `yaml:"chain_file" json:"chain_file"`
	MetadataFile  string `yaml:"metadata_file" json:"metadata_file"`
}

type DeployTarget struct {
	Name          string            `yaml:"name" json:"name"`
	Type          string            `yaml:"type" json:"type"`
	Directory     string            `yaml:"directory" json:"directory"`
	Command       string            `yaml:"command" json:"command"`
	CertFile      string            `yaml:"cert_file" json:"cert_file"`
	KeyFile       string            `yaml:"key_file" json:"key_file"`
	PublicKeyFile string            `yaml:"public_key_file" json:"public_key_file"`
	FullChainFile string            `yaml:"fullchain_file" json:"fullchain_file"`
	ChainFile     string            `yaml:"chain_file" json:"chain_file"`
	MetadataFile  string            `yaml:"metadata_file" json:"metadata_file"`
	Env           map[string]string `yaml:"env" json:"env"`
}

type HookConfig struct {
	PreIssue    []string `yaml:"pre_issue" json:"pre_issue"`
	PostIssue   []string `yaml:"post_issue" json:"post_issue"`
	PreRenew    []string `yaml:"pre_renew" json:"pre_renew"`
	PostRenew   []string `yaml:"post_renew" json:"post_renew"`
	PostRevoke  []string `yaml:"post_revoke" json:"post_revoke"`
	PostInstall []string `yaml:"post_install" json:"post_install"`
	PostDeploy  []string `yaml:"post_deploy" json:"post_deploy"`
}
