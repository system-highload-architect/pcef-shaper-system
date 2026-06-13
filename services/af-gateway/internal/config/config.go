package config

import (
	"os"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	PcrfAddr          string `yaml:"pcrf_addr"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envPcrf := os.Getenv("PCRF_ENGINE_ADDR"); envPcrf != "" {
		cfg.PcrfAddr = envPcrf
	}
	return cfg
}
