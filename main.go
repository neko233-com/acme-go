package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"acme-go/internal/acme"
	"acme-go/internal/config"
)

const version = "0.1.0"

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

	switch args[0] {
	case "plan":
		return runPlan(args[1:])
	case "list":
		return runList(args[1:])
	case "info":
		return runInfo(args[1:])
	case "issue":
		return runIssue(args[1:], false)
	case "renew":
		return runIssue(args[1:], true)
	case "revoke":
		return runRevoke(args[1:])
	case "providers":
		return acme.Providers(os.Stdout)
	case "version":
		fmt.Println(version)
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	configPath := fs.String("config", "config.yaml", "path to config file")
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

func runIssue(args []string, renewMode bool) error {
	fs := flag.NewFlagSet("issue", flag.ContinueOnError)
	configPath := fs.String("config", "config.yaml", "path to config file")
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

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	configPath := fs.String("config", "config.yaml", "path to config file")
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
	configPath := fs.String("config", "config.yaml", "path to config file")
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
	configPath := fs.String("config", "config.yaml", "path to config file")
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

func printUsage() {
	fmt.Print(`acme-go is a config-driven ACME client.

Usage:
  acme-go plan   -config config.yaml
  acme-go list   -config config.yaml
  acme-go info   -config config.yaml -name example
  acme-go issue  -config config.yaml [-name example] [-force]
  acme-go renew  -config config.yaml [-name example] [-force]
  acme-go revoke -config config.yaml -name example
  acme-go providers
  acme-go version
`)
}
