package config

import (
	"os"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	DataDiskPath      string `yaml:"data_disk_path"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envPath := os.Getenv("CLICKHOUSE_DISK_PATH"); envPath != "" {
		cfg.DataDiskPath = envPath
	}
	return cfg
}
