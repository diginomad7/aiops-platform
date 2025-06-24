package detector

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"aiops-platform/internal/ml"
	"aiops-platform/internal/types"
)

// Detector представляет основной детектор аномалий
type Detector struct {
	config     *types.Config
	server     *http.Server
	processors []Processor
	shutdown   chan struct{}
	mu         sync.RWMutex
	running    bool
	mlPipeline *ml.Pipeline
}

// Processor интерфейс для обработчиков аномалий
type Processor interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
}

// New создает новый детектор
func New(config *types.Config) (*Detector, error) {
	detector := &Detector{
		config:     config,
		processors: make([]Processor, 0),
		shutdown:   make(chan struct{}),
	}

	// Инициализируем HTTP сервер
	mux := http.NewServeMux()
	mux.HandleFunc("/health", detector.healthHandler)
	mux.HandleFunc("/metrics", detector.metricsHandler)
	mux.HandleFunc("/api/v1/anomalies", detector.anomaliesHandler)

	detector.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler: mux,
	}

	// Добавляем процессоры
	if err := detector.initProcessors(); err != nil {
		return nil, fmt.Errorf("failed to initialize processors: %w", err)
	}

	// Инициализируем ML pipeline
	var err error
	detector.mlPipeline, err = ml.NewPipeline(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ML pipeline: %w", err)
	}

	return detector, nil
}

// Start запускает детектор
func (d *Detector) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("detector already running")
	}

	log.Printf("Starting AIOps detector...")

	// Запускаем ML pipeline
	if err := d.mlPipeline.Start(ctx); err != nil {
		return fmt.Errorf("failed to start ML pipeline: %w", err)
	}

	// Запускаем HTTP сервер
	go func() {
		log.Printf("Starting HTTP server on %s", d.server.Addr)
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	d.running = true

	log.Printf("AIOps detector started successfully")
	return nil
}

// Stop останавливает детектор
func (d *Detector) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil
	}

	log.Println("Stopping AIOps detector...")

	// Останавливаем ML pipeline
	if err := d.mlPipeline.Stop(ctx); err != nil {
		log.Printf("Error stopping ML pipeline: %v", err)
	}

	d.running = false
	log.Println("AIOps detector stopped")
	return nil
}

// initProcessors инициализирует процессоры
func (d *Detector) initProcessors() error {
	// Метрический процессор
	metricProcessor := &MetricProcessor{
		config: d.config,
	}
	d.processors = append(d.processors, metricProcessor)

	// Лог процессор
	logProcessor := &LogProcessor{
		config: d.config,
	}
	d.processors = append(d.processors, logProcessor)

	return nil
}

// HTTP обработчики

func (d *Detector) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

func (d *Detector) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# HELP aiops_detector_up Detector is running\n"))
	w.Write([]byte("# TYPE aiops_detector_up gauge\n"))
	w.Write([]byte("aiops_detector_up 1\n"))
}

func (d *Detector) anomaliesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		d.getAnomalies(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (d *Detector) getAnomalies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"anomalies":[],"total":0,"note":"Ready for Rundeck integration"}`))
}

func (d *Detector) performHealthCheck() {
	// Простая проверка здоровья
	log.Printf("Detector health check - processors: %d", len(d.processors))
}

// MetricProcessor обрабатывает метрики
type MetricProcessor struct {
	config *types.Config
}

func (p *MetricProcessor) Start(ctx context.Context) error {
	log.Printf("MetricProcessor started")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Обрабатываем метрики
			p.processMetrics()
		}
	}
}

func (p *MetricProcessor) Stop(ctx context.Context) error {
	log.Printf("MetricProcessor stopped")
	return nil
}

func (p *MetricProcessor) Name() string {
	return "metric-processor"
}

func (p *MetricProcessor) processMetrics() {
	// Заглушка для обработки метрик
	log.Printf("Processing metrics from %s", p.config.Monitoring.Prometheus.URL)
}

// LogProcessor обрабатывает логи
type LogProcessor struct {
	config *types.Config
}

func (p *LogProcessor) Start(ctx context.Context) error {
	log.Printf("LogProcessor started")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Обрабатываем логи
			p.processLogs()
		}
	}
}

func (p *LogProcessor) Stop(ctx context.Context) error {
	log.Printf("LogProcessor stopped")
	return nil
}

func (p *LogProcessor) Name() string {
	return "log-processor"
}

func (p *LogProcessor) processLogs() {
	// Заглушка для обработки логов
	log.Printf("Processing logs from %s", p.config.Logging.Loki.URL)
}

// DetectAnomalies обнаруживает аномалии в метриках
func (d *Detector) DetectAnomalies(metrics []types.MetricData) ([]*types.Anomaly, error) {
	// Обнаружение аномалий с помощью ML pipeline
	anomalies, err := d.mlPipeline.ProcessMetrics(metrics)
	if err != nil {
		return nil, fmt.Errorf("ML anomaly detection failed: %w", err)
	}

	// В будущем аномалии будут отправляться в Rundeck через REST API
	if len(anomalies) > 0 {
		log.Printf("Detected %d anomalies - will be sent to Rundeck orchestrator", len(anomalies))
	}

	return anomalies, nil
}

// GetMetrics возвращает метрики детектора
func (d *Detector) GetMetrics() map[string]interface{} {
	metrics := map[string]interface{}{
		"processors": len(d.processors),
		"running":    d.running,
	}

	// Добавляем метрики ML pipeline
	mlMetrics := d.mlPipeline.GetMetrics()
	if mlMetrics != nil {
		metrics["ml_processed_samples"] = mlMetrics.ProcessedSamples
		metrics["ml_detected_anomalies"] = mlMetrics.DetectedAnomalies
		metrics["ml_processing_latency"] = mlMetrics.ProcessingLatency
		metrics["ml_model_accuracy"] = mlMetrics.ModelAccuracy
		metrics["ml_error_count"] = mlMetrics.ErrorCount
	}

	return metrics
}

// GetHealth возвращает информацию о здоровье детектора
func (d *Detector) GetHealth() *types.HealthCheck {
	status := "healthy"
	message := "Detector is running normally - ready for Rundeck integration"

	if !d.running {
		status = "stopped"
		message = "Detector is not running"
	}

	return &types.HealthCheck{
		Service:   "aiops-detector",
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// ExportPrometheusMetrics экспортирует метрики в формате Prometheus
func (d *Detector) ExportPrometheusMetrics() string {
	var result string

	// Метрика статуса детектора
	result += "# HELP aiops_detector_up Detector is running\n"
	result += "# TYPE aiops_detector_up gauge\n"
	if d.running {
		result += "aiops_detector_up 1\n"
	} else {
		result += "aiops_detector_up 0\n"
	}

	// Метрики ML pipeline
	result += "# HELP aiops_ml_anomalies_detected_total Total number of anomalies detected\n"
	result += "# TYPE aiops_ml_anomalies_detected_total counter\n"
	result += "aiops_ml_anomalies_detected_total 0\n"

	// Комментарий о Rundeck интеграции
	result += "# Ready for Rundeck orchestrator integration\n"

	return result
}
