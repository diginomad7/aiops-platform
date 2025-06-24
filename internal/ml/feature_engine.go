package ml

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"aiops-platform/internal/types"
)

// FeatureEngine извлекает признаки из обработанных метрик
type FeatureEngine struct {
	config         *types.Config
	featureHistory []types.FeatureVector
	historyMutex   sync.RWMutex

	// Статистики
	extractedCount int64
	errorCount     int64

	// Настройки
	windowSize    int
	maxHistory    int
	retentionTime time.Duration
}

// NewFeatureEngine создает новый движок извлечения признаков
func NewFeatureEngine(config *types.Config) (*FeatureEngine, error) {
	engine := &FeatureEngine{
		config:         config,
		featureHistory: make([]types.FeatureVector, 0),
		windowSize:     config.ML.FeatureWindow,
		maxHistory:     10000,
		retentionTime:  6 * time.Hour,
	}

	if engine.windowSize <= 0 {
		engine.windowSize = 10
	}

	return engine, nil
}

// Start запускает движок извлечения признаков
func (fe *FeatureEngine) Start(ctx context.Context) error {
	log.Printf("FeatureEngine started")

	// Запускаем периодическую очистку истории
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			fe.cleanupHistory()
		}
	}
}

// ExtractFeatures извлекает признаки из обработанных метрик
func (fe *FeatureEngine) ExtractFeatures(processed *types.ProcessedMetrics) (*types.FeatureVector, error) {
	if processed == nil || len(processed.Data) == 0 {
		return nil, fmt.Errorf("no data provided for feature extraction")
	}

	// Группируем метрики по именам
	grouped := fe.groupMetricsByName(processed.Data)

	// Извлекаем различные типы признаков
	features := make([]float64, 0)
	labels := make(map[string]string)

	// Статистические признаки
	statFeatures := fe.extractStatisticalFeatures(grouped)
	features = append(features, statFeatures...)

	// Временные признаки
	timeFeatures := fe.extractTimeFeatures(processed.Data)
	features = append(features, timeFeatures...)

	// Тренд признаки
	trendFeatures := fe.extractTrendFeatures(grouped)
	features = append(features, trendFeatures...)

	// Корреляционные признаки
	correlationFeatures := fe.extractCorrelationFeatures(grouped)
	features = append(features, correlationFeatures...)

	// Создаем вектор признаков
	featureVector := &types.FeatureVector{
		Values:     features,
		Labels:     labels,
		Timestamp:  processed.Timestamp,
		MetricName: fe.getPrimaryMetricName(processed.Data),
	}

	// Добавляем в историю
	fe.addToHistory(featureVector)

	fe.extractedCount++
	return featureVector, nil
}

// GetStatus возвращает статус движка
func (fe *FeatureEngine) GetStatus() map[string]interface{} {
	fe.historyMutex.RLock()
	historyLen := len(fe.featureHistory)
	fe.historyMutex.RUnlock()

	return map[string]interface{}{
		"feature_history": historyLen,
		"extracted_count": fe.extractedCount,
		"error_count":     fe.errorCount,
		"window_size":     fe.windowSize,
	}
}

// groupMetricsByName группирует метрики по именам
func (fe *FeatureEngine) groupMetricsByName(metrics []types.MetricData) map[string][]types.MetricData {
	grouped := make(map[string][]types.MetricData)

	for _, metric := range metrics {
		grouped[metric.Name] = append(grouped[metric.Name], metric)
	}

	// Сортируем по времени
	for name := range grouped {
		sort.Slice(grouped[name], func(i, j int) bool {
			return grouped[name][i].Timestamp.Before(grouped[name][j].Timestamp)
		})
	}

	return grouped
}

// extractStatisticalFeatures извлекает статистические признаки
func (fe *FeatureEngine) extractStatisticalFeatures(grouped map[string][]types.MetricData) []float64 {
	features := make([]float64, 0)

	for _, metrics := range grouped {
		if len(metrics) == 0 {
			continue
		}

		values := make([]float64, len(metrics))
		for i, m := range metrics {
			values[i] = m.Value
		}

		// Базовые статистики
		mean := fe.calculateMean(values)
		std := fe.calculateStdDev(values, mean)
		min := fe.calculateMin(values)
		max := fe.calculateMax(values)

		features = append(features, mean, std, min, max)

		// Квантили
		q25, median, q75 := fe.calculateQuantiles(values)
		features = append(features, q25, median, q75)

		// Дополнительные метрики
		variance := std * std
		range_ := max - min
		iqr := q75 - q25

		features = append(features, variance, range_, iqr)

		// Асимметрия и эксцесс (упрощенные версии)
		skewness := fe.calculateSkewness(values, mean, std)
		kurtosis := fe.calculateKurtosis(values, mean, std)

		features = append(features, skewness, kurtosis)
	}

	return features
}

// extractTimeFeatures извлекает временные признаки
func (fe *FeatureEngine) extractTimeFeatures(metrics []types.MetricData) []float64 {
	features := make([]float64, 0)

	if len(metrics) <= 1 {
		return append(features, 0, 0, 0) // Пустые временные признаки
	}

	// Интервалы между измерениями
	intervals := make([]float64, len(metrics)-1)
	for i := 1; i < len(metrics); i++ {
		intervals[i-1] = metrics[i].Timestamp.Sub(metrics[i-1].Timestamp).Seconds()
	}

	// Статистики интервалов
	meanInterval := fe.calculateMean(intervals)
	stdInterval := fe.calculateStdDev(intervals, meanInterval)

	features = append(features, meanInterval, stdInterval)

	// Регулярность (коэффициент вариации интервалов)
	regularity := 0.0
	if meanInterval > 0 {
		regularity = stdInterval / meanInterval
	}
	features = append(features, regularity)

	return features
}

// extractTrendFeatures извлекает признаки трендов
func (fe *FeatureEngine) extractTrendFeatures(grouped map[string][]types.MetricData) []float64 {
	features := make([]float64, 0)

	for _, metrics := range grouped {
		if len(metrics) <= 2 {
			features = append(features, 0, 0, 0)
			continue
		}

		values := make([]float64, len(metrics))
		for i, m := range metrics {
			values[i] = m.Value
		}

		// Линейный тренд (простая регрессия)
		slope := fe.calculateSlope(values)

		// Монотонность
		monotonicity := fe.calculateMonotonicity(values)

		// Изменчивость (коэффициент вариации)
		mean := fe.calculateMean(values)
		std := fe.calculateStdDev(values, mean)
		variability := 0.0
		if mean != 0 {
			variability = std / math.Abs(mean)
		}

		features = append(features, slope, monotonicity, variability)
	}

	return features
}

// extractCorrelationFeatures извлекает корреляционные признаки
func (fe *FeatureEngine) extractCorrelationFeatures(grouped map[string][]types.MetricData) []float64 {
	features := make([]float64, 0)

	// Преобразуем в список метрик для корреляции
	metricNames := make([]string, 0, len(grouped))
	for name := range grouped {
		metricNames = append(metricNames, name)
	}

	if len(metricNames) < 2 {
		return append(features, 0) // Нет корреляции
	}

	// Вычисляем корреляцию между первыми двумя метриками
	metrics1 := grouped[metricNames[0]]
	metrics2 := grouped[metricNames[1]]

	correlation := fe.calculateCorrelation(metrics1, metrics2)
	features = append(features, correlation)

	return features
}

// Математические вспомогательные функции

func (fe *FeatureEngine) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (fe *FeatureEngine) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values) - 1)

	return fe.sqrt(variance)
}

func (fe *FeatureEngine) calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (fe *FeatureEngine) calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (fe *FeatureEngine) calculateQuantiles(values []float64) (q25, median, q75 float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	q25 = sorted[n/4]
	median = sorted[n/2]
	q75 = sorted[3*n/4]

	return q25, median, q75
}

func (fe *FeatureEngine) calculateSkewness(values []float64, mean, std float64) float64 {
	if len(values) <= 2 || std == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		normalized := (v - mean) / std
		sum += normalized * normalized * normalized
	}

	return sum / float64(len(values))
}

func (fe *FeatureEngine) calculateKurtosis(values []float64, mean, std float64) float64 {
	if len(values) <= 3 || std == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		normalized := (v - mean) / std
		sum += normalized * normalized * normalized * normalized
	}

	return sum/float64(len(values)) - 3 // Excess kurtosis
}

func (fe *FeatureEngine) calculateSlope(values []float64) float64 {
	n := len(values)
	if n <= 1 {
		return 0
	}

	// Простая линейная регрессия y = ax + b
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	meanX := sumX / float64(n)
	meanY := sumY / float64(n)

	denominator := sumX2 - float64(n)*meanX*meanX
	if denominator == 0 {
		return 0
	}

	slope := (sumXY - float64(n)*meanX*meanY) / denominator
	return slope
}

func (fe *FeatureEngine) calculateMonotonicity(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	increasing := 0
	decreasing := 0

	for i := 1; i < len(values); i++ {
		if values[i] > values[i-1] {
			increasing++
		} else if values[i] < values[i-1] {
			decreasing++
		}
	}

	total := len(values) - 1
	if total == 0 {
		return 0
	}

	// Возвращаем процент монотонности (-1 = убывание, 1 = возрастание, 0 = случайно)
	return (float64(increasing) - float64(decreasing)) / float64(total)
}

func (fe *FeatureEngine) calculateCorrelation(metrics1, metrics2 []types.MetricData) float64 {
	minLen := len(metrics1)
	if len(metrics2) < minLen {
		minLen = len(metrics2)
	}

	if minLen <= 1 {
		return 0
	}

	values1 := make([]float64, minLen)
	values2 := make([]float64, minLen)

	for i := 0; i < minLen; i++ {
		values1[i] = metrics1[i].Value
		values2[i] = metrics2[i].Value
	}

	mean1 := fe.calculateMean(values1)
	mean2 := fe.calculateMean(values2)

	numerator := 0.0
	sum1 := 0.0
	sum2 := 0.0

	for i := 0; i < minLen; i++ {
		diff1 := values1[i] - mean1
		diff2 := values2[i] - mean2
		numerator += diff1 * diff2
		sum1 += diff1 * diff1
		sum2 += diff2 * diff2
	}

	denominator := fe.sqrt(sum1 * sum2)
	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func (fe *FeatureEngine) sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}

	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// Вспомогательные методы

func (fe *FeatureEngine) getPrimaryMetricName(metrics []types.MetricData) string {
	if len(metrics) == 0 {
		return "unknown"
	}
	return metrics[0].Name
}

func (fe *FeatureEngine) addToHistory(feature *types.FeatureVector) {
	fe.historyMutex.Lock()
	defer fe.historyMutex.Unlock()

	fe.featureHistory = append(fe.featureHistory, *feature)

	// Ограничиваем размер истории
	if len(fe.featureHistory) > fe.maxHistory {
		fe.featureHistory = fe.featureHistory[len(fe.featureHistory)-fe.maxHistory:]
	}
}

func (fe *FeatureEngine) cleanupHistory() {
	fe.historyMutex.Lock()
	defer fe.historyMutex.Unlock()

	cutoff := time.Now().Add(-fe.retentionTime)

	var filtered []types.FeatureVector
	for _, feature := range fe.featureHistory {
		if feature.Timestamp.After(cutoff) {
			filtered = append(filtered, feature)
		}
	}

	fe.featureHistory = filtered
	log.Printf("FeatureEngine: cleaned up old features, remaining: %d", len(fe.featureHistory))
}
