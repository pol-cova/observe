package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RefreshInterval int        `yaml:"refresh_interval"`
	Thresholds      Thresholds `yaml:"thresholds"`
	Panels          []string   `yaml:"panels"`
}

type Thresholds struct {
	CPU    float64 `yaml:"cpu"`
	Memory float64 `yaml:"memory"`
	Swap   float64 `yaml:"swap"`
	Disk   float64 `yaml:"disk"`
	IOWait float64 `yaml:"io_wait"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.RefreshInterval < 100 {
		cfg.RefreshInterval = 500
	}
	return cfg, nil
}
