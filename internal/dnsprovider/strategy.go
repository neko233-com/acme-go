package dnsprovider

import (
	"fmt"
	"sort"
	"strings"

	"acme-go/internal/config"
	"acme-go/internal/dnsprovider/alicloud"
	"acme-go/internal/dnsprovider/aws"
	"acme-go/internal/dnsprovider/azure"
	"acme-go/internal/dnsprovider/cloudflare"
	"acme-go/internal/dnsprovider/gcloud"
	"acme-go/internal/dnsprovider/tencentcloud"
	"acme-go/internal/dnsprovider/volcengine"

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
		cloudflare.Strategy{},
		gcloud.Strategy{},
		tencentcloud.Strategy{},
		volcengine.Strategy{},
	}
}
