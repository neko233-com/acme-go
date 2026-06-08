package digitalocean

import (
	"github.com/neko233-com/acme233/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	doprovider "github.com/go-acme/lego/v4/providers/dns/digitalocean"
)

type Strategy struct{}

func (Strategy) ID() string { return "digitalocean" }

func (Strategy) Aliases() []string { return []string{"do", "digital-ocean"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return doprovider.NewDNSProvider()
}
