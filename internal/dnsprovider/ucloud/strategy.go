package ucloud

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	ucloudprovider "github.com/go-acme/lego/v4/providers/dns/ucloud"
)

type Strategy struct{}

func (Strategy) ID() string { return "ucloud" }

func (Strategy) Aliases() []string { return []string{"ucloud-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return ucloudprovider.NewDNSProvider()
}
