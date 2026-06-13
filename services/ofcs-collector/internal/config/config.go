package config

import (
	"os"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	KafkaBrokers      string `yaml:"kafka_brokers"`
	ClickhouseAddr    string `yaml:"clickhouse_addr"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{
		BaseConfig:     *base,
		KafkaBrokers:   "localhost:9092",
		ClickhouseAddr: "localhost:8123", // Наш дефолтный порт эмулятора ClickHouse
	}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	if envKafka := os.Getenv("KAFKA_BROKERS"); envKafka != "" {
		cfg.KafkaBrokers = envKafka
	}
	if envCH := os.Getenv("CLICKHOUSE_ADDR"); envCH != "" {
		cfg.ClickhouseAddr = envCH
	}
	return cfg
}
