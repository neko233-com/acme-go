package config

import (
	"fmt"
	"strings"
)

// EffectiveDNS resolves the final provider configuration for one certificate.
// The returned config is ready for runtime consumption and env derivation.
func (c Config) EffectiveDNS(cert CertificateSpec) (DNSConfig, error) {
	effective := c.DNS
	if cert.DNS.ProviderRef != "" {
		providerConfig, exists := c.DNSProviders[cert.DNS.ProviderRef]
		if !exists {
			return DNSConfig{}, fmt.Errorf("certificate %q references unknown dns provider %q", cert.Name, cert.DNS.ProviderRef)
		}
		mergeDNSConfig(&effective, providerConfig)
	}
	mergeDNSConfig(&effective, cert.DNS)
	effective.ProviderRef = ""
	normalizeDNSConfig(&effective)
	return effective, nil
}

// normalizeDNSConfig applies whitespace normalization and derives env vars once
// credentials are present. Explicit env keys keep precedence over derived keys.
func normalizeDNSConfig(dnsConfig *DNSConfig) {
	if dnsConfig.Env == nil {
		dnsConfig.Env = map[string]string{}
	}
	dnsConfig.ProviderRef = strings.TrimSpace(dnsConfig.ProviderRef)
	dnsConfig.Provider = strings.TrimSpace(dnsConfig.Provider)
	dnsConfig.AccountMode = normalizeAccountMode(dnsConfig.AccountMode)
	dnsConfig.Region = strings.TrimSpace(dnsConfig.Region)
	for key, value := range deriveDNSEnv(*dnsConfig) {
		if _, exists := dnsConfig.Env[key]; !exists {
			dnsConfig.Env[key] = value
		}
	}
}

// deriveDNSEnv maps the low-config credential model into provider-specific env
// variables expected by lego DNS drivers.
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
