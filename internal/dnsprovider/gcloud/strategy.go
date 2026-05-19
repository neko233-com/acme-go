package gcloud

import (
	"github.com/neko233-com/acme-go/internal/config"

	"github.com/go-acme/lego/v4/challenge"
	googleprovider "github.com/go-acme/lego/v4/providers/dns/gcloud"
)

type Strategy struct{}

func (Strategy) ID() string { return "gcloud" }

func (Strategy) Aliases() []string {
	return []string{"gcp", "googlecloud", "google-cloud", "google", "gcloud-intl", "gcloud-global"}
}

func (Strategy) NewProvider(_ config.DNSConfig) (challenge.Provider, error) {
	return googleprovider.NewDNSProvider()
}
