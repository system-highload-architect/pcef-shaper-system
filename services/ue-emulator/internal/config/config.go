package config

import (
	"os"
	"strconv"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig    `yaml:",inline"`
	GatewayAddr          string `yaml:"gateway_addr"`
	SimulatedSubscribers int    `yaml:"simulated_subscribers"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envGateway := os.Getenv("ACCESS_GATEWAY_ADDR"); envGateway != "" {
		cfg.GatewayAddr = envGateway
	}
	if envSubs := os.Getenv("SIMULATED_SUBSCRIBERS_COUNT"); envSubs != "" {
		if s, err := strconv.Atoi(envSubs); err == nil {
			cfg.SimulatedSubscribers = s
		}
	}
	return cfg
}
