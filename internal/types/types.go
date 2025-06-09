package types

import (
	"time"
)

// AnomalyType представляет тип аномалии
type AnomalyType string

const (
	AnomalyTypeMetric AnomalyType = "metric"
	AnomalyTypeLog    AnomalyType = "log"
	AnomalyTypeTrace  AnomalyType = "trace"
)

// Severity представляет серьезность аномалии
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// ActionType представляет тип действия по восстановлению
type ActionType string

const (
	ActionTypeScale        ActionType = "scale"
	ActionTypeRestart      ActionType = "restart"
	ActionTypeScript       ActionType = "script"
	ActionTypeNotification ActionType = "notification"
)

// Anomaly представляет обнаруженную аномалию
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

// RemediationAction представляет действие по восстановлению
type RemediationAction struct {
	ID          string                 `json:"id"`
	AnomalyID   string                 `json:"anomaly_id"`
	Type        ActionType             `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	ExecutedAt  *time.Time             `json:"executed_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Result      *ActionResult          `json:"result,omitempty"`
}

// ActionResult представляет результат выполнения действия
type ActionResult struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
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

// MetricData представляет метрические данные
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

// HealthCheck представляет результат проверки здоровья
type HealthCheck struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Config представляет общую конфигурацию
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
	Logging     LoggingConfig     `yaml:"logging"`
	ML          MLConfig          `yaml:"ml"`
	Remediation RemediationConfig `yaml:"remediation"`
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

// MLConfig представляет конфигурацию ML компонентов
type MLConfig struct {
	ModelPath       string  `yaml:"model_path"`
	TrainingEnabled bool    `yaml:"training_enabled"`
	Threshold       float64 `yaml:"threshold"`
}

// RemediationConfig представляет конфигурацию восстановления
type RemediationConfig struct {
	Enabled         bool          `yaml:"enabled"`
	AutoExecute     bool          `yaml:"auto_execute"`
	MaxRetries      int           `yaml:"max_retries"`
	RetryDelay      time.Duration `yaml:"retry_delay"`
	NotificationURL string        `yaml:"notification_url"`
}
