package types
import (
	"time"
)
// AnomalyType представляет тип аномалии
type AnomalyType string
const (
	MetricAnomaly AnomalyType = "metric"
	LogAnomaly    AnomalyType = "log"
	SystemAnomaly AnomalyType = "system"
)
// Severity представляет уровень серьезности
type Severity string
const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)
// ActionType представляет тип действия для восстановления
type ActionType string
const (
	ActionTypeKubernetes   ActionType = "kubernetes"
	ActionTypeScript       ActionType = "script"
	ActionTypeNotification ActionType = "notification"
)
// Anomaly представляет аномалию в системе
type Anomaly struct {
	ID          string                 `json:"id"`
	Type        AnomalyType            `json:"type"`
	Severity    Severity               `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Metric      string                 `json:"metric,omitempty"`
	Value       float64                `json:"value,omitempty"`
	Threshold   float64                `json:"threshold,omitempty"`
	Labels      map[string]string      `json:"labels"`
	Metadata    map[string]interface{} `json:"metadata"`
	DetectedAt  time.Time              `json:"detected_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Status      string                 `json:"status"`
}
// DetectorConfig представляет конфигурацию детектора
type DetectorConfig struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Enabled    bool                   `json:"enabled"`
	Parameters map[string]interface{} `json:"parameters"`
	Thresholds map[string]float64     `json:"thresholds"`
	Labels     map[string]string      `json:"labels"`
	Schedule   string                 `json:"schedule"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}
// MetricData представляет данные метрики
type MetricData struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels"`
}
// LogEntry представляет запись в логе
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Labels    map[string]string      `json:"labels"`
	Fields    map[string]interface{} `json:"fields"`
}
// Event представляет событие в системе
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Subject   string                 `json:"subject"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}
// HealthCheck представляет проверку здоровья
type HealthCheck struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
// Config представляет конфигурацию приложения
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Logging    LoggingConfig    `yaml:"logging"`
	ML         MLConfig         `yaml:"ml"`
	Rundeck    RundeckConfig    `yaml:"rundeck"`
}
// ServerConfig представляет конфигурацию сервера
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}
// DatabaseConfig представляет конфигурацию базы данных
type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
// MonitoringConfig представляет конфигурацию мониторинга
type MonitoringConfig struct {
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Grafana    GrafanaConfig    `yaml:"grafana"`
}
// PrometheusConfig представляет конфигурацию Prometheus
type PrometheusConfig struct {
	URL string `yaml:"url"`
}
// GrafanaConfig представляет конфигурацию Grafana
type GrafanaConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
// LoggingConfig представляет конфигурацию логирования
type LoggingConfig struct {
	Level  string     `yaml:"level"`
	Format string     `yaml:"format"`
	Loki   LokiConfig `yaml:"loki"`
}
// LokiConfig представляет конфигурацию Loki
type LokiConfig struct {
	URL string `yaml:"url"`
}
// MLConfig представляет конфигурацию ML
type MLConfig struct {
	ModelPath       string  `yaml:"model_path"`
	TrainingEnabled bool    `yaml:"training_enabled"`
	Threshold       float64 `yaml:"threshold"`
	FeatureWindow   int     `yaml:"feature_window"`
	ModelType       string  `yaml:"model_type"`
	StoragePath     string  `yaml:"storage_path"`
}
// FeatureVector представляет вектор признаков для ML
type FeatureVector struct {
	Values     []float64         `json:"values"`
	Labels     map[string]string `json:"labels"`
	Timestamp  time.Time         `json:"timestamp"`
	MetricName string            `json:"metric_name"`
}
// ProcessedMetrics представляет обработанные метрики
type ProcessedMetrics struct {
	Data       []MetricData   `json:"data"`
	Features   *FeatureVector `json:"features,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
	WindowSize int            `json:"window_size"`
}
// ModelInfo представляет информацию о модели
type ModelInfo struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Version    string                 `json:"version"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Accuracy   float64                `json:"accuracy"`
	Parameters map[string]interface{} `json:"parameters"`
	FilePath   string                 `json:"file_path"`
}
// TrainingData представляет данные для обучения
type TrainingData struct {
	Features [][]float64            `json:"features"`
	Labels   []int                  `json:"labels"` // 0 - normal, 1 - anomaly
	Metadata map[string]interface{} `json:"metadata"`
}
// RundeckConfig представляет конфигурацию Rundeck
type RundeckConfig struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Project  string `yaml:"project"`
}
