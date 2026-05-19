package linode

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	linodeprovider "github.com/go-acme/lego/v4/providers/dns/linode"
)

type Strategy struct{}

func (Strategy) ID() string { return "linode" }

func (Strategy) Aliases() []string { return []string{"linode-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return linodeprovider.NewDNSProvider()
}
