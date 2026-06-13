package config

import (
	"os"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	PcefCoreAddr      string `yaml:"pcef_core_addr"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envPcef := os.Getenv("PCEF_CORE_ADDR"); envPcef != "" {
		cfg.PcefCoreAddr = envPcef
	}
	return cfg
}
