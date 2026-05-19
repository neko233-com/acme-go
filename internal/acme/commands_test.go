package acme

import (
	"testing"

	"github.com/neko233-com/acme-go/internal/config"
)

func TestInspectCertificateUsesEffectiveDNSProvider(t *testing.T) {
	cfg := &config.Config{
		DNS: config.DNSConfig{Provider: "cloudflare"},
		DNSProviders: map[string]config.DNSConfig{
			"aliyun-main": {Provider: "alicloud-cn"},
		},
	}
	cert := config.CertificateSpec{
		Name:      "example",
		Domains:   []string{"example.com"},
		Challenge: "dns-01",
		DNS: config.DNSConfig{
			ProviderRef: "aliyun-main",
		},
	}

	info, err := inspectCertificate(cfg, cert)
	if err != nil {
		t.Fatalf("inspectCertificate: %v", err)
	}
	if info.Provider != "alicloud-cn" {
		t.Fatalf("provider: got %q want %q", info.Provider, "alicloud-cn")
	}
}
