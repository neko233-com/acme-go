package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultCADirectoryURL = "https://acme-v02.api.letsencrypt.org/directory"

type Config struct {
	CA           CAConfig          `yaml:"ca"`
	Account      AccountConfig     `yaml:"account"`
	DNS          DNSConfig         `yaml:"dns"`
	Certificates []CertificateSpec `yaml:"certificates"`
}

type CAConfig struct {
	DirectoryURL string `yaml:"directory_url"`
}

type AccountConfig struct {
	Email     string `yaml:"email"`
	KeyPath   string `yaml:"key_path"`
	AcceptTOS bool   `yaml:"accept_tos"`
}

type DNSConfig struct {
	Provider string            `yaml:"provider"`
	Env      map[string]string `yaml:"env"`
}

type CertificateSpec struct {
	Name            string   `yaml:"name"`
	Domains         []string `yaml:"domains"`
	OutputDir       string   `yaml:"output_dir"`
	KeyType         string   `yaml:"key_type"`
	Bundle          bool     `yaml:"bundle"`
	RenewBeforeDays int      `yaml:"renew_before_days"`
	Challenge       string   `yaml:"challenge"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(data))), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.applyDefaults(path); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
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
		if !cert.Bundle {
			cert.Bundle = true
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
