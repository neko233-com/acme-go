package acme

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/neko233-com/acme-go/internal/config"
	"github.com/neko233-com/acme-go/internal/deploy"
	"github.com/neko233-com/acme-go/internal/hook"
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
	paths := cert.Paths()
	return deploy.Context{
		Name:         cert.Name,
		OutputDir:    cert.OutputDir,
		CertFile:     paths.CertFile,
		KeyFile:      paths.KeyFile,
		PublicKey:    paths.PublicKeyFile,
		FullChain:    paths.FullChainFile,
		ChainFile:    paths.ChainFile,
		MetadataFile: paths.MetadataFile,
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
	paths := cert.Paths()
	for _, path := range []string{paths.CertFile, paths.KeyFile, paths.PublicKeyFile, paths.FullChainFile, paths.ChainFile, paths.MetadataFile} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("required local certificate file %s: %w", path, err)
		}
	}
	return nil
}
