package scaleway

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	scwprovider "github.com/go-acme/lego/v4/providers/dns/scaleway"
)

type Strategy struct{}

func (Strategy) ID() string { return "scaleway" }

func (Strategy) Aliases() []string { return []string{"scw", "scaleway-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return scwprovider.NewDNSProvider()
}
