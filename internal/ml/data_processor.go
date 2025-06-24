package ml

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"

	"aiops-platform/internal/types"
)

// DataProcessor обрабатывает и подготавливает метрические данные для ML pipeline
type DataProcessor struct {
	config      *types.Config
	buffer      []types.MetricData
	bufferMutex sync.RWMutex

	// Хранилище для обучающих данных
	trainingData  []types.ProcessedMetrics
	trainingMutex sync.RWMutex

	// Статистики
	processedCount int64
	errorCount     int64

	// Настройки
	bufferSize    int
	windowSize    int
	retentionTime time.Duration
}

// NewDataProcessor создает новый обработчик данных
func NewDataProcessor(config *types.Config) (*DataProcessor, error) {
	processor := &DataProcessor{
		config:        config,
		buffer:        make([]types.MetricData, 0),
		trainingData:  make([]types.ProcessedMetrics, 0),
		bufferSize:    1000,
		windowSize:    config.ML.FeatureWindow,
		retentionTime: 24 * time.Hour,
	}

	if processor.windowSize <= 0 {
		processor.windowSize = 10 // Значение по умолчанию
	}

	return processor, nil
}

// Start запускает обработчик данных
func (dp *DataProcessor) Start(ctx context.Context) error {
	log.Printf("DataProcessor started")

	// Запускаем периодическую очистку старых данных
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			dp.cleanupOldData()
		}
	}
}

// ProcessMetrics обрабатывает метрики и добавляет их в буфер
func (dp *DataProcessor) ProcessMetrics(metrics []types.MetricData) (*types.ProcessedMetrics, error) {
	if len(metrics) == 0 {
		return nil, nil
	}

	// Добавляем метрики в буфер
	dp.bufferMutex.Lock()
	dp.buffer = append(dp.buffer, metrics...)

	// Ограничиваем размер буфера
	if len(dp.buffer) > dp.bufferSize {
		dp.buffer = dp.buffer[len(dp.buffer)-dp.bufferSize:]
	}
	dp.bufferMutex.Unlock()

	// Нормализуем данные
	normalizedMetrics := dp.normalizeMetrics(metrics)

	// Применяем фильтрацию
	filteredMetrics := dp.filterMetrics(normalizedMetrics)

	// Создаем обработанные метрики
	processed := &types.ProcessedMetrics{
		Data:       filteredMetrics,
		Timestamp:  time.Now(),
		WindowSize: len(filteredMetrics),
	}

	// Сохраняем для обучения
	dp.saveForTraining(processed)

	dp.processedCount++
	return processed, nil
}

// GetBufferedData возвращает буферизованные данные
func (dp *DataProcessor) GetBufferedData() []types.MetricData {
	dp.bufferMutex.RLock()
	defer dp.bufferMutex.RUnlock()

	// Возвращаем копию буфера
	result := make([]types.MetricData, len(dp.buffer))
	copy(result, dp.buffer)

	// Очищаем буфер
	dp.buffer = dp.buffer[:0]

	return result
}

// GetTrainingData возвращает данные для обучения
func (dp *DataProcessor) GetTrainingData() []types.ProcessedMetrics {
	dp.trainingMutex.RLock()
	defer dp.trainingMutex.RUnlock()

	// Возвращаем копию обучающих данных
	result := make([]types.ProcessedMetrics, len(dp.trainingData))
	copy(result, dp.trainingData)

	return result
}

// GetStatus возвращает статус процессора
func (dp *DataProcessor) GetStatus() map[string]interface{} {
	dp.bufferMutex.RLock()
	bufferLen := len(dp.buffer)
	dp.bufferMutex.RUnlock()

	dp.trainingMutex.RLock()
	trainingLen := len(dp.trainingData)
	dp.trainingMutex.RUnlock()

	return map[string]interface{}{
		"buffer_size":     bufferLen,
		"training_data":   trainingLen,
		"processed_count": dp.processedCount,
		"error_count":     dp.errorCount,
		"window_size":     dp.windowSize,
	}
}

// normalizeMetrics нормализует значения метрик
func (dp *DataProcessor) normalizeMetrics(metrics []types.MetricData) []types.MetricData {
	if len(metrics) == 0 {
		return metrics
	}

	// Группируем метрики по именам
	grouped := make(map[string][]types.MetricData)
	for _, metric := range metrics {
		grouped[metric.Name] = append(grouped[metric.Name], metric)
	}

	var normalized []types.MetricData

	for metricName, metricGroup := range grouped {
		// Вычисляем статистики для нормализации
		values := make([]float64, len(metricGroup))
		for i, m := range metricGroup {
			values[i] = m.Value
		}

		mean, stddev := dp.calculateStats(values)

		// Применяем Z-score нормализацию, если stddev > 0
		for _, metric := range metricGroup {
			normalizedValue := metric.Value
			if stddev > 0 {
				normalizedValue = (metric.Value - mean) / stddev
			}

			normalized = append(normalized, types.MetricData{
				Name:      metricName,
				Value:     normalizedValue,
				Timestamp: metric.Timestamp,
				Labels:    metric.Labels,
			})
		}
	}

	return normalized
}

// filterMetrics фильтрует метрики от выбросов
func (dp *DataProcessor) filterMetrics(metrics []types.MetricData) []types.MetricData {
	if len(metrics) <= 3 {
		return metrics // Слишком мало данных для фильтрации
	}

	// Группируем по именам метрик
	grouped := make(map[string][]types.MetricData)
	for _, metric := range metrics {
		grouped[metric.Name] = append(grouped[metric.Name], metric)
	}

	var filtered []types.MetricData

	for _, metricGroup := range grouped {
		// Фильтруем выбросы для каждой группы метрик
		filteredGroup := dp.removeOutliers(metricGroup)
		filtered = append(filtered, filteredGroup...)
	}

	return filtered
}

// removeOutliers удаляет выбросы используя IQR метод
func (dp *DataProcessor) removeOutliers(metrics []types.MetricData) []types.MetricData {
	if len(metrics) <= 3 {
		return metrics
	}

	// Извлекаем значения и сортируем
	values := make([]float64, len(metrics))
	for i, metric := range metrics {
		values[i] = metric.Value
	}
	sort.Float64s(values)

	// Вычисляем квартили
	n := len(values)
	q1 := values[n/4]
	q3 := values[3*n/4]
	iqr := q3 - q1

	// Границы для выбросов
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	// Фильтруем метрики
	var filtered []types.MetricData
	for _, metric := range metrics {
		if metric.Value >= lowerBound && metric.Value <= upperBound {
			filtered = append(filtered, metric)
		}
	}

	return filtered
}

// calculateStats вычисляет среднее и стандартное отклонение
func (dp *DataProcessor) calculateStats(values []float64) (mean, stddev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// Вычисляем среднее
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Вычисляем стандартное отклонение
	if len(values) <= 1 {
		return mean, 0
	}

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values) - 1)
	stddev = variance
	if variance > 0 {
		// Простое приближение квадратного корня
		stddev = dp.sqrt(variance)
	}

	return mean, stddev
}

// sqrt вычисляет квадратный корень методом Ньютона
func (dp *DataProcessor) sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}

	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// saveForTraining сохраняет данные для обучения
func (dp *DataProcessor) saveForTraining(processed *types.ProcessedMetrics) {
	dp.trainingMutex.Lock()
	defer dp.trainingMutex.Unlock()

	dp.trainingData = append(dp.trainingData, *processed)

	// Ограничиваем размер обучающих данных
	maxTrainingData := 10000
	if len(dp.trainingData) > maxTrainingData {
		dp.trainingData = dp.trainingData[len(dp.trainingData)-maxTrainingData:]
	}
}

// cleanupOldData очищает старые данные
func (dp *DataProcessor) cleanupOldData() {
	dp.trainingMutex.Lock()
	defer dp.trainingMutex.Unlock()

	cutoff := time.Now().Add(-dp.retentionTime)

	// Фильтруем старые данные
	var filtered []types.ProcessedMetrics
	for _, data := range dp.trainingData {
		if data.Timestamp.After(cutoff) {
			filtered = append(filtered, data)
		}
	}

	dp.trainingData = filtered
	log.Printf("DataProcessor: cleaned up old data, remaining: %d entries", len(dp.trainingData))
}
