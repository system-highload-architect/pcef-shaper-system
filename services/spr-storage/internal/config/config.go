package config

import (
	"os"
	"strconv"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	MaxMockProfiles   int `yaml:"max_mock_profiles"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	// Перекрытие лимитов емкости из ENV Kubernetes при необходимости
	if envMax := os.Getenv("SPR_MAX_PROFILES"); envMax != "" {
		if m, err := strconv.Atoi(envMax); err == nil {
			cfg.MaxMockProfiles = m
		}
	}
	return cfg
}
