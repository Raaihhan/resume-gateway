package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load membaca YAML. Prioritas lokasi:
// 1) argumen path (jika diisi)
// 2) ENV CONFIG_PATH
// 3) ./config.yaml
// 4) ./config/config.yaml
func Load(path string) (*Config, error) {
	if path == "" {
		if v := os.Getenv("CONFIG_PATH"); v != "" {
			path = v
		} else if _, err := os.Stat("config.yaml"); err == nil {
			path = "config.yaml"
		} else if _, err := os.Stat(filepath.Join("config", "config.yaml")); err == nil {
			path = filepath.Join("config", "config.yaml")
		} else {
			return nil, errors.New("config file not found (set CONFIG_PATH or create ./config.yaml)")
		}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
