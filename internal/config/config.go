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
	Env                        map[string]string `yaml:"env" json:"env"`
	DisableCNAMESupport        bool              `yaml:"disable_cname_support" json:"disable_cname_support"`
	DisableCompletePropagation bool              `yaml:"disable_complete_propagation" json:"disable_complete_propagation"`
	RecursiveNameservers       []string          `yaml:"recursive_nameservers" json:"recursive_nameservers"`
}

type CertificateSpec struct {
	Name            string   `yaml:"name" json:"name"`
	Domains         []string `yaml:"domains" json:"domains"`
	OutputDir       string   `yaml:"output_dir" json:"output_dir"`
	KeyType         string   `yaml:"key_type" json:"key_type"`
	Bundle          *bool    `yaml:"bundle" json:"bundle"`
	MustStaple      bool     `yaml:"must_staple" json:"must_staple"`
	PreferredChain  string   `yaml:"preferred_chain" json:"preferred_chain"`
	CSRPath         string   `yaml:"csr_path" json:"csr_path"`
	RenewBeforeDays int      `yaml:"renew_before_days" json:"renew_before_days"`
	Challenge       string   `yaml:"challenge" json:"challenge"`
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
		if incoming.RenewBeforeDays != 0 {
			merged.RenewBeforeDays = incoming.RenewBeforeDays
		}
		if incoming.Challenge != "" {
			merged.Challenge = incoming.Challenge
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
		if cert.Challenge != "dns-01" {
			return fmt.Errorf("certificate %q uses unsupported challenge %q", cert.Name, cert.Challenge)
		}
		switch cert.KeyType {
		case "ec256", "ec384", "rsa2048", "rsa4096", "rsa8192":
		default:
			return fmt.Errorf("certificate %q uses unsupported key_type %q", cert.Name, cert.KeyType)
		}
	}
	return nil
}
