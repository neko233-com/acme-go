package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load reads the primary YAML config, merges the optional .local.json override,
// applies defaults, then validates the fully resolved result.
func Load(path string) (*Config, error) {
	cfg, err := loadYAML(path)
	if err != nil {
		return nil, err
	}

	localOverridePath := filepath.Join(filepath.Dir(path), ".local.json")
	if override, err := loadJSON(localOverridePath); err == nil {
		cfg.merge(override)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if err := cfg.applyDefaults(path); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadYAML(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(data))), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func loadJSON(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal([]byte(os.ExpandEnv(string(data))), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse local override %s: %w", path, err)
	}
	return cfg, nil
}
