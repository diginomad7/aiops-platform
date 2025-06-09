package detector

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"aiops-platform/internal/types"
)

// Detector представляет основной детектор аномалий
type Detector struct {
	config     *types.Config
	server     *http.Server
	processors []Processor
	shutdown   chan struct{}
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

	return detector, nil
}

// Start запускает детектор
func (d *Detector) Start(ctx context.Context) error {
	log.Printf("Starting detector on %s", d.server.Addr)

	// Запускаем процессоры
	for _, processor := range d.processors {
		log.Printf("Starting processor: %s", processor.Name())
		go func(p Processor) {
			if err := p.Start(ctx); err != nil {
				log.Printf("Processor %s error: %v", p.Name(), err)
			}
		}(processor)
	}

	// Запускаем HTTP сервер
	go func() {
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Запускаем основной цикл
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-d.shutdown:
			return nil
		case <-ticker.C:
			// Периодические проверки
			d.performHealthCheck()
		}
	}
}

// Stop останавливает детектор
func (d *Detector) Stop(ctx context.Context) error {
	log.Println("Stopping detector...")

	// Останавливаем процессоры
	for _, processor := range d.processors {
		log.Printf("Stopping processor: %s", processor.Name())
		if err := processor.Stop(ctx); err != nil {
			log.Printf("Error stopping processor %s: %v", processor.Name(), err)
		}
	}

	// Останавливаем HTTP сервер
	if err := d.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	close(d.shutdown)
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
	w.Write([]byte(`{"anomalies":[],"total":0}`))
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
