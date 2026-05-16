package acme

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"acme-go/internal/config"
	"acme-go/internal/deploy"
	"acme-go/internal/hook"
)

func runPreHooks(cert config.CertificateSpec, mode Mode, ctx deploy.Context, out io.Writer) error {
	commands := cert.Hooks.PreIssue
	if mode == ModeRenew {
		commands = cert.Hooks.PreRenew
	}
	return hook.Run(commands, deployEnv(ctx), out)
}

func runSuccessActions(cfg *config.Config, cert config.CertificateSpec, mode Mode, out io.Writer) error {
	ctx := newDeployContext(cfg, cert)
	if err := deploy.Install(cert.Install, ctx); err != nil {
		return err
	}
	if err := hook.Run(cert.Hooks.PostInstall, deployEnv(ctx), out); err != nil {
		return err
	}
	if err := deploy.RunTargets(cert.Deploy, ctx, out); err != nil {
		return err
	}
	if err := hook.Run(cert.Hooks.PostDeploy, deployEnv(ctx), out); err != nil {
		return err
	}
	commands := cert.Hooks.PostIssue
	if mode == ModeRenew {
		commands = cert.Hooks.PostRenew
	}
	return hook.Run(commands, deployEnv(ctx), out)
}

func runRevokeHooks(cfg *config.Config, cert config.CertificateSpec, out io.Writer) error {
	return hook.Run(cert.Hooks.PostRevoke, deployEnv(newDeployContext(cfg, cert)), out)
}

func installExisting(cfg *config.Config, cert config.CertificateSpec, out io.Writer) error {
	ctx := newDeployContext(cfg, cert)
	if err := deploy.Install(cert.Install, ctx); err != nil {
		return err
	}
	return hook.Run(cert.Hooks.PostInstall, deployEnv(ctx), out)
}

func deployExisting(cfg *config.Config, cert config.CertificateSpec, out io.Writer) error {
	ctx := newDeployContext(cfg, cert)
	if err := deploy.RunTargets(cert.Deploy, ctx, out); err != nil {
		return err
	}
	return hook.Run(cert.Hooks.PostDeploy, deployEnv(ctx), out)
}

func newDeployContext(cfg *config.Config, cert config.CertificateSpec) deploy.Context {
	return deploy.Context{
		Name:         cert.Name,
		OutputDir:    cert.OutputDir,
		CertFile:     filepath.Join(cert.OutputDir, "cert.pem"),
		KeyFile:      filepath.Join(cert.OutputDir, "privkey.pem"),
		FullChain:    filepath.Join(cert.OutputDir, "fullchain.pem"),
		ChainFile:    filepath.Join(cert.OutputDir, "issuer.pem"),
		Provider:     cfg.DNS.Provider,
		Challenge:    cert.Challenge,
		Domain:       cert.Domains[0],
		DomainsCSV:   strings.Join(cert.Domains, ","),
		DirectoryURL: cfg.CA.DirectoryURL,
	}
}

func deployEnv(ctx deploy.Context) map[string]string {
	return deploy.BuildEnv(ctx)
}

func validateLocalCertificateMaterial(cert config.CertificateSpec) error {
	for _, name := range []string{"cert.pem", "privkey.pem", "fullchain.pem", "issuer.pem"} {
		path := filepath.Join(cert.OutputDir, name)
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("required local certificate file %s: %w", path, err)
		}
	}
	return nil
}
