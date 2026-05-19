package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultCADirectoryURL = "https://acme-v02.api.letsencrypt.org/directory"

type Config struct {
	CA           CAConfig          `yaml:"ca" json:"ca"`
	Account      AccountConfig     `yaml:"account" json:"account"`
	DNS          DNSConfig         `yaml:"dns" json:"dns"`
	Certificates []CertificateSpec `yaml:"certificates" json:"certificates"`
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
	Name            string           `yaml:"name" json:"name"`
	Domains         []string         `yaml:"domains" json:"domains"`
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

func Load(path string) (*Config, error) {
	cfg, err := loadYAML(path)
	if err != nil {
		return nil, err
	}

	localOverridePath := filepath.Join(filepath.Dir(path), ".local.json")
	if override, err := loadJSON(localOverridePath); err == nil {
		cfg.merge(override)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := cfg.applyDefaults(path); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadYAML(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(data))), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func loadJSON(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal([]byte(os.ExpandEnv(string(data))), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse local override %s: %w", path, err)
	}
	return cfg, nil
}

func (c *Config) merge(override Config) {
	if override.CA.DirectoryURL != "" {
		c.CA.DirectoryURL = override.CA.DirectoryURL
	}
	if override.Account.Email != "" {
		c.Account.Email = override.Account.Email
	}
	if override.Account.KeyPath != "" {
		c.Account.KeyPath = override.Account.KeyPath
	}
	if override.Account.AcceptTOS {
		c.Account.AcceptTOS = true
	}
	if override.DNS.Provider != "" {
		c.DNS.Provider = override.DNS.Provider
	}
	if override.DNS.AccountMode != "" {
		c.DNS.AccountMode = override.DNS.AccountMode
	}
	if override.DNS.Region != "" {
		c.DNS.Region = override.DNS.Region
	}
	mergeDNSCredentials(&c.DNS.Credentials, override.DNS.Credentials)
	if override.DNS.DisableCNAMESupport {
		c.DNS.DisableCNAMESupport = true
	}
	if override.DNS.DisableCompletePropagation {
		c.DNS.DisableCompletePropagation = true
	}
	if len(override.DNS.RecursiveNameservers) > 0 {
		c.DNS.RecursiveNameservers = append([]string(nil), override.DNS.RecursiveNameservers...)
	}
	if c.DNS.Env == nil {
		c.DNS.Env = map[string]string{}
	}
	for key, value := range override.DNS.Env {
		c.DNS.Env[key] = value
	}
	if len(override.Certificates) == 0 {
		return
	}

	byName := map[string]int{}
	for index, cert := range c.Certificates {
		byName[cert.Name] = index
	}
	for _, incoming := range override.Certificates {
		index, exists := byName[incoming.Name]
		if !exists {
			c.Certificates = append(c.Certificates, incoming)
			continue
		}
		merged := c.Certificates[index]
		if len(incoming.Domains) > 0 {
			merged.Domains = append([]string(nil), incoming.Domains...)
		}
		if incoming.OutputDir != "" {
			merged.OutputDir = incoming.OutputDir
		}
		mergeCertificateFiles(&merged.OutputFiles, incoming.OutputFiles)
		if incoming.KeyType != "" {
			merged.KeyType = incoming.KeyType
		}
		if incoming.Bundle != nil {
			merged.Bundle = incoming.Bundle
		}
		if incoming.MustStaple {
			merged.MustStaple = true
		}
		if incoming.PreferredChain != "" {
			merged.PreferredChain = incoming.PreferredChain
		}
		if incoming.CSRPath != "" {
			merged.CSRPath = incoming.CSRPath
		}
		if incoming.WebrootPath != "" {
			merged.WebrootPath = incoming.WebrootPath
		}
		if incoming.HTTPBind != "" {
			merged.HTTPBind = incoming.HTTPBind
		}
		if incoming.HTTPPort != "" {
			merged.HTTPPort = incoming.HTTPPort
		}
		if incoming.TLSALPNBind != "" {
			merged.TLSALPNBind = incoming.TLSALPNBind
		}
		if incoming.TLSALPNPort != "" {
			merged.TLSALPNPort = incoming.TLSALPNPort
		}
		if incoming.RenewBeforeDays != 0 {
			merged.RenewBeforeDays = incoming.RenewBeforeDays
		}
		if incoming.Challenge != "" {
			merged.Challenge = incoming.Challenge
		}
		mergeInstallConfig(&merged.Install, incoming.Install)
		mergeHooks(&merged.Hooks, incoming.Hooks)
		if len(incoming.Deploy) > 0 {
			merged.Deploy = append([]DeployTarget(nil), incoming.Deploy...)
		}
		c.Certificates[index] = merged
	}
}

func (c *Config) applyDefaults(configPath string) error {
	baseDir := filepath.Dir(configPath)
	if c.CA.DirectoryURL == "" {
		c.CA.DirectoryURL = defaultCADirectoryURL
	}
	if c.Account.KeyPath == "" {
		c.Account.KeyPath = filepath.Join(baseDir, ".acme", "account.pem")
	}
	if c.DNS.Env == nil {
		c.DNS.Env = map[string]string{}
	}
	c.DNS.Provider = strings.TrimSpace(c.DNS.Provider)
	c.DNS.AccountMode = normalizeAccountMode(c.DNS.AccountMode)
	c.DNS.Region = strings.TrimSpace(c.DNS.Region)
	for key, value := range deriveDNSEnv(c.DNS) {
		if _, exists := c.DNS.Env[key]; !exists {
			c.DNS.Env[key] = value
		}
	}
	for idx := range c.Certificates {
		cert := &c.Certificates[idx]
		if cert.Name == "" && len(cert.Domains) > 0 {
			cert.Name = cert.Domains[0]
		}
		if cert.OutputDir == "" {
			cert.OutputDir = filepath.Join(baseDir, "certs", cert.Name)
		}
		if cert.KeyType == "" {
			cert.KeyType = "ec256"
		}
		cert.KeyType = strings.ToLower(cert.KeyType)
		if cert.RenewBeforeDays == 0 {
			cert.RenewBeforeDays = 30
		}
		if cert.Challenge == "" {
			cert.Challenge = "dns-01"
		}
		cert.Challenge = strings.ToLower(cert.Challenge)
		if cert.HTTPBind == "" {
			cert.HTTPBind = "0.0.0.0"
		}
		if cert.HTTPPort == "" {
			cert.HTTPPort = "80"
		}
		if cert.TLSALPNBind == "" {
			cert.TLSALPNBind = "0.0.0.0"
		}
		if cert.TLSALPNPort == "" {
			cert.TLSALPNPort = "443"
		}
		if cert.Bundle == nil {
			defaultBundle := true
			cert.Bundle = &defaultBundle
		}
	}
	return nil
}

func (c *Config) Validate() error {
	if c.Account.Email == "" {
		return fmt.Errorf("account.email is required")
	}
	if !c.Account.AcceptTOS {
		return fmt.Errorf("account.accept_tos must be true")
	}
	if c.DNS.Provider == "" {
		return fmt.Errorf("dns.provider is required")
	}
	if c.DNS.AccountMode != "" {
		switch c.DNS.AccountMode {
		case "cn", "intl", "global":
		default:
			return fmt.Errorf("dns.account_mode must be one of cn, intl, global")
		}
	}
	if len(c.Certificates) == 0 {
		return fmt.Errorf("at least one certificate entry is required")
	}
	seen := map[string]struct{}{}
	for _, cert := range c.Certificates {
		if cert.Name == "" {
			return fmt.Errorf("certificate.name is required")
		}
		if _, exists := seen[cert.Name]; exists {
			return fmt.Errorf("duplicate certificate name %q", cert.Name)
		}
		seen[cert.Name] = struct{}{}
		if len(cert.Domains) == 0 {
			return fmt.Errorf("certificate %q must declare at least one domain", cert.Name)
		}
		switch cert.Challenge {
		case "dns-01", "http-01", "webroot", "standalone", "tls-alpn-01":
		default:
			return fmt.Errorf("certificate %q uses unsupported challenge %q", cert.Name, cert.Challenge)
		}
		if cert.Challenge == "webroot" && cert.WebrootPath == "" {
			return fmt.Errorf("certificate %q requires webroot_path for webroot challenge", cert.Name)
		}
		if cert.Challenge == "tls-alpn-01" && len(cert.Domains) > 1 {
			for _, domain := range cert.Domains {
				if strings.HasPrefix(domain, "*.") {
					return fmt.Errorf("certificate %q cannot use wildcard domains with tls-alpn-01", cert.Name)
				}
			}
		}
		if err := validateInstallConfig(cert.Name, cert.Install); err != nil {
			return err
		}
		if err := validateDeployTargets(cert.Name, cert.Deploy); err != nil {
			return err
		}
		switch cert.KeyType {
		case "ec256", "ec384", "rsa2048", "rsa4096", "rsa8192":
		default:
			return fmt.Errorf("certificate %q uses unsupported key_type %q", cert.Name, cert.KeyType)
		}
	}
	return nil
}

func mergeCertificateFiles(base *CertificateFiles, incoming CertificateFiles) {
	if incoming.CertFile != "" {
		base.CertFile = incoming.CertFile
	}
	if incoming.KeyFile != "" {
		base.KeyFile = incoming.KeyFile
	}
	if incoming.PublicKeyFile != "" {
		base.PublicKeyFile = incoming.PublicKeyFile
	}
	if incoming.FullChainFile != "" {
		base.FullChainFile = incoming.FullChainFile
	}
	if incoming.ChainFile != "" {
		base.ChainFile = incoming.ChainFile
	}
	if incoming.MetadataFile != "" {
		base.MetadataFile = incoming.MetadataFile
	}
}

func mergeInstallConfig(base *InstallConfig, incoming InstallConfig) {
	if incoming.CertFile != "" {
		base.CertFile = incoming.CertFile
	}
	if incoming.KeyFile != "" {
		base.KeyFile = incoming.KeyFile
	}
	if incoming.PublicKeyFile != "" {
		base.PublicKeyFile = incoming.PublicKeyFile
	}
	if incoming.FullChainFile != "" {
		base.FullChainFile = incoming.FullChainFile
	}
	if incoming.ChainFile != "" {
		base.ChainFile = incoming.ChainFile
	}
	if incoming.MetadataFile != "" {
		base.MetadataFile = incoming.MetadataFile
	}
}

func mergeDNSCredentials(base *DNSCredentials, incoming DNSCredentials) {
	if incoming.AccessKey != "" {
		base.AccessKey = incoming.AccessKey
	}
	if incoming.SecretKey != "" {
		base.SecretKey = incoming.SecretKey
	}
	if incoming.SecretID != "" {
		base.SecretID = incoming.SecretID
	}
	if incoming.SecretToken != "" {
		base.SecretToken = incoming.SecretToken
	}
	if incoming.Username != "" {
		base.Username = incoming.Username
	}
	if incoming.APIToken != "" {
		base.APIToken = incoming.APIToken
	}
	if incoming.APIKey != "" {
		base.APIKey = incoming.APIKey
	}
	if incoming.APIEmail != "" {
		base.APIEmail = incoming.APIEmail
	}
	if incoming.ProjectID != "" {
		base.ProjectID = incoming.ProjectID
	}
	if incoming.ServiceAccountFile != "" {
		base.ServiceAccountFile = incoming.ServiceAccountFile
	}
	if incoming.CompartmentID != "" {
		base.CompartmentID = incoming.CompartmentID
	}
	if incoming.TenancyID != "" {
		base.TenancyID = incoming.TenancyID
	}
	if incoming.UserID != "" {
		base.UserID = incoming.UserID
	}
	if incoming.Fingerprint != "" {
		base.Fingerprint = incoming.Fingerprint
	}
	if incoming.PrivateKeyFile != "" {
		base.PrivateKeyFile = incoming.PrivateKeyFile
	}
	if incoming.PrivateKeyPass != "" {
		base.PrivateKeyPass = incoming.PrivateKeyPass
	}
	if incoming.SubscriptionID != "" {
		base.SubscriptionID = incoming.SubscriptionID
	}
	if incoming.TenantID != "" {
		base.TenantID = incoming.TenantID
	}
	if incoming.ClientID != "" {
		base.ClientID = incoming.ClientID
	}
	if incoming.ClientSecret != "" {
		base.ClientSecret = incoming.ClientSecret
	}
	if incoming.ResourceGroup != "" {
		base.ResourceGroup = incoming.ResourceGroup
	}
}

func deriveDNSEnv(cfg DNSConfig) map[string]string {
	values := map[string]string{}
	provider := detectProviderFamily(cfg.Provider)
	mode := normalizeAccountMode(cfg.AccountMode)
	set := func(key, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		values[key] = value
	}

	switch provider {
	case "alicloud":
		set("ALICLOUD_ACCESS_KEY", cfg.Credentials.AccessKey)
		set("ALICLOUD_SECRET_KEY", cfg.Credentials.SecretKey)
	case "digitalocean":
		set("DO_AUTH_TOKEN", firstNonEmpty(cfg.Credentials.APIToken, cfg.Credentials.APIKey))
	case "baiducloud":
		set("BAIDUCLOUD_ACCESS_KEY_ID", firstNonEmpty(cfg.Credentials.AccessKey, cfg.Credentials.SecretID))
		set("BAIDUCLOUD_SECRET_ACCESS_KEY", cfg.Credentials.SecretKey)
	case "volcengine":
		set("VOLC_ACCESSKEY", cfg.Credentials.AccessKey)
		set("VOLC_SECRETKEY", cfg.Credentials.SecretKey)
		if cfg.Region != "" {
			set("VOLC_REGION", cfg.Region)
		} else if mode == "intl" || mode == "global" {
			set("VOLC_REGION", "ap-singapore")
		} else if mode == "cn" {
			set("VOLC_REGION", "cn-beijing")
		}
	case "tencentcloud":
		set("TENCENTCLOUD_SECRET_ID", firstNonEmpty(cfg.Credentials.SecretID, cfg.Credentials.AccessKey))
		set("TENCENTCLOUD_SECRET_KEY", cfg.Credentials.SecretKey)
		set("TENCENTCLOUD_SESSION_TOKEN", cfg.Credentials.SecretToken)
	case "cloudflare":
		set("CLOUDFLARE_DNS_API_TOKEN", firstNonEmpty(cfg.Credentials.APIToken, cfg.Credentials.APIKey))
		set("CLOUDFLARE_EMAIL", cfg.Credentials.APIEmail)
	case "aws":
		set("AWS_ACCESS_KEY_ID", firstNonEmpty(cfg.Credentials.AccessKey, cfg.Credentials.SecretID))
		set("AWS_SECRET_ACCESS_KEY", cfg.Credentials.SecretKey)
		set("AWS_SESSION_TOKEN", cfg.Credentials.SecretToken)
		set("AWS_REGION", firstNonEmpty(cfg.Region, defaultAWSRegion(mode)))
	case "gcloud":
		set("GCE_PROJECT", cfg.Credentials.ProjectID)
		set("GOOGLE_APPLICATION_CREDENTIALS", cfg.Credentials.ServiceAccountFile)
	case "hetzner":
		set("HETZNER_API_TOKEN", firstNonEmpty(cfg.Credentials.APIToken, cfg.Credentials.APIKey))
	case "huaweicloud":
		set("HUAWEICLOUD_ACCESS_KEY_ID", firstNonEmpty(cfg.Credentials.AccessKey, cfg.Credentials.SecretID))
		set("HUAWEICLOUD_SECRET_ACCESS_KEY", cfg.Credentials.SecretKey)
		set("HUAWEICLOUD_REGION", cfg.Region)
	case "linode":
		set("LINODE_TOKEN", firstNonEmpty(cfg.Credentials.APIToken, cfg.Credentials.APIKey))
	case "ibmcloud":
		set("SOFTLAYER_USERNAME", cfg.Credentials.Username)
		set("SOFTLAYER_API_KEY", firstNonEmpty(cfg.Credentials.APIKey, cfg.Credentials.APIToken))
	case "oraclecloud":
		set("OCI_COMPARTMENT_OCID", cfg.Credentials.CompartmentID)
		set("OCI_REGION", cfg.Region)
		set("OCI_TENANCY_OCID", cfg.Credentials.TenancyID)
		set("OCI_USER_OCID", cfg.Credentials.UserID)
		set("OCI_PUBKEY_FINGERPRINT", cfg.Credentials.Fingerprint)
		set("OCI_PRIVKEY_FILE", cfg.Credentials.PrivateKeyFile)
		set("OCI_PRIVKEY_PASS", cfg.Credentials.PrivateKeyPass)
	case "scaleway":
		set("SCALEWAY_API_TOKEN", firstNonEmpty(cfg.Credentials.APIToken, cfg.Credentials.SecretKey, cfg.Credentials.APIKey))
		set("SCALEWAY_PROJECT_ID", cfg.Credentials.ProjectID)
	case "azure":
		set("AZURE_SUBSCRIPTION_ID", cfg.Credentials.SubscriptionID)
		set("AZURE_TENANT_ID", cfg.Credentials.TenantID)
		set("AZURE_CLIENT_ID", cfg.Credentials.ClientID)
		set("AZURE_CLIENT_SECRET", cfg.Credentials.ClientSecret)
		set("AZURE_RESOURCE_GROUP", cfg.Credentials.ResourceGroup)
	case "ucloud":
		set("UCLOUD_PUBLIC_KEY", firstNonEmpty(cfg.Credentials.AccessKey, cfg.Credentials.SecretID))
		set("UCLOUD_PRIVATE_KEY", cfg.Credentials.SecretKey)
		set("UCLOUD_REGION", cfg.Region)
		set("UCLOUD_PROJECT_ID", cfg.Credentials.ProjectID)
	case "vultr":
		set("VULTR_API_KEY", firstNonEmpty(cfg.Credentials.APIKey, cfg.Credentials.APIToken))
	}

	return values
}

func detectProviderFamily(provider string) string {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(provider)))
	switch normalized {
	case "alicloud", "aliyun", "alidns", "alibaba", "alicloudcn", "alicloudintl", "alicloudglobal", "aliyunintl", "alibabacloud":
		return "alicloud"
	case "aws", "route53", "amazon", "amazonwebservices", "awsglobal", "awsintl":
		return "aws"
	case "azure", "azuredns", "azurednsglobal", "azurednsintl":
		return "azure"
	case "baiducloud", "baidu", "bce", "baidudns":
		return "baiducloud"
	case "cloudflare", "cf", "cloudflareglobal":
		return "cloudflare"
	case "digitalocean", "digitaloceancom", "digitaloceancloud", "do":
		return "digitalocean"
	case "gcloud", "gcp", "googlecloud", "googlecloudglobal", "google":
		return "gcloud"
	case "hetzner", "hcloud", "hetznerdns":
		return "hetzner"
	case "huaweicloud", "huawei", "huaweicloudcn", "huaweicloudintl", "huaweidns", "huaweicloudglobal":
		return "huaweicloud"
	case "ibmcloud", "ibm", "softlayer", "ibmdns":
		return "ibmcloud"
	case "linode", "linodedns":
		return "linode"
	case "oraclecloud", "oracle", "oci", "oracledns":
		return "oraclecloud"
	case "scaleway", "scw", "scalewaydns":
		return "scaleway"
	case "tencentcloud", "tencent", "dnspod", "tencentdns", "tencentcloudcn", "tencentcloudintl", "dnspodintl":
		return "tencentcloud"
	case "ucloud", "uclouddns":
		return "ucloud"
	case "vultr", "vultrdns":
		return "vultr"
	case "volcengine", "volc", "volcdns", "volcanicengine", "volcenginecn", "volcengineintl", "volcengineglobal":
		return "volcengine"
	default:
		return normalized
	}
}

func normalizeAccountMode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "international", "overseas":
		return "intl"
	case "domestic", "mainland", "china":
		return "cn"
	default:
		return normalized
	}
}

func defaultAWSRegion(mode string) string {
	if mode == "cn" {
		return "cn-north-1"
	}
	return "us-east-1"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mergeHooks(base *HookConfig, incoming HookConfig) {
	if len(incoming.PreIssue) > 0 {
		base.PreIssue = append([]string(nil), incoming.PreIssue...)
	}
	if len(incoming.PostIssue) > 0 {
		base.PostIssue = append([]string(nil), incoming.PostIssue...)
	}
	if len(incoming.PreRenew) > 0 {
		base.PreRenew = append([]string(nil), incoming.PreRenew...)
	}
	if len(incoming.PostRenew) > 0 {
		base.PostRenew = append([]string(nil), incoming.PostRenew...)
	}
	if len(incoming.PostRevoke) > 0 {
		base.PostRevoke = append([]string(nil), incoming.PostRevoke...)
	}
	if len(incoming.PostInstall) > 0 {
		base.PostInstall = append([]string(nil), incoming.PostInstall...)
	}
	if len(incoming.PostDeploy) > 0 {
		base.PostDeploy = append([]string(nil), incoming.PostDeploy...)
	}
}

func validateInstallConfig(_ string, install InstallConfig) error {
	if install.CertFile == "" && install.KeyFile == "" && install.PublicKeyFile == "" && install.FullChainFile == "" && install.ChainFile == "" && install.MetadataFile == "" {
		return nil
	}
	return nil
}

func validateDeployTargets(name string, targets []DeployTarget) error {
	for _, target := range targets {
		switch strings.ToLower(target.Type) {
		case "", "copy":
			if target.Directory == "" && target.CertFile == "" && target.KeyFile == "" && target.PublicKeyFile == "" && target.FullChainFile == "" && target.ChainFile == "" && target.MetadataFile == "" {
				return fmt.Errorf("certificate %q deploy target %q requires directory or explicit output files", name, target.Name)
			}
		case "command":
			if strings.TrimSpace(target.Command) == "" {
				return fmt.Errorf("certificate %q deploy target %q requires command", name, target.Name)
			}
		default:
			return fmt.Errorf("certificate %q deploy target %q uses unsupported type %q", name, target.Name, target.Type)
		}
	}
	return nil
}
