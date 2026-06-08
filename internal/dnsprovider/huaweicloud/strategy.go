package huaweicloud

import (
	"github.com/neko233-com/acme233/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	hwprovider "github.com/go-acme/lego/v4/providers/dns/huaweicloud"
)

type Strategy struct{}

func (Strategy) ID() string { return "huaweicloud" }

func (Strategy) Aliases() []string {
	return []string{"huawei", "huawei-cloud", "huawei-dns", "huaweicloud-cn", "huaweicloud-intl"}
}

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return hwprovider.NewDNSProvider()
}
