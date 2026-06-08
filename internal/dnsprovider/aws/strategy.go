package aws

import (
	"github.com/neko233-com/acme233/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/route53"
)

type Strategy struct{}

func (Strategy) ID() string { return "aws" }

func (Strategy) Aliases() []string {
	return []string{"route53", "amazon", "amazonwebservices", "aws-intl", "aws-global"}
}

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return route53.NewDNSProvider()
}
