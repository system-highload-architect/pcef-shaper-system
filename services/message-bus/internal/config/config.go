package config

import (
	"os"
	"strconv"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	QueueCapacity     int `yaml:"queue_capacity"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envCap := os.Getenv("KAFKA_QUEUE_CAPACITY"); envCap != "" {
		if c, err := strconv.Atoi(envCap); err == nil {
			cfg.QueueCapacity = c
		}
	}
	return cfg
}
