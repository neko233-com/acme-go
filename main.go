package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/neko233-com/acme-go/internal/acme"
	"github.com/neko233-com/acme-go/internal/config"
	"github.com/neko233-com/acme-go/internal/doc"
	"github.com/neko233-com/acme-go/internal/update"
)

var version = "dev"

var (
	openGuide      = doc.OpenGuide
	resolveGuide   = doc.ResolveGuidePath
	locateExecPath = os.Executable
	locateWorkDir  = os.Getwd
	autoUpdate     = update.MaybeAutoUpdate
)

const defaultConfigPath = "config_acme.json"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	if shouldAutoUpdate(args[0]) {
		executablePath, err := locateExecPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: auto update skipped: locate executable: %v\n", err)
		} else if err := autoUpdate(version, executablePath, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "warning: auto update skipped: %v\n", err)
		}
	}

	switch args[0] {
	case "validate":
		return runValidate(args[1:])
	case "plan":
		return runPlan(args[1:])
	case "paths":
		return runPaths(args[1:])
	case "list":
		return runList(args[1:])
	case "info":
		return runInfo(args[1:])
	case "issue":
		return runIssue(args[1:], false)
	case "renew":
		return runIssue(args[1:], true)
	case "renew-loop", "auto-renew", "watch":
		return runRenewLoop(args[1:])
	case "revoke":
		return runRevoke(args[1:])
	case "install-cert":
		return runInstallCert(args[1:])
	case "deploy":
		return runDeploy(args[1:])
	case "providers":
		return acme.Providers(os.Stdout)
	case "version":
		return runVersion(args[1:])
	case "upgrade":
		return runUpgrade(args[1:])
	case "doc":
		return runDoc(args[1:])
	case "help", "-h", "--help":
		return runHelp(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func shouldAutoUpdate(command string) bool {
	switch command {
	case "help", "-h", "--help", "doc", "version", "upgrade":
		return false
	default:
		return true
	}
}

func runHelp(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	helpText, err := commandHelp(args[0])
	if err != nil {
		return err
	}
	fmt.Print(helpText)
	return nil
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	return acme.Validate(cfg, os.Stdout)
}

func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "issue or inspect only a single certificate entry")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	return acme.Plan(cfg, *name, os.Stdout)
}

func runPaths(args []string) error {
	fs := flag.NewFlagSet("paths", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "show paths only for a single certificate entry")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	return acme.Paths(cfg, *name, os.Stdout)
}

func runIssue(args []string, renewMode bool) error {
	fs := flag.NewFlagSet("issue", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "issue or renew only a single certificate entry")
	force := fs.Bool("force", false, "force issuing even if the current certificate is still valid")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	mode := acme.ModeIssue
	if renewMode {
		mode = acme.ModeRenew
	}

	result, err := acme.Run(cfg, acme.Options{
		Name:  *name,
		Force: *force,
		Mode:  mode,
		Out:   os.Stdout,
	})
	if err != nil {
		return err
	}
	if result.Changed == 0 && *force {
		return errors.New("no certificate matched the selected name")
	}
	return nil
}

func runRenewLoop(args []string) error {
	fs := flag.NewFlagSet("renew-loop", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "renew only a single certificate entry")
	force := fs.Bool("force", false, "force renewal on each scheduled run")
	intervalValue := fs.String("interval", "", "override automation.renew_interval with a Go duration such as 30m or 24h")
	runOnce := fs.Bool("once", false, "run one renew cycle immediately and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if *runOnce {
		_, err := acme.Run(cfg, acme.Options{Name: *name, Force: *force, Mode: acme.ModeRenew, Out: os.Stdout})
		return err
	}

	var interval time.Duration
	if *intervalValue != "" {
		interval, err = time.ParseDuration(*intervalValue)
		if err != nil {
			return fmt.Errorf("parse -interval: %w", err)
		}
	} else {
		interval, err = cfg.Automation.RenewIntervalDuration()
		if err != nil {
			return err
		}
	}

	targetName := *name
	if targetName == "" {
		targetName = "all certificates"
	}
	fmt.Fprintf(os.Stdout, "starting renew loop for %s; interval=%s; press Ctrl+C to stop\n", targetName, interval)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return acme.AutoRenewLoop(ctx, cfg, acme.AutoRenewOptions{
		Name:           *name,
		Force:          *force,
		Interval:       interval,
		RunImmediately: true,
		Out:            os.Stdout,
	})
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	return acme.List(cfg, os.Stdout)
}

func runInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "certificate entry name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("-name is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	return acme.Info(cfg, *name, os.Stdout)
}

func runRevoke(args []string) error {
	fs := flag.NewFlagSet("revoke", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "certificate entry name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("-name is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	return acme.Revoke(cfg, *name, os.Stdout)
}

func runInstallCert(args []string) error {
	fs := flag.NewFlagSet("install-cert", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "certificate entry name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("-name is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	return acme.Install(cfg, *name, os.Stdout)
}

func runDeploy(args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigPath, "path to config file")
	name := fs.String("name", "", "certificate entry name")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return errors.New("-name is required")
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	return acme.Deploy(cfg, *name, os.Stdout)
}

func runVersion(args []string) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	info, err := update.CheckVersion(version)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(info)
}

func runUpgrade(args []string) error {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	return update.Upgrade(version, executablePath, os.Stdout)
}

func runDoc(args []string) error {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	printOnly := fs.Bool("print-path", false, "print the resolved HTML guide path without opening it")
	if err := fs.Parse(args); err != nil {
		return err
	}
	workingDir, err := locateWorkDir()
	if err != nil {
		return fmt.Errorf("locate working directory: %w", err)
	}
	executablePath, err := locateExecPath()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	if *printOnly {
		guidePath, err := resolveGuide(workingDir, executablePath)
		if err != nil {
			return err
		}
		fmt.Println(guidePath)
		return nil
	}
	guidePath, err := openGuide(workingDir, executablePath)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "opened %s\n", guidePath)
	return nil
}

func printUsage() {
	fmt.Print(`acme-go is a config-driven ACME client.

Usage:
	acme-go help [command]
	acme-go doc [-print-path]
	acme-go validate -config config_acme.json
	acme-go plan   -config config_acme.json
	acme-go paths  -config config_acme.json [-name example]
	acme-go list   -config config_acme.json
	acme-go info   -config config_acme.json -name example
	acme-go issue  -config config_acme.json [-name example] [-force]
	acme-go renew  -config config_acme.json [-name example] [-force]
	acme-go renew-loop  -config config_acme.json [-name example] [-force] [-interval 24h] [-once]
	acme-go auto-renew  -config config_acme.json [-name example] [-force] [-interval 24h] [-once]
	acme-go revoke -config config_acme.json -name example
	acme-go install-cert -config config_acme.json -name example
	acme-go deploy -config config_acme.json -name example
  acme-go providers
  acme-go version
	acme-go upgrade
`)
}

func commandHelp(name string) (string, error) {
	switch name {
	case "help", "-h", "--help":
		return "acme-go help [command]\nShow overall usage or detailed help for a single command.\n", nil
	case "doc":
		return "acme-go doc [-print-path]\nOpen how-to-use.html in the default browser, or print its resolved path.\n", nil
	case "validate":
		return "acme-go validate -config config_acme.json\nLoad config, merge <name>.local.json, apply defaults, and print a JSON summary.\n", nil
	case "plan":
		return "acme-go plan -config config_acme.json [-name example]\nShow which certificates would issue or skip based on local state.\n", nil
	case "paths":
		return "acme-go paths -config config_acme.json [-name example]\nPrint resolved output paths for certificate files.\n", nil
	case "list":
		return "acme-go list -config config_acme.json\nList all configured certificates and their current status.\n", nil
	case "info":
		return "acme-go info -config config_acme.json -name example\nShow detailed information for a single certificate entry.\n", nil
	case "issue":
		return "acme-go issue -config config_acme.json [-name example] [-force]\nIssue a new certificate or replace an existing one.\n", nil
	case "renew":
		return "acme-go renew -config config_acme.json [-name example] [-force]\nRenew certificates that are due, or force renewal.\n", nil
	case "renew-loop", "auto-renew", "watch":
		return "acme-go renew-loop -config config_acme.json [-name example] [-force] [-interval 24h] [-once]\nAlias: auto-renew, watch. Run renew on a schedule. The interval defaults to automation.renew_interval, which falls back to 24h unless -interval overrides it. Use -once to run a single renew cycle and exit.\n", nil
	case "revoke":
		return "acme-go revoke -config config_acme.json -name example\nRevoke a locally stored certificate through the ACME server.\n", nil
	case "install-cert":
		return "acme-go install-cert -config config_acme.json -name example\nCopy generated certificate files into configured install destinations.\n", nil
	case "deploy":
		return "acme-go deploy -config config_acme.json -name example\nRun configured deploy targets with existing local certificate material.\n", nil
	case "providers":
		return "acme-go providers\nPrint supported DNS providers and aliases as JSON.\n", nil
	case "version":
		return "acme-go version\nPrint current version, latest GitHub release, and concise change summary.\n", nil
	case "upgrade":
		return "acme-go upgrade\nDownload the latest release artifact for the current OS and architecture and replace the local binary. Automatic update checks are enabled by default; set ACME_GO_AUTO_UPDATE=false to disable them.\n", nil
	default:
		return "", fmt.Errorf("unknown help topic %q", name)
	}
}
