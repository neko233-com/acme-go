package vultr

import (
	"github.com/neko233-com/acme233/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	vultrprovider "github.com/go-acme/lego/v4/providers/dns/vultr"
)

type Strategy struct{}

func (Strategy) ID() string { return "vultr" }

func (Strategy) Aliases() []string { return []string{"vultr-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return vultrprovider.NewDNSProvider()
}
