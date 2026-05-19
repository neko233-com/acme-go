package hetzner

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	hetznprovider "github.com/go-acme/lego/v4/providers/dns/hetzner"
)

type Strategy struct{}

func (Strategy) ID() string { return "hetzner" }

func (Strategy) Aliases() []string { return []string{"hcloud", "hetzner-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return hetznprovider.NewDNSProvider()
}
