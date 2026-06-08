package oraclecloud

import (
	"github.com/neko233-com/acme233/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	ociprovider "github.com/go-acme/lego/v4/providers/dns/oraclecloud"
)

type Strategy struct{}

func (Strategy) ID() string { return "oraclecloud" }

func (Strategy) Aliases() []string { return []string{"oracle", "oci", "oracle-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return ociprovider.NewDNSProvider()
}
