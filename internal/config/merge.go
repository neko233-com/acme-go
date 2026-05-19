package config

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
	if override.Automation.RenewInterval != "" {
		c.Automation.RenewInterval = override.Automation.RenewInterval
	}
	mergeDNSConfig(&c.DNS, override.DNS)
	if len(override.DNSProviders) > 0 {
		if c.DNSProviders == nil {
			c.DNSProviders = map[string]DNSConfig{}
		}
		for name, incoming := range override.DNSProviders {
			merged := c.DNSProviders[name]
			mergeDNSConfig(&merged, incoming)
			c.DNSProviders[name] = merged
		}
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
		mergeDNSConfig(&merged.DNS, incoming.DNS)
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

// mergeDNSConfig applies non-zero fields from incoming onto base so top-level
// defaults, named providers, and certificate overrides can share one merge
// model.
func mergeDNSConfig(base *DNSConfig, incoming DNSConfig) {
	if incoming.ProviderRef != "" {
		base.ProviderRef = incoming.ProviderRef
	}
	if incoming.Provider != "" {
		base.Provider = incoming.Provider
	}
	if incoming.AccountMode != "" {
		base.AccountMode = incoming.AccountMode
	}
	if incoming.Region != "" {
		base.Region = incoming.Region
	}
	mergeDNSCredentials(&base.Credentials, incoming.Credentials)
	if incoming.DisableCNAMESupport {
		base.DisableCNAMESupport = true
	}
	if incoming.DisableCompletePropagation {
		base.DisableCompletePropagation = true
	}
	if len(incoming.RecursiveNameservers) > 0 {
		base.RecursiveNameservers = append([]string(nil), incoming.RecursiveNameservers...)
	}
	if len(incoming.Env) > 0 {
		if base.Env == nil {
			base.Env = map[string]string{}
		}
		for key, value := range incoming.Env {
			base.Env[key] = value
		}
	}
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
