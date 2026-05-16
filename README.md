# acme-go

`acme-go` is a config-driven ACME client inspired by `acme.sh`, implemented in Go and structured around vendor strategies.

## Current Direction

- Strategy pattern for DNS vendors, with one vendor per subdirectory under `internal/dnsprovider/`.
- First-class vendor strategies for AliCloud, Cloudflare, Volcengine, Tencent Cloud, Azure DNS, Google Cloud DNS, and AWS Route53.
- Config-driven ACME flows with `.local.json` override support.
- DNS-01, HTTP-01 standalone, HTTP webroot, and TLS-ALPN-01 issuance and renewal, plus CSR-based issuance, certificate listing, metadata inspection, revoke, install-cert, deploy, and hook support.
- Cross-platform build and CI targets for `linux`, `windows`, and `darwin` on `amd64` and `arm64`.

## Vendor Strategy Layout

```text
internal/dnsprovider/
  alicloud/
  aws/
  azure/
  cloudflare/
  gcloud/
  tencentcloud/
  volcengine/
```

Each strategy owns vendor-specific provider construction. The Volcengine strategy includes a local DNS challenge provider built on the Volcengine Go SDK.

## Quick Start

1. Copy `config.example.yaml` to `config.yaml`.
2. Put private overrides and provider credentials into `.local.json`.
3. Run `go run . providers` to confirm supported vendors.
4. Run `go run . plan -config config.yaml`.
5. Run `go run . issue -config config.yaml`.

`config.yaml` stays commit-friendly, while `.local.json` has higher priority and is ignored by Git.

## Commands

```bash
go run . providers
go run . plan -config config.yaml
go run . list -config config.yaml
go run . info -config config.yaml -name example-prod
go run . issue -config config.yaml [-name example-prod] [-force]
go run . renew -config config.yaml [-name example-prod] [-force]
go run . revoke -config config.yaml -name example-prod
go run . install-cert -config config.yaml -name example-prod
go run . deploy -config config.yaml -name example-prod
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
  provider: alicloud
  env:
    ALICLOUD_ACCESS_KEY: ${ALICLOUD_ACCESS_KEY}
    ALICLOUD_SECRET_KEY: ${ALICLOUD_SECRET_KEY}
  disable_cname_support: false
  disable_complete_propagation: false
  recursive_nameservers: []

certificates:
  - name: example-prod
    domains:
      - example.com
      - '*.example.com'
    output_dir: certs/example-prod
    key_type: ec256
    preferred_chain: ""
    must_staple: false
    csr_path: ""
    renew_before_days: 30
    challenge: dns-01
    webroot_path: ""
    http_bind: 0.0.0.0
    http_port: "80"
    tlsalpn_bind: 0.0.0.0
    tlsalpn_port: "443"
    install:
      key_file: /etc/nginx/ssl/example/privkey.pem
      fullchain_file: /etc/nginx/ssl/example/fullchain.pem
    deploy:
      - name: edge-copy
        type: copy
        directory: deploy/example
      - name: reload-nginx
        type: command
        command: nginx -s reload
    hooks:
      pre_issue: []
      post_issue: []
      pre_renew: []
      post_renew: []
      post_install: []
      post_deploy: []
      post_revoke: []
    bundle: true
```

Challenge selection summary:

- `dns-01`: uses the configured DNS vendor strategy.
- `http-01` or `standalone`: starts a local HTTP listener on `http_bind:http_port`.
- `webroot`: writes challenge files under `webroot_path`.
- `tls-alpn-01`: starts a local TLS-ALPN listener on `tlsalpn_bind:tlsalpn_port`.

`install` copies generated files into fixed destinations. `deploy` supports `copy` and `command` targets. `hooks` run shell commands around issue, renew, deploy, install, and revoke flows.

Example local override for AliCloud:

```json
{
  "dns": {
    "provider": "alicloud",
    "env": {
      "ALICLOUD_ACCESS_KEY": "your-access-key",
      "ALICLOUD_SECRET_KEY": "your-secret-key"
    }
  }
}
```

Example local override for Volcengine:

```json
{
  "dns": {
    "provider": "volcengine",
    "env": {
      "VOLC_ACCESSKEY": "your-access-key",
      "VOLC_SECRETKEY": "your-secret-key",
      "VOLC_REGION": "cn-beijing"
    }
  }
}
```

## Supported Vendors

- `alicloud`: aliases `aliyun`, `alidns`
- `cloudflare`
- `volcengine`
- `tencentcloud`: aliases `tencent`, `dnspod`
- `azure`
- `gcloud`: aliases `gcp`, `google cloud`
- `aws`: aliases `route53`

## Automated Testing

- `go test ./...` covers config, provider registry, challenge/post-action helpers, and core certificate helpers.
- `go test -tags=integration -timeout 20m ./...` runs the live ACME staging flow when `.local.json` is present.
- `test-auto.cmd` and `test-auto.sh` are the cross-platform test entrypoints used by local automation and CI.
- `git-auto-up.cmd` and `git-auto-up.sh` run the scripted test chain first, then push only when all checks pass.

## Spec-Driven

- Machine-readable specs live under `specs/`.
- `specs/provider_aliases.json` drives vendor alias resolution tests.
- `specs/config_cases.json` drives config loading and `.local.json` override tests.
- `specs/command_dispatch.json` drives CLI dispatch tests.
- `internal/testspec` provides the shared spec loader used by the Go tests.

## Notes

- This codebase is moving toward broad `acme.sh` parity, and the implemented surface today includes DNS and local challenge modes plus config-driven install/deploy/hook flows.
- `issue` supports direct domain issuance and CSR-based issuance through `csr_path`.
- `renew` prefers ACME renew when local certificate material exists, and falls back to obtain when it does not.
- `list` and `info` are metadata-backed and report certificate validity from local files.
- For `neko233.com` integration, wildcard CNAME behavior requires `disable_cname_support` and `disable_complete_propagation` in the integration config.
