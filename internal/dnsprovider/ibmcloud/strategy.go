package ibmcloud

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	ibmprovider "github.com/go-acme/lego/v4/providers/dns/ibmcloud"
)

type Strategy struct{}

func (Strategy) ID() string { return "ibmcloud" }

func (Strategy) Aliases() []string { return []string{"ibm", "softlayer", "ibm-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return ibmprovider.NewDNSProvider()
}
