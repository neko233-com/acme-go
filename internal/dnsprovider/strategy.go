package dnsprovider

import (
	"fmt"
	"sort"
	"strings"

	"github.com/neko233-com/acme-go/internal/config"
	"github.com/neko233-com/acme-go/internal/dnsprovider/alicloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/aws"
	"github.com/neko233-com/acme-go/internal/dnsprovider/azure"
	"github.com/neko233-com/acme-go/internal/dnsprovider/baiducloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/cloudflare"
	"github.com/neko233-com/acme-go/internal/dnsprovider/digitalocean"
	"github.com/neko233-com/acme-go/internal/dnsprovider/gcloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/hetzner"
	"github.com/neko233-com/acme-go/internal/dnsprovider/huaweicloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/ibmcloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/linode"
	"github.com/neko233-com/acme-go/internal/dnsprovider/oraclecloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/scaleway"
	"github.com/neko233-com/acme-go/internal/dnsprovider/tencentcloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/ucloud"
	"github.com/neko233-com/acme-go/internal/dnsprovider/volcengine"
	"github.com/neko233-com/acme-go/internal/dnsprovider/vultr"

	"github.com/go-acme/lego/v4/challenge"
)

type Strategy interface {
	ID() string
	Aliases() []string
	NewProvider(cfg config.DNSConfig) (challenge.Provider, error)
}

func Resolve(name string) (Strategy, error) {
	normalized := normalize(name)
	for _, strategy := range supportedStrategies() {
		if normalized == normalize(strategy.ID()) {
			return strategy, nil
		}
		for _, alias := range strategy.Aliases() {
			if normalized == normalize(alias) {
				return strategy, nil
			}
		}
	}
	return nil, fmt.Errorf("unsupported dns provider %q", name)
}

func Supported() []string {
	providers := make([]string, 0, len(supportedStrategies()))
	for _, strategy := range supportedStrategies() {
		providers = append(providers, strategy.ID())
	}
	sort.Strings(providers)
	return providers
}

func normalize(value string) string {
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(value)))
}

func supportedStrategies() []Strategy {
	return []Strategy{
		alicloud.Strategy{},
		aws.Strategy{},
		azure.Strategy{},
		baiducloud.Strategy{},
		cloudflare.Strategy{},
		digitalocean.Strategy{},
		gcloud.Strategy{},
		hetzner.Strategy{},
		huaweicloud.Strategy{},
		ibmcloud.Strategy{},
		linode.Strategy{},
		oraclecloud.Strategy{},
		scaleway.Strategy{},
		tencentcloud.Strategy{},
		ucloud.Strategy{},
		vultr.Strategy{},
		volcengine.Strategy{},
	}
}
