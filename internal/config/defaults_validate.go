package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func (c *Config) applyDefaults(configPath string) error {
	baseDir := filepath.Dir(configPath)
	if c.CA.DirectoryURL == "" {
		c.CA.DirectoryURL = defaultCADirectoryURL
	}
	if c.Account.KeyPath == "" {
		c.Account.KeyPath = filepath.Join(baseDir, ".acme", "account.pem")
	}
	if strings.TrimSpace(c.Automation.RenewInterval) == "" {
		c.Automation.RenewInterval = defaultAutomationRenewInterval
	}
	if c.DNS.Env == nil {
		c.DNS.Env = map[string]string{}
	}
	normalizeDNSConfig(&c.DNS)
	for name, dnsConfig := range c.DNSProviders {
		normalizeDNSConfig(&dnsConfig)
		c.DNSProviders[name] = dnsConfig
	}
	for idx := range c.Certificates {
		cert := &c.Certificates[idx]
		if cert.Name == "" && len(cert.Domains) > 0 {
			cert.Name = cert.Domains[0]
		}
		normalizeDNSConfig(&cert.DNS)
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
	if _, err := c.Automation.RenewIntervalDuration(); err != nil {
		return err
	}
	if err := validateDNSConfig("dns", c.DNS, false); err != nil {
		return err
	}
	for name, dnsConfig := range c.DNSProviders {
		if err := validateDNSConfig(fmt.Sprintf("dns_providers.%s", name), dnsConfig, dnsConfig.ProviderRef == ""); err != nil {
			return err
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
		if err := validateDNSConfig(fmt.Sprintf("certificate %q dns", cert.Name), cert.DNS, false); err != nil {
			return err
		}
		if cert.Challenge == "dns-01" {
			effectiveDNS, err := c.EffectiveDNS(cert)
			if err != nil {
				return err
			}
			if effectiveDNS.Provider == "" {
				return fmt.Errorf("certificate %q requires a dns provider; set dns.provider, dns_providers.<name>, or certificates[].dns", cert.Name)
			}
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

// RenewIntervalDuration parses the configured auto-renew interval.
// An empty interval means no periodic renew loop is configured.
func (a AutomationConfig) RenewIntervalDuration() (time.Duration, error) {
	if strings.TrimSpace(a.RenewInterval) == "" {
		return 0, nil
	}
	interval, err := time.ParseDuration(strings.TrimSpace(a.RenewInterval))
	if err != nil {
		return 0, fmt.Errorf("automation.renew_interval must be a valid duration: %w", err)
	}
	if interval <= 0 {
		return 0, fmt.Errorf("automation.renew_interval must be greater than zero")
	}
	return interval, nil
}

func validateDNSConfig(scope string, dnsConfig DNSConfig, requireProvider bool) error {
	if requireProvider && dnsConfig.Provider == "" && dnsConfig.ProviderRef == "" {
		return fmt.Errorf("%s.provider is required", scope)
	}
	if dnsConfig.AccountMode != "" {
		switch dnsConfig.AccountMode {
		case "cn", "intl", "global":
		default:
			return fmt.Errorf("%s.account_mode must be one of cn, intl, global", scope)
		}
	}
	return nil
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
