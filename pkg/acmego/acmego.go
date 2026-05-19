package acmego

import (
	"io"

	"github.com/neko233-com/acme-go/internal/acme"
	"github.com/neko233-com/acme-go/internal/config"
	"github.com/neko233-com/acme-go/internal/doc"
	"github.com/neko233-com/acme-go/internal/update"
)

type Config = config.Config
type CAConfig = config.CAConfig
type AccountConfig = config.AccountConfig
type DNSConfig = config.DNSConfig
type CertificateSpec = config.CertificateSpec
type CertificateFiles = config.CertificateFiles
type CertificatePaths = config.CertificatePaths
type InstallConfig = config.InstallConfig
type DeployTarget = config.DeployTarget
type HookConfig = config.HookConfig

type Mode = acme.Mode

const (
	ModeIssue Mode = acme.ModeIssue
	ModeRenew Mode = acme.ModeRenew
)

type Options = acme.Options
type Result = acme.Result
type Metadata = acme.Metadata
type ConfigSummary = acme.ConfigSummary
type CertificatePathInfo = acme.CertificatePathInfo
type ReleaseDiff = update.ReleaseDiff
type VersionInfo = update.VersionInfo

func Load(path string) (*Config, error) {
	return config.Load(path)
}

func Plan(cfg *Config, name string, out io.Writer) error {
	return acme.Plan(cfg, name, out)
}

func Validate(cfg *Config, out io.Writer) error {
	return acme.Validate(cfg, out)
}

func Paths(cfg *Config, name string, out io.Writer) error {
	return acme.Paths(cfg, name, out)
}

func Run(cfg *Config, options Options) (Result, error) {
	return acme.Run(cfg, options)
}

func Issue(cfg *Config, name string, force bool, out io.Writer) (Result, error) {
	return acme.Run(cfg, Options{Name: name, Force: force, Mode: ModeIssue, Out: out})
}

func Renew(cfg *Config, name string, force bool, out io.Writer) (Result, error) {
	return acme.Run(cfg, Options{Name: name, Force: force, Mode: ModeRenew, Out: out})
}

func List(cfg *Config, out io.Writer) error {
	return acme.List(cfg, out)
}

func Info(cfg *Config, name string, out io.Writer) error {
	return acme.Info(cfg, name, out)
}

func Revoke(cfg *Config, name string, out io.Writer) error {
	return acme.Revoke(cfg, name, out)
}

func Install(cfg *Config, name string, out io.Writer) error {
	return acme.Install(cfg, name, out)
}

func Deploy(cfg *Config, name string, out io.Writer) error {
	return acme.Deploy(cfg, name, out)
}

func Providers(out io.Writer) error {
	return acme.Providers(out)
}

func CheckVersion(currentVersion string) (VersionInfo, error) {
	return update.CheckVersion(currentVersion)
}

func Upgrade(currentVersion, executablePath string, out io.Writer) error {
	return update.Upgrade(currentVersion, executablePath, out)
}

func MaybeAutoUpdate(currentVersion, executablePath string, out io.Writer) error {
	return update.MaybeAutoUpdate(currentVersion, executablePath, out)
}

func ResolveGuidePath(workingDir, executablePath string) (string, error) {
	return doc.ResolveGuidePath(workingDir, executablePath)
}

func OpenGuide(workingDir, executablePath string) (string, error) {
	return doc.OpenGuide(workingDir, executablePath)
}
