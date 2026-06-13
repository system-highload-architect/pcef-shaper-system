package config

import (
	"os"
	// Импортируем наше общее шасси из корня монорепозитория
	"pcef-shaper-system/internal/chassis/config"

	"gopkg.in/yaml.v3"
)

// Config расширяет базовое шасси уникальными адресами CoreDNS K8s
type Config struct {
	config.BaseConfig `yaml:",inline"` // Вшиваем базовые поля (Name, Port) inline-пакетом

	PcrfAddr string `yaml:"pcrf_addr"` // Адрес PCRF-мозга сети
	OcsAddr  string `yaml:"ocs_addr"`  // Адрес OCS-биллинга Aerospike
	OfcsAddr string `yaml:"ofcs_addr"` // Адрес асинхронного OFCS Kafka
}

// LoadConfig — локальный оркестратор настроек конкретного сервиса
func LoadConfig(yamlPath string) *Config {
	// 1. Сначала загружаем базовые общие поля через наше шасси
	base := config.LoadBaseConfig(yamlPath)

	cfg := &Config{
		BaseConfig: *base,
	}

	// 2. Дочитываем специфичные локальные поля из YAML
	if data, err := os.ReadFile(yamlPath); err == nil {
		_ = yaml.Unmarshal(data, cfg)
	}

	// 3. Перекрываем локальные адреса переменными окружения из k8s/pcef-configmap.yaml!
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
