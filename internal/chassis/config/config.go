package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// BaseConfig инкапсулирует общие параметры рантайма для всех 7 Go-сервисов
// BaseConfig encapsulates standard runtime metrics shared by all 7 Go services
type BaseConfig struct {
	ServiceName     string `yaml:"service_name"`
	BindAddr        string `yaml:"bind_addr"`
	LogLevel        string `yaml:"log_level"`
	ShutdownTimeout int    `yaml:"shutdown_timeout"` // В секундах / In seconds
}

// LoadBaseConfig атомарно собирает параметры из YAML и перекрывает их ENV-переменными K8s
// LoadBaseConfig combines local YAML parameters and overrides them with K8s cluster ENV variables
func LoadBaseConfig(defaultYamlPath string) *BaseConfig {
	// Дефолтные enterprise-значения на случай отсутствия окружения
	// Hardcoded architecture fallbacks in case of environment starvation
	cfg := &BaseConfig{
		ServiceName:     "pcef-generic-service",
		BindAddr:        ":50050",
		LogLevel:        "INFO",
		ShutdownTimeout: 15,
	}

	// 1. Попытка парсинга локального YAML файла (для dev-разработки на ПК)
	// 1. Attempting to parse local YAML config (for local workstation debugging)
	if data, err := os.ReadFile(defaultYamlPath); err == nil {
		if yamlErr := yaml.Unmarshal(data, cfg); yamlErr != nil {
			log.Printf("[CHASSIS WARN] Сбой разбора локального файла %s: %v", defaultYamlPath, yamlErr)
		} else {
			log.Printf("[CHASSIS INIT] Локальная конфигурация %s успешно загружена", defaultYamlPath)
		}
	}

	// 2. Слой перекрытия ENV-переменными из Kubernetes ConfigMap (Продакшен-эшелон)
	// 2. K8s ConfigMap environment injection override layer (Production runtime enforcement)
	if envName := os.Getenv("SERVICE_NAME"); envName != "" {
		cfg.ServiceName = envName
	}
	if envAddr := os.Getenv("BIND_ADDR"); envAddr != "" {
		cfg.BindAddr = envAddr
	}
	if envLog := os.Getenv("LOG_LEVEL"); envLog != "" {
		cfg.LogLevel = strings.ToUpper(envLog)
	}
	if envTimeout := os.Getenv("SHUTDOWN_TIMEOUT"); envTimeout != "" {
		if t, err := strconv.Atoi(envTimeout); err == nil {
			cfg.ShutdownTimeout = t
		}
	}

	log.Printf("[CHASSIS SUCCESS] Контур конфигурации для [%s] успешно взведен в RAM (Слушает порт %s)", cfg.ServiceName, cfg.BindAddr)
	return cfg
}
