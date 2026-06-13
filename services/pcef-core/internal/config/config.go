package config

import (
	"os"
	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	config.BaseConfig `yaml:",inline"`
	PcrfAddr          string `yaml:"pcrf_addr"`
	OcsAddr           string `yaml:"ocs_addr"`
	OfcsAddr          string `yaml:"ofcs_addr"`
}

func LoadConfig(yamlPath string) *Config {
	base := config.LoadBaseConfig(yamlPath)

	// Задаем жесткие дефолтные b2b-фолбэки для локального дебага на ПК
	// Hardcoded local fallbacks to guarantee compile-time safety
	cfg := &Config{
		BaseConfig: *base,
		PcrfAddr:   "localhost:50053",
		OcsAddr:    "localhost:50054", // ПУЛЕНЕПРОБИВАЕМЫЙ ДЕФОЛТ ПОРТА OCS
		OfcsAddr:   "localhost:50055",
	}

	// 1. Дочитываем специфичные локальные поля из YAML (если файл существует)
	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	// 2. Перекрываем локальные адреса ENV-переменными ИЗ КУБЕРА (только если они НЕ пустые!)
	// 2. Override with K8s environment variables strictly if they are populated!
	if envPcrf := os.Getenv("PCRF_ENGINE_ADDR"); envPcrf != "" {
		cfg.PcrfAddr = envPcrf
	}
	if envOcs := os.Getenv("OCS_RATING_ADDR"); envOcs != "" {
		cfg.OcsAddr = envOcs
	}
	if envOfcs := os.Getenv("OFCS_COLLECTOR_ADDR"); envOfcs != "" {
		cfg.OfcsAddr = envOfcs
	}

	return cfg
}
