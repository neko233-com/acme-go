package baiducloud

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	bceprovider "github.com/go-acme/lego/v4/providers/dns/baiducloud"
)

type Strategy struct{}

func (Strategy) ID() string { return "baiducloud" }

func (Strategy) Aliases() []string { return []string{"baidu", "bce", "baidu-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return bceprovider.NewDNSProvider()
}
