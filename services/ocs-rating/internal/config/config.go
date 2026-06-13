package config

import (
	"os"
	"strconv"

	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig   `yaml:",inline"`
	QuotaChunkBytes     uint64 `yaml:"quota_chunk_bytes"`
	InitialBalanceBytes uint64 `yaml:"initial_balance_bytes"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)
	cfg := &Config{BaseConfig: *base}

	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	// Перекрытие из k8s/pcef-configmap.yaml
	if envChunk := os.Getenv("GY_QUOTA_CHUNK_MB"); envChunk != "" {
		if mb, err := strconv.ParseUint(envChunk, 10, 64); err == nil {
			cfg.QuotaChunkBytes = mb * 1024 * 1024 // Конвертируем МБ в Байты
		}
	}
	return cfg
}
