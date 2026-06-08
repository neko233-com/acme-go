package acme

import (
	"fmt"
	"io"
	"os"

	"github.com/neko233-com/acme233/internal/config"
	"github.com/neko233-com/acme233/internal/deploy"
	"github.com/neko233-com/acme233/internal/hook"
)

func runPreHooks(cert config.CertificateSpec, mode Mode, ctx deploy.Context, out io.Writer) error {
	commands := cert.Hooks.PreIssue
	if mode == ModeRenew {
		commands = cert.Hooks.PreRenew
	}
	return hook.Run(commands, deployEnv(ctx), out)
}

func runSuccessActions(runtime certificateRuntime, mode Mode, out io.Writer) error {
	if err := deploy.Install(runtime.cert.Install, runtime.deployContext); err != nil {
		return err
	}
	if err := hook.Run(runtime.cert.Hooks.PostInstall, deployEnv(runtime.deployContext), out); err != nil {
		return err
	}
	if err := deploy.RunTargets(runtime.cert.Deploy, runtime.deployContext, out); err != nil {
		return err
	}
	if err := hook.Run(runtime.cert.Hooks.PostDeploy, deployEnv(runtime.deployContext), out); err != nil {
		return err
	}
	commands := runtime.cert.Hooks.PostIssue
	if mode == ModeRenew {
		commands = runtime.cert.Hooks.PostRenew
	}
	return hook.Run(commands, deployEnv(runtime.deployContext), out)
}

func runRevokeHooks(runtime certificateRuntime, out io.Writer) error {
	return hook.Run(runtime.cert.Hooks.PostRevoke, deployEnv(runtime.deployContext), out)
}

func installExisting(runtime certificateRuntime, out io.Writer) error {
	if err := deploy.Install(runtime.cert.Install, runtime.deployContext); err != nil {
		return err
	}
	return hook.Run(runtime.cert.Hooks.PostInstall, deployEnv(runtime.deployContext), out)
}

func deployExisting(runtime certificateRuntime, out io.Writer) error {
	if err := deploy.RunTargets(runtime.cert.Deploy, runtime.deployContext, out); err != nil {
		return err
	}
	return hook.Run(runtime.cert.Hooks.PostDeploy, deployEnv(runtime.deployContext), out)
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
