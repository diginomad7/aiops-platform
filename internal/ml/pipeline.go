package ml

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"aiops-platform/internal/types"
)

// Pipeline представляет ML pipeline для детекции аномалий
type Pipeline struct {
	config          *types.Config
	dataProcessor   *DataProcessor
	featureEngine   *FeatureEngine
	modelManager    *ModelManager
	anomalyDetector *AnomalyDetector
	metrics         *PipelineMetrics
	rundeckClient   *RundeckClient
	mu              sync.RWMutex
	running         bool
}

// PipelineMetrics содержит метрики работы pipeline
type PipelineMetrics struct {
	ProcessedSamples  int64     `json:"processed_samples"`
	DetectedAnomalies int64     `json:"detected_anomalies"`
	ProcessingLatency float64   `json:"processing_latency_ms"`
	ModelAccuracy     float64   `json:"model_accuracy"`
	LastProcessedTime time.Time `json:"last_processed_time"`
	ErrorCount        int64     `json:"error_count"`
}

// NewPipeline создает новый ML pipeline
func NewPipeline(config *types.Config) (*Pipeline, error) {
	pipeline := &Pipeline{
		config:  config,
		metrics: &PipelineMetrics{},
	}

	// Инициализируем компоненты
	var err error

	pipeline.dataProcessor, err = NewDataProcessor(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create data processor: %w", err)
	}

	pipeline.featureEngine, err = NewFeatureEngine(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create feature engine: %w", err)
	}

	pipeline.modelManager, err = NewModelManager(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create model manager: %w", err)
	}

	pipeline.anomalyDetector, err = NewAnomalyDetector(config, pipeline.modelManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create anomaly detector: %w", err)
	}

	// Initialize Rundeck client if configured
	if config.Rundeck.URL != "" {
		pipeline.rundeckClient = NewRundeckClient(
			config.Rundeck.URL,
			config.Rundeck.Username,
			config.Rundeck.Password,
		)

		// Test connection to Rundeck
		if err := pipeline.rundeckClient.HealthCheck(); err != nil {
			log.Printf("Warning: Rundeck health check failed: %v", err)
		} else {
			log.Printf("Successfully connected to Rundeck orchestrator")
		}
	}

	return pipeline, nil
}

// Start запускает ML pipeline
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("pipeline already running")
	}

	log.Printf("Starting ML pipeline...")

	// Загружаем предобученную модель или инициализируем новую
	if err := p.modelManager.LoadModel(); err != nil {
		log.Printf("Failed to load existing model, creating new one: %v", err)
		if err := p.modelManager.InitializeModel(); err != nil {
			return fmt.Errorf("failed to initialize model: %w", err)
		}
	}

	p.running = true

	// Запускаем компоненты
	go p.dataProcessor.Start(ctx)
	go p.featureEngine.Start(ctx)

	// Если включено обучение, запускаем процесс обучения
	if p.config.ML.TrainingEnabled {
		go p.startTrainingLoop(ctx)
	}

	// Запускаем основной цикл обработки
	go p.processingLoop(ctx)

	log.Printf("ML pipeline started successfully")
	return nil
}

// Stop останавливает ML pipeline
func (p *Pipeline) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	log.Printf("Stopping ML pipeline...")

	// Сохраняем модель
	if err := p.modelManager.SaveModel(); err != nil {
		log.Printf("Failed to save model: %v", err)
	}

	p.running = false
	log.Printf("ML pipeline stopped")
	return nil
}

// ProcessMetrics обрабатывает метрики и возвращает результат детекции аномалий
func (p *Pipeline) ProcessMetrics(metrics []types.MetricData) ([]*types.Anomaly, error) {
	start := time.Now()
	defer func() {
		p.metrics.ProcessingLatency = float64(time.Since(start).Nanoseconds()) / 1e6
		p.metrics.LastProcessedTime = time.Now()
		p.metrics.ProcessedSamples++
	}()

	// Предобработка данных
	processedData, err := p.dataProcessor.ProcessMetrics(metrics)
	if err != nil {
		p.metrics.ErrorCount++
		return nil, fmt.Errorf("data processing failed: %w", err)
	}

	// Извлечение признаков
	features, err := p.featureEngine.ExtractFeatures(processedData)
	if err != nil {
		p.metrics.ErrorCount++
		return nil, fmt.Errorf("feature extraction failed: %w", err)
	}

	// Детекция аномалий
	anomalies, err := p.anomalyDetector.DetectAnomalies(features)
	if err != nil {
		p.metrics.ErrorCount++
		return nil, fmt.Errorf("anomaly detection failed: %w", err)
	}

	p.metrics.DetectedAnomalies += int64(len(anomalies))

	// Trigger automatic remediation for critical/high severity anomalies
	if p.rundeckClient != nil && len(anomalies) > 0 {
		go p.triggerRemediation(anomalies)
	}

	return anomalies, nil
}

// triggerRemediation triggers automated remediation through Rundeck
func (p *Pipeline) triggerRemediation(anomalies []*types.Anomaly) {
	// Filter for high severity anomalies that require immediate action
	actionableAnomalies := make([]types.Anomaly, 0)

	for _, anomaly := range anomalies {
		if anomaly.Severity == types.SeverityCritical || anomaly.Severity == types.SeverityHigh {
			// Convert pointer to value for Rundeck integration
			actionableAnomalies = append(actionableAnomalies, *anomaly)
		}
	}

	if len(actionableAnomalies) == 0 {
		return
	}

	log.Printf("Triggering remediation for %d high-severity anomalies", len(actionableAnomalies))

	// Use batch remediation for efficiency
	if err := p.rundeckClient.BatchTriggerRemediation(actionableAnomalies); err != nil {
		log.Printf("Failed to trigger Rundeck remediation: %v", err)
	} else {
		log.Printf("Successfully triggered Rundeck remediation for %d anomalies", len(actionableAnomalies))
	}
}

// GetMetrics возвращает метрики pipeline
func (p *Pipeline) GetMetrics() *PipelineMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Создаем копию метрик
	metrics := *p.metrics
	return &metrics
}

// processingLoop основной цикл обработки
func (p *Pipeline) processingLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !p.running {
				return
			}
			// Периодическая обработка буферизованных данных
			p.processBufferedData()
		}
	}
}

// startTrainingLoop запускает цикл обучения модели
func (p *Pipeline) startTrainingLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Переобучение раз в день
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !p.running {
				return
			}
			log.Printf("Starting model retraining...")
			if err := p.retrainModel(); err != nil {
				log.Printf("Model retraining failed: %v", err)
			} else {
				log.Printf("Model retraining completed successfully")
			}
		}
	}
}

// processBufferedData обрабатывает буферизованные данные
func (p *Pipeline) processBufferedData() {
	bufferedData := p.dataProcessor.GetBufferedData()
	if len(bufferedData) == 0 {
		return
	}

	log.Printf("Processing %d buffered metrics", len(bufferedData))

	_, err := p.ProcessMetrics(bufferedData)
	if err != nil {
		log.Printf("Failed to process buffered data: %v", err)
	}
}

// retrainModel переобучает модель на новых данных
func (p *Pipeline) retrainModel() error {
	// Собираем обучающие данные за последнее время
	trainingData := p.dataProcessor.GetTrainingData()
	if len(trainingData) < 100 { // Минимум данных для обучения
		return fmt.Errorf("insufficient training data: %d samples", len(trainingData))
	}

	// Извлекаем признаки
	features := make([][]float64, len(trainingData))
	for i, data := range trainingData {
		feat, err := p.featureEngine.ExtractFeatures(&data)
		if err != nil {
			return fmt.Errorf("feature extraction for training failed: %w", err)
		}
		features[i] = feat.Values
	}

	// Переобучаем модель
	if err := p.modelManager.TrainModel(features); err != nil {
		return fmt.Errorf("model training failed: %w", err)
	}

	// Обновляем метрики точности
	accuracy, err := p.evaluateModel(features)
	if err != nil {
		log.Printf("Failed to evaluate model: %v", err)
	} else {
		p.metrics.ModelAccuracy = accuracy
	}

	return nil
}

// evaluateModel оценивает точность модели
func (p *Pipeline) evaluateModel(features [][]float64) (float64, error) {
	if len(features) == 0 {
		return 0, fmt.Errorf("no features for evaluation")
	}

	// Простая оценка - процент обнаруженных аномалий в разумных пределах
	totalSamples := len(features)
	anomalyCount := 0

	for _, feature := range features {
		feat := &types.FeatureVector{
			Values:    feature,
			Timestamp: time.Now(),
		}

		anomalies, err := p.anomalyDetector.DetectAnomalies(feat)
		if err != nil {
			continue
		}

		if len(anomalies) > 0 {
			anomalyCount++
		}
	}

	// Хорошая точность - обнаружение 1-10% аномалий
	anomalyRate := float64(anomalyCount) / float64(totalSamples)
	if anomalyRate >= 0.01 && anomalyRate <= 0.1 {
		return 0.9, nil // Высокая точность
	} else if anomalyRate <= 0.15 {
		return 0.7, nil // Средняя точность
	} else {
		return 0.5, nil // Низкая точность
	}
}

// GetHealth возвращает состояние здоровья pipeline
func (p *Pipeline) GetHealth() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	health := map[string]interface{}{
		"running":            p.running,
		"processed_samples":  p.metrics.ProcessedSamples,
		"detected_anomalies": p.metrics.DetectedAnomalies,
		"model_accuracy":     p.metrics.ModelAccuracy,
		"error_count":        p.metrics.ErrorCount,
		"last_processed":     p.metrics.LastProcessedTime,
	}

	// Добавляем статус компонентов
	health["components"] = map[string]interface{}{
		"data_processor":   p.dataProcessor.GetStatus(),
		"feature_engine":   p.featureEngine.GetStatus(),
		"model_manager":    p.modelManager.GetStatus(),
		"anomaly_detector": p.anomalyDetector.GetStatus(),
	}

	return health
}

// ExportMetrics экспортирует метрики в формате Prometheus
func (p *Pipeline) ExportMetrics() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metrics := fmt.Sprintf(`# HELP aiops_ml_processed_samples_total Total processed samples
# TYPE aiops_ml_processed_samples_total counter
aiops_ml_processed_samples_total %d

# HELP aiops_ml_detected_anomalies_total Total detected anomalies
# TYPE aiops_ml_detected_anomalies_total counter
aiops_ml_detected_anomalies_total %d

# HELP aiops_ml_processing_latency_ms Processing latency in milliseconds
# TYPE aiops_ml_processing_latency_ms gauge
aiops_ml_processing_latency_ms %f

# HELP aiops_ml_model_accuracy Model accuracy score
# TYPE aiops_ml_model_accuracy gauge
aiops_ml_model_accuracy %f

# HELP aiops_ml_error_count_total Total error count
# TYPE aiops_ml_error_count_total counter
aiops_ml_error_count_total %d

# HELP aiops_ml_pipeline_running Pipeline running status
# TYPE aiops_ml_pipeline_running gauge
aiops_ml_pipeline_running %d
`,
		p.metrics.ProcessedSamples,
		p.metrics.DetectedAnomalies,
		p.metrics.ProcessingLatency,
		p.metrics.ModelAccuracy,
		p.metrics.ErrorCount,
		func() int {
			if p.running {
				return 1
			}
			return 0
		}())

	return metrics
}
