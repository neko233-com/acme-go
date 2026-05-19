# acme-go

Config-driven ACME automation in Go, inspired by acme.sh.

Go 语言实现的配置驱动 ACME 自动化工具，设计目标参考 acme.sh。

- English HTML guide: [how-to-use.html](./how-to-use.html)
- 中文 HTML 使用说明: [how-to-use.html](./how-to-use.html)

## Table of Contents | 目录

- [Overview | 项目概览](#overview--项目概览)
- [Features | 功能特性](#features--功能特性)
- [Quick Start | 快速开始](#quick-start--快速开始)
- [Go Library Usage | Go 二方库接入](#go-library-usage--go-二方库接入)
- [Commands | 命令说明](#commands--命令说明)
- [Challenge Modes | Challenge 模式](#challenge-modes--challenge-模式)
- [Configuration | 配置模型](#configuration--配置模型)
- [Deploy Install Hooks | 安装部署与钩子](#deploy-install-hooks--安装部署与钩子)
- [Supported Vendors | 支持的 DNS 厂商](#supported-vendors--支持的-dns-厂商)
- [Testing and Spec | 测试与规范](#testing-and-spec--测试与规范)
- [GitHub Actions Release | GitHub Actions 打包发布](#github-actions-release--github-actions-打包发布)
- [Library Publishing | 库版本发布](#library-publishing--库版本发布)
- [VS Code Actions Loop | VS Code Actions 反复验证](#vs-code-actions-loop--vs-code-actions-反复验证)
- [Repository Layout | 仓库结构](#repository-layout--仓库结构)
- [Notes | 说明](#notes--说明)

## Overview | 项目概览

`acme-go` is a config-driven ACME client built around vendor strategies, local automation, and release-friendly packaging.

`acme-go` 是一个面向配置驱动、厂商策略模式、本地自动化以及可发布产物打包的 ACME 客户端。

It currently focuses on the most common operational surface used in real deployments: issue, renew, revoke, install, deploy, hooks, and multi-provider DNS integration.

当前实现重点覆盖真实使用场景里最常见的操作面：签发、续期、吊销、安装证书、部署、钩子，以及多厂商 DNS 集成。

## Features | 功能特性

- DNS vendor strategy layout under `internal/dnsprovider/`, with one provider per subdirectory.
- Challenge support for `dns-01`, `http-01`, `standalone`, `webroot`, and `tls-alpn-01`.
- Lifecycle commands for `help`, `doc`, `validate`, `paths`, `plan`, `issue`, `renew`, `revoke`, `list`, `info`, `install-cert`, `deploy`, `version`, and `upgrade`.
- Config merge with `.local.json` override so secrets stay out of Git.
- Low-config DNS credentials mapping: use `dns.credentials` for common vendors and let acme-go derive vendor env vars.
- Domestic and international provider aliases such as `alicloud-cn`, `alicloud-intl`, `volcengine-cn`, `volcengine-intl`, `tencentcloud-cn`, and `tencentcloud-intl`.
- Config-driven install, deploy, and hook pipeline.
- Automatic self-update checks are enabled by default for operational commands, with a local cooldown cache to avoid repeated GitHub checks.
- GitHub Actions CI and tag-based release packaging for Linux, Windows, and macOS on `amd64` and `arm64`.
- Importable Go package for other services, such as nginx GUI servers that need automatic free SSL renewal.
- Fully configurable generated certificate file names under `output_dir`, including `.crt`, `.key`, `.pub`, chain, fullchain, and metadata files.

- DNS 厂商策略目录位于 `internal/dnsprovider/`，每个厂商一个独立子目录。
- 支持 `dns-01`、`http-01`、`standalone`、`webroot`、`tls-alpn-01` 五类 challenge。
- 支持 `help`、`doc`、`validate`、`paths`、`plan`、`issue`、`renew`、`revoke`、`list`、`info`、`install-cert`、`deploy`、`version`、`upgrade` 等生命周期命令。
- 支持 `.local.json` 覆盖配置，便于将敏感信息与 Git 隔离。
- 支持低配置 DNS 凭据模型，优先填写 `dns.credentials`，由 acme-go 自动映射到各厂商需要的环境变量。
- 支持国内版/国际版 DNS 厂商别名，例如 `alicloud-cn`、`alicloud-intl`、`volcengine-cn`、`volcengine-intl`、`tencentcloud-cn`、`tencentcloud-intl`。
- 支持配置驱动的 install、deploy、hook 流程。
- 默认对常用操作命令开启自动自升级检查，并带有本地冷却缓存，避免频繁请求 GitHub。
- 支持 GitHub Actions 持续集成与基于 tag 的 Linux、Windows、macOS 多架构打包发布。
- 支持作为 Go 二方库被其他服务接入，例如让 nginx GUI 服务器获得自动免费续期 SSL 的能力。
- 支持完整配置 `output_dir` 下的产物文件名，包括 `.crt`、`.key`、`.pub`、chain、fullchain、metadata 等文件。

## Quick Start | 快速开始

1. Copy `config.example.yaml` to `config.yaml`.
2. Put private credentials into `.local.json`.
3. Run `go run . validate -config config.yaml`.
4. Run `go run . paths -config config.yaml` to inspect resolved certificate file paths.
5. Run `go run . providers` to inspect supported DNS vendors.
6. Run `go run . plan -config config.yaml`.
7. Run `go run . issue -config config.yaml`.

1. 复制 `config.example.yaml` 为 `config.yaml`。
2. 将私密凭据放入 `.local.json`。
3. 运行 `go run . validate -config config.yaml`。
4. 运行 `go run . paths -config config.yaml` 查看解析后的证书文件路径。
5. 运行 `go run . providers` 查看支持的 DNS 厂商。
6. 运行 `go run . plan -config config.yaml`。
7. 运行 `go run . issue -config config.yaml`。

`config.yaml` stays commit-friendly, while `.local.json` has higher priority and should remain uncommitted.

`config.yaml` 适合提交到仓库，`.local.json` 优先级更高，建议始终不提交。

## Go Library Usage | Go 二方库接入

Other Go services can depend on this repository directly:

其他 Go 服务可以直接依赖本仓库：

```bash
go get github.com/neko233-com/acme-go/pkg/acmego
```

Minimal nginx GUI integration example:

nginx GUI 服务接入示例：

```go
package ssl

import (
  "io"

  "github.com/neko233-com/acme-go/pkg/acmego"
)

func RenewNginxCertificate() error {
  cfg := &acmego.Config{
    CA: acmego.CAConfig{DirectoryURL: "https://acme-v02.api.letsencrypt.org/directory"},
    Account: acmego.AccountConfig{
      Email:     "ops@example.com",
      KeyPath:   "/var/lib/nginx-gui/acme/account.pem",
      AcceptTOS: true,
    },
    DNS: acmego.DNSConfig{
      Provider: "cloudflare",
      Env: map[string]string{
        "CLOUDFLARE_DNS_API_TOKEN": "token-from-your-secret-store",
      },
    },
    Certificates: []acmego.CertificateSpec{
      {
        Name:      "site-a",
        Domains:   []string{"example.com", "*.example.com"},
        OutputDir: "/etc/nginx/ssl/site-a",
        OutputFiles: acmego.CertificateFiles{
          FullChainFile: "server.crt",
          KeyFile:       "server.key",
          PublicKeyFile: "server.pub",
          CertFile:      "leaf.crt",
          ChainFile:     "ca.crt",
          MetadataFile:  "acme.json",
        },
        KeyType:         "ec256",
        RenewBeforeDays: 30,
        Challenge:       "dns-01",
      },
    },
  }

  _, err := acmego.Renew(cfg, "site-a", false, io.Discard)
  return err
}
```

`OutputDir` controls the directory. `OutputFiles` controls generated file names. Relative file names are resolved under `OutputDir`; absolute file paths are also accepted. Suggested nginx names are `server.crt` for fullchain, `server.key` for the private key, `server.pub` for the public key, `leaf.crt` for the leaf certificate, `ca.crt` for the issuer chain, and `acme.json` for renewal metadata.

`OutputDir` 用于指定目录，`OutputFiles` 用于指定生成文件名。相对路径会放在 `OutputDir` 下，绝对路径也可以直接使用。nginx GUI 场景建议用 `server.crt` 存 fullchain，`server.key` 存私钥，`server.pub` 存公钥，`leaf.crt` 存叶子证书，`ca.crt` 存签发链，`acme.json` 存续期元数据。

## Commands | 命令说明

```bash
go run . help [command]
go run . doc [-print-path]
go run . providers
go run . validate -config config.yaml
go run . paths -config config.yaml [-name example-prod]
go run . plan -config config.yaml
go run . list -config config.yaml
go run . info -config config.yaml -name example-prod
go run . issue -config config.yaml [-name example-prod] [-force]
go run . renew -config config.yaml [-name example-prod] [-force]
go run . revoke -config config.yaml -name example-prod
go run . install-cert -config config.yaml -name example-prod
go run . deploy -config config.yaml -name example-prod
go run . version
go run . upgrade
```

- `validate`: load config, apply defaults, merge `.local.json`, and print a short JSON summary.
- `help`: print overall usage, or command-specific help such as `acme-go help version`.
- `doc`: open `how-to-use.html` in the default browser, or use `-print-path` to print the resolved file path.
- `paths`: print resolved output paths for generated certificate material.
- `plan`: show whether a certificate entry would issue or skip.
- `issue`: obtain a new certificate or replace an existing one.
- `renew`: renew when the local certificate is near expiry or when `-force` is passed.
- `revoke`: revoke the local certificate using the ACME server.
- `install-cert`: copy local certificate files into configured install destinations.
- `deploy`: run configured deploy targets using existing local certificate material.
- `version`: print the current version, whether a newer GitHub release exists, and a concise changelog summary.
- `upgrade`: download the latest matching GitHub release artifact for the current OS and architecture and replace the local binary.
- Automatic update checks are enabled by default before operational commands. Set `ACME_GO_AUTO_UPDATE=false` to disable them.

- `validate`：加载配置、应用默认值、合并 `.local.json`，并输出简短 JSON 摘要。
- `help`：输出总帮助，或按命令查看帮助，例如 `acme-go help version`。
- `doc`：在默认浏览器中打开 `how-to-use.html`，也可以通过 `-print-path` 只输出最终解析到的文件路径。
- `paths`：输出证书产物的最终解析路径。
- `plan`：展示每个证书条目是会执行签发还是跳过。
- `issue`：申请新证书，或覆盖已有本地证书。
- `renew`：当证书接近过期时续期，也可配合 `-force` 强制执行。
- `revoke`：通过 ACME 服务端吊销本地证书。
- `install-cert`：将本地证书文件复制到配置的安装目标位置。
- `deploy`：基于已有本地证书材料执行部署目标。
- `version`：输出当前版本、是否已有更新，以及较新版本的摘要差异。
- `upgrade`：自动下载当前系统架构对应的最新 GitHub Release 并替换本地二进制。
- 默认会在常用操作命令执行前自动检查并应用更新；如需关闭，可设置 `ACME_GO_AUTO_UPDATE=false`。

## Challenge Modes | Challenge 模式

- `dns-01`: uses the configured DNS vendor strategy and is the default mode.
- `http-01`: runs a local HTTP challenge server.
- `standalone`: same operational behavior as the built-in local HTTP server flow.
- `webroot`: writes ACME HTTP challenge files under `webroot_path`.
- `tls-alpn-01`: runs a local TLS-ALPN challenge server.

- `dns-01`：使用配置的 DNS 厂商策略，也是默认模式。
- `http-01`：启动本地 HTTP challenge 服务。
- `standalone`：与内置本地 HTTP 服务模式保持相同运行行为。
- `webroot`：将 ACME HTTP challenge 文件写入 `webroot_path`。
- `tls-alpn-01`：启动本地 TLS-ALPN challenge 服务。

## Configuration | 配置模型

Example:

示例：

```yaml
ca:
  directory_url: https://acme-v02.api.letsencrypt.org/directory

account:
  email: ops@example.com
  key_path: .acme/account.pem
  accept_tos: true

dns:
  provider: alicloud-cn
  account_mode: cn
  credentials:
    access_key: ${ALICLOUD_ACCESS_KEY}
    secret_key: ${ALICLOUD_SECRET_KEY}
  env: {}
  disable_cname_support: false
  disable_complete_propagation: false
  recursive_nameservers: []

For common vendors, prefer `dns.credentials` over manually writing `dns.env`. acme-go derives the provider env vars for AliCloud, Volcengine, Tencent Cloud, Cloudflare, AWS, Google Cloud, and Azure. `dns.env` still works and overrides the derived values when you need vendor-specific tuning.

常见厂商建议优先填写 `dns.credentials`，不要手写 `dns.env`。acme-go 会自动为阿里云、火山引擎、腾讯云、Cloudflare、AWS、Google Cloud、Azure 推导所需环境变量。若你需要厂商特定参数，`dns.env` 仍然可用，且优先级更高。

If you operate multiple regional accounts, use provider aliases such as `alicloud-cn`, `alicloud-intl`, `volcengine-cn`, `volcengine-intl`, `tencentcloud-cn`, or `tencentcloud-intl`. You can also set `dns.account_mode` to `cn`, `intl`, or `global`; for example, Volcengine defaults to `cn-beijing` for `cn` and `ap-singapore` for `intl/global` when no explicit `dns.region` is set.

如果你同时管理国内版和国际版账号，可以直接使用 `alicloud-cn`、`alicloud-intl`、`volcengine-cn`、`volcengine-intl`、`tencentcloud-cn`、`tencentcloud-intl` 这些别名，也可以配合 `dns.account_mode: cn|intl|global`。例如火山引擎在未显式设置 `dns.region` 时，会为 `cn` 默认 `cn-beijing`，为 `intl/global` 默认 `ap-singapore`。

certificates:
  - name: example-prod
    domains:
      - example.com
      - '*.example.com'
    output_dir: certs/example-prod
    output_files:
      # Leaf certificate. Suggested names: cert.pem, tls.crt, server.crt.
      cert_file: cert.pem
      # Private key. Suggested names: privkey.pem, tls.key, server.key.
      key_file: privkey.pem
      # Public key derived from the certificate. Suggested names: pubkey.pem, server.pub.
      public_key_file: pubkey.pem
      # Leaf + issuer chain. Suggested names: fullchain.pem, fullchain.crt, server.crt for nginx.
      fullchain_file: fullchain.pem
      # Issuer chain only. Suggested names: issuer.pem, chain.pem, ca.crt.
      chain_file: issuer.pem
      # Renewal metadata. Suggested names: metadata.json, acme.json.
      metadata_file: metadata.json
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
      public_key_file: /etc/nginx/ssl/example/pubkey.pem
      fullchain_file: /etc/nginx/ssl/example/fullchain.pem
      metadata_file: /etc/nginx/ssl/example/acme.json
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

Local override example for `.local.json`:

`.local.json` 本地覆盖示例：

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

`config.schema.json` is included for editor validation. VS Code maps it to `config.yaml` and `config.*.yaml` through `.vscode/settings.json` when the YAML extension is installed.

仓库内置 `config.schema.json` 用于编辑器校验。安装 YAML 扩展后，VS Code 会通过 `.vscode/settings.json` 将它关联到 `config.yaml` 与 `config.*.yaml`。

## Deploy Install Hooks | 安装部署与钩子

`install` copies generated files to fixed destinations. `deploy` supports `copy` and `command` targets. `hooks` let you run custom commands before or after key lifecycle stages.

`install` 用于把生成后的证书复制到固定目标位置。`deploy` 当前支持 `copy` 和 `command` 两类目标。`hooks` 用于在关键生命周期前后执行自定义命令。

Hook phases:

钩子阶段：

- `pre_issue`
- `post_issue`
- `pre_renew`
- `post_renew`
- `post_install`
- `post_deploy`
- `post_revoke`

Useful environment variables exposed to hooks and deploy commands include:

Hook 与 deploy 命令可用的重要环境变量包括：

- `ACME_CERT_NAME`
- `ACME_CERT_OUTPUT_DIR`
- `ACME_CERT_FILE`
- `ACME_CERT_KEY_FILE`
- `ACME_CERT_PUBLIC_KEY_FILE`
- `ACME_CERT_FULLCHAIN_FILE`
- `ACME_CERT_CHAIN_FILE`
- `ACME_CERT_METADATA_FILE`
- `ACME_PROVIDER`
- `ACME_CHALLENGE`
- `ACME_DOMAIN`
- `ACME_DOMAINS`
- `ACME_DIRECTORY_URL`
- `ACME_DEPLOY_TARGET`

## Supported Vendors | 支持的 DNS 厂商

- `alicloud`: aliases `aliyun`, `alidns`
- `cloudflare`
- `volcengine`
- `tencentcloud`: aliases `tencent`, `dnspod`
- `azure`
- `gcloud`: aliases `gcp`, `google cloud`
- `aws`: aliases `route53`

- `alicloud`：别名 `aliyun`、`alidns`
- `cloudflare`
- `volcengine`
- `tencentcloud`：别名 `tencent`、`dnspod`
- `azure`
- `gcloud`：别名 `gcp`、`google cloud`
- `aws`：别名 `route53`

## Testing and Spec | 测试与规范

- `go test ./...` covers config, provider registry, challenge helpers, deploy/install behavior, and core certificate helpers.
- `go test -tags=integration -timeout 20m ./...` runs the live ACME staging flow when `.local.json` is present.
- `test-auto.cmd` and `test-auto.sh` are the cross-platform automation entrypoints.
- `git-auto-up.cmd` and `git-auto-up.sh` run tests first, then commit and push when checks pass.
- Specs live under `specs/` and drive config loading, provider aliases, and command dispatch tests.

- `go test ./...` 覆盖配置、provider registry、challenge helper、deploy/install 行为以及证书核心辅助逻辑。
- `go test -tags=integration -timeout 20m ./...` 会在存在 `.local.json` 时执行真实 ACME staging 测试。
- `test-auto.cmd` 和 `test-auto.sh` 是跨平台自动化测试入口。
- `git-auto-up.cmd` 和 `git-auto-up.sh` 会先测试，再在全部通过后提交并推送。
- `specs/` 下保存机器可读规范，用于驱动配置加载、provider alias 和命令分发测试。

## GitHub Actions Release | GitHub Actions 打包发布

The repository includes:

仓库内置：

- `ci.yml`: runs cross-platform test automation on push and pull request.
- `release.yml`: builds release archives for `linux`, `windows`, and `darwin` on `amd64` and `arm64`, bundles docs, generates `SHA256SUMS`, and publishes assets on version tags such as `v1.0.0`.
- `publish-lib.yml`: manually publishes Go library tags from the GitHub Actions UI, with a dry-run mode for VS Code GitHub Actions extension debugging.

- `ci.yml`：在 push 与 pull request 时执行跨平台自动化测试。
- `release.yml`：为 `linux`、`windows`、`darwin` 的 `amd64` 与 `arm64` 构建发布压缩包，附带文档，生成 `SHA256SUMS`，并在 `v1.0.0` 这类版本 tag 上发布资产。
- `publish-lib.yml`：通过 GitHub Actions UI 手动发布 Go 库 tag，并提供 dry-run 模式，方便在 VS Code GitHub Actions 插件里调试。

To trigger a release:

触发发布方式：

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Library Publishing | 库版本发布

Go library publishing is tag-based. If no semantic tag exists, the first version is `v0.0.1`; later runs increment the patch version by default.

Go 库发布基于 tag。如果还没有语义化版本 tag，首个版本自动从 `v0.0.1` 开始；后续默认递增 patch 版本。

Windows:

```cmd
publish-lib.cmd
publish-lib.cmd v0.0.1
```

Linux/macOS:

```bash
chmod +x publish-lib.sh
./publish-lib.sh
./publish-lib.sh v0.0.1
```

The scripts run the non-integration test suite, verify `pkg/acmego`, create an annotated tag, and push it to GitHub. Consumers can then use:

脚本会运行非 integration 测试，验证 `pkg/acmego`，创建 annotated tag 并推送到 GitHub。其他项目随后可以使用：

```bash
go get github.com/neko233-com/acme-go/pkg/acmego@v0.0.1
```

For VS Code GitHub Actions extension debugging, open the Actions view, choose `ci` or `publish-lib`, and run the workflow manually. Keep `publish-lib` in `dry_run=true` until the computed version and tests look correct; run again with `dry_run=false` to create the tag.

使用 VS Code GitHub Actions 插件调试时，在 Actions 视图选择 `ci` 或 `publish-lib` 并手动运行。调试 `publish-lib` 时先保持 `dry_run=true`，确认版本号和测试结果无误后，再用 `dry_run=false` 创建 tag。

## VS Code Actions Loop | VS Code Actions 反复验证

This workspace recommends the official `GitHub Actions` VS Code extension through `.vscode/extensions.json`.

本工作区通过 `.vscode/extensions.json` 推荐安装官方 `GitHub Actions` VS Code 扩展。

Suggested loop:

建议循环：

1. Run local `validate: config example`, `validate: tests`, and `validate: github actions` from VS Code Tasks.
2. Commit and push the branch.
3. Open the GitHub Actions extension view.
4. Run `ci` manually with `target_os=ubuntu`, `windows`, or `macos` while debugging; use `target_os=all` before final release.
5. Run `publish-lib` with `dry_run=true` to validate version calculation and tests.
6. Re-run `publish-lib` with `dry_run=false` only when you want to create and push the library tag.

1. 先在 VS Code Tasks 里运行 `validate: config example`、`validate: tests`、`validate: github actions`。
2. 提交并 push 当前分支。
3. 打开 GitHub Actions 扩展视图。
4. 调试时手动运行 `ci`，`target_os` 选 `ubuntu`、`windows` 或 `macos`；最终发布前再选 `all`。
5. 运行 `publish-lib`，保持 `dry_run=true` 来验证版本计算和测试。
6. 只有确认要创建并推送库版本 tag 时，才用 `dry_run=false` 重新运行 `publish-lib`。

## Repository Layout | 仓库结构

```text
.
├─ .github/workflows/
├─ internal/acme/
├─ internal/config/
├─ internal/deploy/
├─ internal/dnsprovider/
├─ internal/hook/
├─ internal/testspec/
├─ specs/
├─ config.example.yaml
├─ how-to-use.html
├─ test-auto.cmd
├─ test-auto.sh
├─ publish-lib.cmd
├─ publish-lib.sh
├─ git-auto-up.cmd
└─ git-auto-up.sh
```

## Notes | 说明

- The current implementation is closer to the common operational surface of acme.sh, but not yet full parity with every ecosystem deploy plugin.
- For `neko233.com` integration, wildcard CNAME behavior may require `disable_cname_support` and `disable_complete_propagation`.
- If you want a browser-friendly entry document, open [how-to-use.html](./how-to-use.html).
- If you want a browser-friendly entry document, open [how-to-use.html](./how-to-use.html).
- You can also run `acme-go doc` to open the same guide from the CLI. Use `acme-go doc -print-path` when you only want the resolved path.

- 当前实现已经接近 acme.sh 的常用操作面，但还未覆盖其全部生态 deploy 插件。
- 对于 `neko233.com` 一类场景，通配符 CNAME 行为可能需要开启 `disable_cname_support` 与 `disable_complete_propagation`。
- 如果需要更适合浏览器查看的入口文档，可以直接打开 [how-to-use.html](./how-to-use.html)。
- 也可以直接运行 `acme-go doc` 从命令行打开同一个 HTML 文档；如果只想确认最终解析到的文档路径，可使用 `acme-go doc -print-path`。
