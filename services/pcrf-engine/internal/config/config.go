package config

import (
	"os"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	SprAddr           string `yaml:"spr_addr"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envSpr := os.Getenv("SCYLLADB_HOSTS"); envSpr != "" {
		cfg.SprAddr = envSpr
	}
	return cfg
}
