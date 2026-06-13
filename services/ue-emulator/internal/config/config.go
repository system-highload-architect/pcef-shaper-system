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
	cfg := &Config{
		BaseConfig:           *base,
		GatewayAddr:          "localhost:50052", // ПУЛЕНЕПРОБИВАЕМЫЙ ДЕФОЛТ ПОРТА К ЯДРУ PCEF-CORE
		SimulatedSubscribers: 5,
	}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envGateway := os.Getenv("PCEF_CORE_ADDR"); envGateway != "" {
		cfg.GatewayAddr = envGateway
	}
	if envSubs := os.Getenv("SIMULATED_SUBSCRIBERS_COUNT"); envSubs != "" {
		if s, err := strconv.Atoi(envSubs); err == nil {
			cfg.SimulatedSubscribers = s
		}
	}
	return cfg
}
