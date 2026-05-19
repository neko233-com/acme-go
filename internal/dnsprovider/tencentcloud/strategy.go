package tencentcloud

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	tencentprovider "github.com/go-acme/lego/v4/providers/dns/tencentcloud"
)

type Strategy struct{}

func (Strategy) ID() string { return "tencentcloud" }

func (Strategy) Aliases() []string { return []string{"tencent", "dnspod", "tencent-dns"} }

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return tencentprovider.NewDNSProvider()
}
