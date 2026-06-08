# acme233

`acme233` is a config-driven ACME automation CLI written in Go. It can issue, renew, revoke, install, deploy, and upgrade certificate workflows with multi-provider DNS support.

[中文](./README.md) | [HTML guide](./how-to-use.html)

## Install

Linux / macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/neko233-com/acme233/main/install.sh | sh
```

Windows PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/neko233-com/acme233/main/install.ps1 | iex"
```

After installation, use the `acme233` command:

```bash
acme233 version
acme233 providers
```

Default install locations:

- Linux / macOS: `~/.local/bin/acme233`
- Windows: `%LOCALAPPDATA%\acme233\bin\acme233.exe`

The installers add the install directory to PATH. To install a specific version or directory:

```bash
curl -fsSL https://raw.githubusercontent.com/neko233-com/acme233/main/install.sh | sh -s -- --version v0.0.7
curl -fsSL https://raw.githubusercontent.com/neko233-com/acme233/main/install.sh | sh -s -- --install-dir /usr/local/bin
```

```powershell
$env:ACME233_VERSION = "v0.0.7"
$env:ACME233_INSTALL_DIR = "$env:USERPROFILE\bin"
powershell -ExecutionPolicy Bypass -c "irm https://raw.githubusercontent.com/neko233-com/acme233/main/install.ps1 | iex"
```

Useful environment variables:

- `ACME233_VERSION`: release tag to install. Defaults to the latest release.
- `ACME233_INSTALL_DIR`: installation directory.
- `ACME233_REPO`: GitHub repository. Defaults to `neko233-com/acme233`.
- `GITHUB_BASE_URL`: GitHub base URL. Defaults to `https://github.com`.
- `GITHUB_TOKEN`: optional token for authenticated release downloads.
- `ACME233_AUTO_UPDATE=false`: disables runtime auto-update checks.

## Features

- Supports `dns-01`, `http-01`, `standalone`, `webroot`, and `tls-alpn-01`.
- Supports common DNS providers including AliCloud, Tencent Cloud, Volcengine, Cloudflare, AWS, Google Cloud, Azure, DigitalOcean, Hetzner, Huawei Cloud, IBM Cloud, Linode, Oracle Cloud, Scaleway, UCloud, Baidu Cloud, and Vultr.
- Keeps public config in `config_acme.json` and local secrets in `config_acme.local.json`.
- Maps compact `dns.credentials` into provider-specific lego environment variables.
- Provides `issue`, `renew`, `auto-renew`, `revoke`, `install-cert`, `deploy`, `version`, and `upgrade`.
- Publishes Linux, Windows, and macOS binaries for amd64 and arm64.
- Can be embedded as a Go package: `github.com/neko233-com/acme233/pkg/acmego`.

## Quick Start

1. Edit `config_acme.json` with account, domains, challenge settings, and output paths.
2. Put DNS credentials in `config_acme.local.json`; do not commit it.
3. Run `acme233 validate -config config_acme.json`.
4. Run `acme233 providers` to list supported DNS providers.
5. Run `acme233 plan -config config_acme.json`.
6. Run `acme233 issue -config config_acme.json`.

Common commands:

```bash
acme233 validate -config config_acme.json
acme233 paths -config config_acme.json -name example-prod
acme233 plan -config config_acme.json
acme233 issue -config config_acme.json -name example-prod
acme233 renew -config config_acme.json -name example-prod
acme233 auto-renew -config config_acme.json -interval 24h
acme233 install-cert -config config_acme.json -name example-prod
acme233 deploy -config config_acme.json -name example-prod
acme233 upgrade
```

## Configuration

Use `config_acme.json` for committed settings and `config_acme.local.json` for private credentials and local overrides.

Minimal example:

```json
{
  "account": {
    "email": "ops@example.com",
    "accept_tos": true
  },
  "dns": {
    "provider": "cloudflare",
    "credentials": {
      "api_token": "put-secret-in-config_acme.local.json"
    }
  },
  "certificates": [
    {
      "name": "example-prod",
      "domains": ["example.com", "*.example.com"],
      "challenge": "dns-01",
      "output_dir": "output_SSL/example.com"
    }
  ]
}
```

More references:

- [config.schema.json](./config.schema.json)
- [config_acme.local.example.json](./config_acme.local.example.json)
- [HTML guide](./how-to-use.html)

## Renewal Service

Linux systemd:

```bash
chmod +x install-systemd-service.sh
sudo ./install-systemd-service.sh acme233-auto-renew /etc/acme233/config_acme.json /usr/local/bin/acme233
```

Example unit:

```text
examples/systemd/acme233-auto-renew.service
```

Windows service through `nssm`:

```cmd
install-windows-service.cmd acme233-auto-renew C:\acme233\config_acme.json C:\acme233\acme233.exe
```

## Go Package

Other Go services can depend on:

```bash
go get github.com/neko233-com/acme233/pkg/acmego
```

Example:

```go
package ssl

import (
  "io"

  "github.com/neko233-com/acme233/pkg/acmego"
)

func IssueSiteCertificate() (acmego.CertificatePaths, error) {
  result, err := acmego.IssueCertificate(acmego.Request{
    Email:    "ops@example.com",
    Provider: "cloudflare",
    Domains:  []string{"example.com", "*.example.com"},
    OutputDir: "/etc/myapp/ssl/example.com",
    Credentials: acmego.DNSCredentials{
      APIToken: "cloudflare-token-from-secret-store",
    },
    Force: true,
  }, io.Discard)
  if err != nil {
    return acmego.CertificatePaths{}, err
  }
  return result.Paths, nil
}
```

## GitHub Release

The release workflow builds zip assets for:

- `acme233_linux_amd64.zip`
- `acme233_linux_arm64.zip`
- `acme233_windows_amd64.zip`
- `acme233_windows_arm64.zip`
- `acme233_darwin_amd64.zip`
- `acme233_darwin_arm64.zip`

Release by pushing a tag:

```bash
git tag -a v0.0.7 -m "release v0.0.7"
git push origin v0.0.7
```

GitHub Actions will build assets and create the release. You can also use `gh release create` to upload local assets manually.

## Development

```bash
go test ./...
go test -tags=integration -timeout 20m ./...
```

Integration tests require `config_acme.local.json`; regular tests do not require a live ACME account.
