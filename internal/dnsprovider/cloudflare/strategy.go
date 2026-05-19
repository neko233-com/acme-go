package cloudflare

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	cloudflareprovider "github.com/go-acme/lego/v4/providers/dns/cloudflare"
)

type Strategy struct{}

func (Strategy) ID() string { return "cloudflare" }

func (Strategy) Aliases() []string { return []string{"cf"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return cloudflareprovider.NewDNSProvider()
}
