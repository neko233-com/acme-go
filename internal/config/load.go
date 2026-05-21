package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Load reads the primary JSON config, merges the optional <name>.local.json
// override, applies defaults, then validates the fully resolved result.
func Load(path string) (*Config, error) {
	cfg, err := loadJSONConfig(path)
	if err != nil {
		return nil, err
	}

	localOverridePath := localOverridePath(path)
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

func localOverridePath(path string) string {
	base := filepath.Base(path)
	extension := filepath.Ext(base)
	if extension == "" {
		return filepath.Join(filepath.Dir(path), base+".local.json")
	}
	name := strings.TrimSuffix(base, extension)
	return filepath.Join(filepath.Dir(path), name+".local.json")
}

func loadJSONConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal([]byte(os.ExpandEnv(string(data))), &cfg); err != nil {
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
