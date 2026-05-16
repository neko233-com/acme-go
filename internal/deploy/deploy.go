package deploy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"acme-go/internal/config"
	"acme-go/internal/hook"
)

type Context struct {
	Name         string
	OutputDir    string
	CertFile     string
	KeyFile      string
	FullChain    string
	ChainFile    string
	Directory    string
	Challenge    string
	Provider     string
	Domain       string
	DomainsCSV   string
	DirectoryURL string
}

func Install(spec config.InstallConfig, ctx Context) error {
	if spec.CertFile == "" && spec.KeyFile == "" && spec.FullChainFile == "" && spec.ChainFile == "" {
		return nil
	}
	if err := copyFileIfConfigured(ctx.CertFile, spec.CertFile); err != nil {
		return err
	}
	if err := copyFileIfConfigured(ctx.KeyFile, spec.KeyFile); err != nil {
		return err
	}
	if err := copyFileIfConfigured(ctx.FullChain, spec.FullChainFile); err != nil {
		return err
	}
	if err := copyFileIfConfigured(ctx.ChainFile, spec.ChainFile); err != nil {
		return err
	}
	return nil
}

func RunTargets(targets []config.DeployTarget, ctx Context, out io.Writer) error {
	for _, target := range targets {
		switch strings.ToLower(target.Type) {
		case "", "copy":
			if err := runCopyTarget(target, ctx); err != nil {
				return err
			}
		case "command":
			env := BuildEnv(ctx)
			for key, value := range target.Env {
				env[key] = value
			}
			env["ACME_DEPLOY_TARGET"] = target.Name
			if err := hook.Run([]string{target.Command}, env, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported deploy target type %q", target.Type)
		}
	}
	return nil
}

func BuildEnv(ctx Context) map[string]string {
	return map[string]string{
		"ACME_CERT_NAME":           ctx.Name,
		"ACME_CERT_OUTPUT_DIR":     ctx.OutputDir,
		"ACME_CERT_FILE":           ctx.CertFile,
		"ACME_CERT_KEY_FILE":       ctx.KeyFile,
		"ACME_CERT_FULLCHAIN_FILE": ctx.FullChain,
		"ACME_CERT_CHAIN_FILE":     ctx.ChainFile,
		"ACME_PROVIDER":            ctx.Provider,
		"ACME_CHALLENGE":           ctx.Challenge,
		"ACME_DOMAIN":              ctx.Domain,
		"ACME_DOMAINS":             ctx.DomainsCSV,
		"ACME_DIRECTORY_URL":       ctx.DirectoryURL,
	}
}

func runCopyTarget(target config.DeployTarget, ctx Context) error {
	if target.Directory != "" {
		if err := os.MkdirAll(target.Directory, 0o755); err != nil {
			return fmt.Errorf("create deploy directory: %w", err)
		}
	}
	certDest := target.CertFile
	keyDest := target.KeyFile
	fullChainDest := target.FullChainFile
	chainDest := target.ChainFile
	if target.Directory != "" {
		if certDest == "" {
			certDest = filepath.Join(target.Directory, "cert.pem")
		}
		if keyDest == "" {
			keyDest = filepath.Join(target.Directory, "privkey.pem")
		}
		if fullChainDest == "" {
			fullChainDest = filepath.Join(target.Directory, "fullchain.pem")
		}
		if chainDest == "" {
			chainDest = filepath.Join(target.Directory, "issuer.pem")
		}
	}
	if err := copyFileIfConfigured(ctx.CertFile, certDest); err != nil {
		return err
	}
	if err := copyFileIfConfigured(ctx.KeyFile, keyDest); err != nil {
		return err
	}
	if err := copyFileIfConfigured(ctx.FullChain, fullChainDest); err != nil {
		return err
	}
	if err := copyFileIfConfigured(ctx.ChainFile, chainDest); err != nil {
		return err
	}
	return nil
}

func copyFileIfConfigured(source, destination string) error {
	if destination == "" {
		return nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source file %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create destination dir for %s: %w", destination, err)
	}
	if err := os.WriteFile(destination, data, 0o600); err != nil {
		return fmt.Errorf("write destination file %s: %w", destination, err)
	}
	return nil
}
