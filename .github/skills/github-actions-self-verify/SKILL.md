---
name: github-actions-self-verify
description: 'Use when: modifying GitHub Actions workflows, VS Code tasks, release packaging, CI, actionlint, or automation scripts in acme-go. Runs repository self-verification before commit or push.'
argument-hint: 'what changed, for example: release zip packaging or CI workflow'
---

# GitHub Actions Self Verify

Use this skill whenever an agent changes GitHub Actions, VS Code task wiring, release packaging, or validation scripts in this repository.

## Procedure

1. Inspect changed workflow and VS Code files:
   - `.github/workflows/ci.yml`
   - `.github/workflows/publish-lib.yml`
   - `.github/workflows/release.yml`
   - `.vscode/tasks.json`
   - `.vscode/settings.json`

2. Run workflow linting:

   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts/validate-github-actions.ps1
   ```

   The script runs actionlint with ShellCheck enabled. On Windows it bootstraps a project-local ShellCheck binary under `.cache/tools/` when no global `shellcheck` is available.

3. Run config validation:

   ```powershell
   go run . validate -config config_acme.json
   go run . validate -config config_acme.test.json
   ```

4. Run the Go test suite and diff check:

   ```powershell
   go test ./...
   git diff --check
   ```

5. For release packaging changes, verify each zip bundle is expected to contain:
   - `acme-go` or `acme-go.exe`
   - `README.md`
   - `how-to-use.html`
   - `_doc/`
   - `config_acme.json`
   - `config_acme.local.example.json`
   - `install-systemd-service.sh`
   - `install-windows-service.cmd`

6. Before pushing, check that local secrets are not staged:

   ```powershell
   git status --short
   git ls-files config_acme.local.json
   ```

`config_acme.local.json` must remain ignored and untracked.