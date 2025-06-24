package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"aiops-platform/internal/types"
)

// LoadConfig загружает конфигурацию из файла
func LoadConfig(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config types.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Применяем значения по умолчанию
	setDefaults(&config)

	return &config, nil
}

// LoadConfigFromEnv загружает конфигурацию из переменных окружения
func LoadConfigFromEnv() *types.Config {
	config := &types.Config{
		Server: types.ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: types.DatabaseConfig{
			Type:     getEnv("DB_TYPE", "sqlite"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			Name:     getEnv("DB_NAME", "aiops"),
			Username: getEnv("DB_USERNAME", "aiops"),
			Password: getEnv("DB_PASSWORD", ""),
		},
		Monitoring: types.MonitoringConfig{
			Prometheus: types.PrometheusConfig{
				URL: getEnv("PROMETHEUS_URL", "http://localhost:9090"),
			},
			Grafana: types.GrafanaConfig{
				URL:      getEnv("GRAFANA_URL", "http://localhost:3000"),
				Username: getEnv("GRAFANA_USERNAME", "admin"),
				Password: getEnv("GRAFANA_PASSWORD", "admin123"),
			},
		},
		Logging: types.LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Loki: types.LokiConfig{
				URL: getEnv("LOKI_URL", "http://localhost:3100"),
			},
		},
		ML: types.MLConfig{
			ModelPath:       getEnv("ML_MODEL_PATH", "/data/models"),
			TrainingEnabled: getEnvBool("ML_TRAINING_ENABLED", true),
			Threshold:       getEnvFloat("ML_THRESHOLD", 0.8),
			FeatureWindow:   getEnvInt("ML_FEATURE_WINDOW", 20),
			ModelType:       getEnv("ML_MODEL_TYPE", "isolation_forest"),
			StoragePath:     getEnv("ML_STORAGE_PATH", "/data/models"),
		},
		// REMEDIATION REMOVED - Will use Rundeck for orchestration
	}

	setDefaults(config)
	return config
}

// setDefaults устанавливает значения по умолчанию
func setDefaults(config *types.Config) {
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}
	if config.ML.Threshold == 0 {
		config.ML.Threshold = 0.8
	}
	if config.ML.FeatureWindow == 0 {
		config.ML.FeatureWindow = 20
	}
	if config.ML.ModelType == "" {
		config.ML.ModelType = "isolation_forest"
	}
	if config.ML.StoragePath == "" {
		config.ML.StoragePath = "/data/models"
	}
	// REMEDIATION DEFAULTS REMOVED - Rundeck will handle orchestration
}

// getEnv возвращает значение переменной окружения или значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt возвращает целое значение переменной окружения
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := parseInt(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool возвращает булево значение переменной окружения
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := parseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvFloat возвращает значение с плавающей точкой переменной окружения
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := parseFloat(value); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvDuration возвращает значение времени переменной окружения
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvStringSlice возвращает срез строк из переменной окружения
func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// Простые парсеры без внешних зависимостей
func parseInt(s string) (int, error) {
	var result int
	for _, char := range s {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(char-'0')
	}
	return result, nil
}

func parseBool(s string) (bool, error) {
	switch s {
	case "true", "True", "TRUE", "1", "yes", "Yes", "YES":
		return true, nil
	case "false", "False", "FALSE", "0", "no", "No", "NO":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean: %s", s)
	}
}

func parseFloat(s string) (float64, error) {
	var result float64
	var decimals float64
	var hasDecimal bool
	var decimalPlace float64 = 1

	for _, char := range s {
		if char == '.' {
			if hasDecimal {
				return 0, fmt.Errorf("invalid float: %s", s)
			}
			hasDecimal = true
			continue
		}
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid float: %s", s)
		}
		if hasDecimal {
			decimalPlace *= 10
			decimals = decimals*10 + float64(char-'0')
		} else {
			result = result*10 + float64(char-'0')
		}
	}

	return result + decimals/decimalPlace, nil
}
