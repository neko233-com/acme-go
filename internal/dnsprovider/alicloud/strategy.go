package alicloud

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
)

type Strategy struct{}

func (Strategy) ID() string { return "alicloud" }

func (Strategy) Aliases() []string {
	return []string{"aliyun", "alidns", "alibaba", "alibaba-cloud", "alicloud-cn", "alicloud-intl", "alicloud-global", "aliyun-intl"}
}

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return alidns.NewDNSProvider()
}
