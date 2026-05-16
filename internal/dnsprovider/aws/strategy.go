package aws

import (
	"acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/route53"
)

type Strategy struct{}

func (Strategy) ID() string { return "aws" }

func (Strategy) Aliases() []string { return []string{"route53", "amazon", "amazonwebservices"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return route53.NewDNSProvider()
}
