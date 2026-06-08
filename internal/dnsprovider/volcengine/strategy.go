package volcengine

import (
	"github.com/neko233-com/acme233/internal/config"

	"github.com/go-acme/lego/v4/challenge"
)

type Strategy struct{}

func (Strategy) ID() string { return "volcengine" }

func (Strategy) Aliases() []string {
	return []string{"volc", "volc-dns", "volcanicengine", "volcengine-cn", "volcengine-intl", "volcengine-global"}
}

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return NewDNSProvider()
}
