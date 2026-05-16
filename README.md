# acme-go

`acme-go` is a config-driven ACME client inspired by `acme.sh`, implemented in Go and aimed at cross-platform automation.

## Scope

- Go 1.26 module and CI baseline.
- Config-driven certificate definitions.
- DNS-01 issuance flow backed by `go-acme/lego`.
- Idempotent `plan`, `issue`, and `renew` commands.
- Cross-platform release matrix for `linux`, `windows`, and `darwin` on `amd64` and `arm64`.

## Quick Start

1. Copy `config.example.yaml` to `config.yaml`.
2. Put private overrides and provider credentials into `.local.json`.
3. Run `go run . plan -config config.yaml`.
4. Run `go run . issue -config config.yaml`.

`config.yaml` stays commit-friendly, while `.local.json` has higher priority and is ignored by Git. The tool stores account material in `.acme/` and writes certificates under `certs/<name>/` by default.

## Commands

```bash
go run . plan -config config.yaml
go run . issue -config config.yaml [-name example-prod] [-force]
go run . renew -config config.yaml [-name example-prod] [-force]
go run . version
```

## Configuration Model

```yaml
ca:
  directory_url: https://acme-v02.api.letsencrypt.org/directory

account:
  email: ops@example.com
  key_path: .acme/account.pem
  accept_tos: true

dns:
  provider: cloudflare
  env:
    CLOUDFLARE_DNS_API_TOKEN: ${CLOUDFLARE_DNS_API_TOKEN}

certificates:
  - name: example-prod
    domains:
      - example.com
      - '*.example.com'
    output_dir: certs/example-prod
    key_type: ec256
    renew_before_days: 30
    challenge: dns-01
    bundle: true
```

Example local override:

```json
{
  "account": {
    "email": "ops@example.com"
  },
  "dns": {
    "provider": "alidns",
    "env": {
      "ALICLOUD_ACCESS_KEY": "your-access-key",
      "ALICLOUD_SECRET_KEY": "your-secret-key"
    }
  }
}
```

## Automated Testing

- `go test ./...` runs fast unit tests.
- `go test -tags=integration -timeout 20m ./...` runs the live ACME staging test when `.local.json` is present.
- `git-auto-up.cmd` runs unit tests first, then integration tests, and only pushes when all tests pass.

## Notes

- This starter focuses on `dns-01` because that is the most common automation path for wildcard certificates and is closest to typical `acme.sh` usage.
- DNS providers come from `go-acme/lego`, so the exact environment variables depend on the selected provider.
- For AliDNS, `lego` expects `ALICLOUD_ACCESS_KEY` and `ALICLOUD_SECRET_KEY`.
- `plan` inspects local certificate expiry and shows whether each entry would be issued or skipped.
- `issue` and `renew` both behave safely by default: if the local certificate is still outside the renewal window, it is skipped unless `-force` is provided.
