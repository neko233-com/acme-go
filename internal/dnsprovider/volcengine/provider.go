package volcengine

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/challenge/dns01"
	volcdns "github.com/volcengine/volc-sdk-golang/service/dns"
)

const (
	defaultTTL             = int64(120)
	defaultPropagationTime = 10 * time.Minute
	defaultPollingInterval = 15 * time.Second
	defaultRecordLine      = "default"
	defaultRegion          = "cn-beijing"
	envAccessKey           = "VOLC_ACCESSKEY"
	envSecretKey           = "VOLC_SECRETKEY"
	envRegion              = "VOLC_REGION"
	envTTL                 = "VOLC_TTL"
	envLine                = "VOLC_LINE"
	envPropagationTimeout  = "VOLC_PROPAGATION_TIMEOUT"
	envPollingInterval     = "VOLC_POLLING_INTERVAL"
)

type DNSProvider struct {
	client             *volcdns.Client
	ttl                int64
	line               string
	propagationTimeout time.Duration
	pollingInterval    time.Duration
}

func NewDNSProvider() (*DNSProvider, error) {
	caller := volcdns.NewVolcCaller()
	accessKey := os.Getenv(envAccessKey)
	secretKey := os.Getenv(envSecretKey)
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("volcengine requires %s and %s", envAccessKey, envSecretKey)
	}
	caller.Volc.SetAccessKey(accessKey)
	caller.Volc.SetSecretKey(secretKey)
	caller.Volc.ServiceInfo.Credentials.Region = envOrDefault(envRegion, defaultRegion)

	ttl, err := parseInt64Env(envTTL, defaultTTL)
	if err != nil {
		return nil, err
	}
	propagationTimeout, err := parseSecondsEnv(envPropagationTimeout, defaultPropagationTime)
	if err != nil {
		return nil, err
	}
	pollingInterval, err := parseSecondsEnv(envPollingInterval, defaultPollingInterval)
	if err != nil {
		return nil, err
	}

	return &DNSProvider{
		client:             volcdns.NewClient(caller),
		ttl:                ttl,
		line:               envOrDefault(envLine, defaultRecordLine),
		propagationTimeout: propagationTimeout,
		pollingInterval:    pollingInterval,
	}, nil
}

func (p *DNSProvider) Present(domain, token, keyAuth string) error {
	ctx := context.Background()
	zoneName, zoneID, host, value, err := p.resolveRecord(ctx, domain, keyAuth)
	if err != nil {
		return err
	}
	if err := p.deleteMatchingRecords(ctx, zoneID, host, value); err != nil {
		return err
	}
	recordType := "TXT"
	resp, err := p.client.CreateRecord(ctx, &volcdns.CreateRecordRequest{
		ZID:   &zoneID,
		Host:  &host,
		Type:  &recordType,
		Value: &value,
		TTL:   &p.ttl,
		Line:  &p.line,
	})
	if err != nil {
		return fmt.Errorf("create volcengine TXT record for zone %s: %w", zoneName, err)
	}
	if resp == nil || resp.RecordID == nil || *resp.RecordID == "" {
		return fmt.Errorf("create volcengine TXT record for zone %s returned no record id", zoneName)
	}
	return nil
}

func (p *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	ctx := context.Background()
	_, zoneID, host, value, err := p.resolveRecord(ctx, domain, keyAuth)
	if err != nil {
		return err
	}
	return p.deleteMatchingRecords(ctx, zoneID, host, value)
}

func (p *DNSProvider) Timeout() (time.Duration, time.Duration) {
	return p.propagationTimeout, p.pollingInterval
}

func (p *DNSProvider) resolveRecord(ctx context.Context, domain, keyAuth string) (string, int64, string, string, error) {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	fqdn := dns01.UnFqdn(info.EffectiveFQDN)
	zoneName, zoneID, err := p.findZone(ctx, fqdn)
	if err != nil {
		return "", 0, "", "", err
	}
	host := strings.TrimSuffix(fqdn, "."+zoneName)
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		host = "@"
	}
	return zoneName, zoneID, host, info.Value, nil
}

func (p *DNSProvider) findZone(ctx context.Context, fqdn string) (string, int64, error) {
	labels := strings.Split(fqdn, ".")
	searchMode := "accurate"
	pageNumber := "1"
	pageSize := "50"
	for index := 0; index < len(labels)-1; index++ {
		candidate := strings.Join(labels[index:], ".")
		resp, err := p.client.ListZones(ctx, &volcdns.ListZonesRequest{
			Key:        &candidate,
			SearchMode: &searchMode,
			PageNumber: &pageNumber,
			PageSize:   &pageSize,
		})
		if err != nil {
			return "", 0, fmt.Errorf("list volcengine zones for %s: %w", candidate, err)
		}
		for _, zone := range resp.Zones {
			if zone.ZoneName == nil || zone.ZID == nil {
				continue
			}
			if strings.EqualFold(strings.TrimSuffix(*zone.ZoneName, "."), candidate) {
				return strings.TrimSuffix(*zone.ZoneName, "."), *zone.ZID, nil
			}
		}
	}
	return "", 0, fmt.Errorf("volcengine zone not found for %s", fqdn)
}

func (p *DNSProvider) deleteMatchingRecords(ctx context.Context, zoneID int64, host, value string) error {
	zoneIDValue := strconv.FormatInt(zoneID, 10)
	recordType := "TXT"
	pageNumber := "1"
	pageSize := "100"
	searchMode := "accurate"
	resp, err := p.client.ListRecords(ctx, &volcdns.ListRecordsRequest{
		ZID:        &zoneIDValue,
		Host:       &host,
		Type:       &recordType,
		PageNumber: &pageNumber,
		PageSize:   &pageSize,
		SearchMode: &searchMode,
	})
	if err != nil {
		return fmt.Errorf("list volcengine records for %s: %w", host, err)
	}
	for _, record := range resp.Records {
		if record.RecordID == nil || record.Value == nil {
			continue
		}
		if *record.Value != value {
			continue
		}
		if err := p.client.DeleteRecord(ctx, &volcdns.DeleteRecordRequest{RecordID: record.RecordID}); err != nil {
			return fmt.Errorf("delete volcengine record %s: %w", *record.RecordID, err)
		}
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func parseInt64Env(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func parseSecondsEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return time.Duration(seconds) * time.Second, nil
}
