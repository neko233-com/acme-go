package azure

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	azuredns "github.com/go-acme/lego/v4/providers/dns/azuredns"
)

type Strategy struct{}

func (Strategy) ID() string { return "azure" }

func (Strategy) Aliases() []string {
	return []string{"azuredns", "azure-dns", "azure-intl", "azure-global"}
}

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return azuredns.NewDNSProvider()
}
