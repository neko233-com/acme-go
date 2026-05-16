package volcengine

import (
	"acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
)

type Strategy struct{}

func (Strategy) ID() string { return "volcengine" }

func (Strategy) Aliases() []string { return []string{"volc", "volc-dns", "volcanicengine"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return NewDNSProvider()
}
