package ml

import (
	"fmt"
	"log"
	"sync"
	"time"

	"aiops-platform/internal/types"
)

// AnomalyDetector детектирует аномалии используя обученную модель
type AnomalyDetector struct {
	config       *types.Config
	modelManager *ModelManager

	// История детекций
	detectionHistory []DetectionResult
	historyMutex     sync.RWMutex

	// Статистики
	totalDetections int64
	anomaliesFound  int64
	falsePositives  int64

	// Настройки
	maxHistory          int
	confidenceThreshold float64
}

// DetectionResult результат детекции
type DetectionResult struct {
	FeatureVector *types.FeatureVector `json:"feature_vector"`
	AnomalyScore  float64              `json:"anomaly_score"`
	IsAnomaly     bool                 `json:"is_anomaly"`
	Confidence    float64              `json:"confidence"`
	Timestamp     time.Time            `json:"timestamp"`
	Anomalies     []*types.Anomaly     `json:"anomalies,omitempty"`
}

// NewAnomalyDetector создает новый детектор аномалий
func NewAnomalyDetector(config *types.Config, modelManager *ModelManager) (*AnomalyDetector, error) {
	detector := &AnomalyDetector{
		config:              config,
		modelManager:        modelManager,
		detectionHistory:    make([]DetectionResult, 0),
		maxHistory:          1000,
		confidenceThreshold: 0.7,
	}

	return detector, nil
}

// DetectAnomalies детектирует аномалии в векторе признаков
func (ad *AnomalyDetector) DetectAnomalies(featureVector *types.FeatureVector) ([]*types.Anomaly, error) {
	if featureVector == nil || len(featureVector.Values) == 0 {
		return nil, fmt.Errorf("empty feature vector provided")
	}

	// Получаем оценку аномальности от модели
	anomalyScore, err := ad.modelManager.PredictAnomaly(featureVector.Values)
	if err != nil {
		ad.recordDetection(featureVector, 0, false, 0, nil)
		return nil, fmt.Errorf("model prediction failed: %w", err)
	}

	// Определяем порог для детекции аномалий
	threshold := ad.config.ML.Threshold
	if threshold == 0 {
		threshold = 0.6 // Значение по умолчанию
	}

	isAnomaly := anomalyScore > threshold
	confidence := ad.calculateConfidence(anomalyScore, threshold)

	var anomalies []*types.Anomaly

	if isAnomaly {
		// Создаем объект аномалии
		anomaly := ad.createAnomaly(featureVector, anomalyScore, confidence)
		anomalies = append(anomalies, anomaly)

		ad.anomaliesFound++
		log.Printf("AnomalyDetector: detected anomaly with score %.3f for metric %s",
			anomalyScore, featureVector.MetricName)
	}

	// Записываем результат детекции
	ad.recordDetection(featureVector, anomalyScore, isAnomaly, confidence, anomalies)

	ad.totalDetections++
	return anomalies, nil
}

// createAnomaly создает объект аномалии
func (ad *AnomalyDetector) createAnomaly(featureVector *types.FeatureVector, score, confidence float64) *types.Anomaly {
	// Определяем серьезность на основе оценки
	severity := ad.determineSeverity(score)

	// Генерируем ID аномалии
	anomalyID := fmt.Sprintf("anomaly_%s_%d", featureVector.MetricName, time.Now().Unix())

	// Создаем описание
	description := ad.generateDescription(featureVector, score, confidence)

	anomaly := &types.Anomaly{
		ID:          anomalyID,
		Type:        types.MetricAnomaly,
		Severity:    severity,
		Title:       fmt.Sprintf("Metric Anomaly: %s", featureVector.MetricName),
		Description: description,
		Source:      "ml_pipeline",
		Metric:      featureVector.MetricName,
		Value:       score,
		Threshold:   ad.config.ML.Threshold,
		Labels:      featureVector.Labels,
		Metadata: map[string]interface{}{
			"anomaly_score":  score,
			"confidence":     confidence,
			"feature_count":  len(featureVector.Values),
			"detection_time": time.Now(),
			"model_type":     "isolation_forest",
		},
		DetectedAt: featureVector.Timestamp,
		Status:     "active",
	}

	return anomaly
}

// determineSeverity определяет серьезность аномалии
func (ad *AnomalyDetector) determineSeverity(score float64) types.Severity {
	if score >= 0.9 {
		return types.SeverityCritical
	} else if score >= 0.8 {
		return types.SeverityHigh
	} else if score >= 0.7 {
		return types.SeverityMedium
	} else {
		return types.SeverityLow
	}
}

// generateDescription генерирует описание аномалии
func (ad *AnomalyDetector) generateDescription(featureVector *types.FeatureVector, score, confidence float64) string {
	return fmt.Sprintf(
		"Anomalous behavior detected in metric '%s' with score %.3f (confidence: %.1f%%). "+
			"Feature vector contains %d features. "+
			"This indicates potential system performance issues or unusual patterns.",
		featureVector.MetricName,
		score,
		confidence*100,
		len(featureVector.Values),
	)
}

// calculateConfidence вычисляет уверенность в детекции
func (ad *AnomalyDetector) calculateConfidence(score, threshold float64) float64 {
	if score <= threshold {
		// Нормальное поведение - уверенность основана на том, насколько далеко от порога
		distance := threshold - score
		confidence := 0.5 + (distance * 0.5 / threshold)
		if confidence > 1.0 {
			confidence = 1.0
		}
		return confidence
	} else {
		// Аномальное поведение - уверенность основана на превышении порога
		excess := score - threshold
		confidence := 0.5 + (excess * 0.5 / (1.0 - threshold))
		if confidence > 1.0 {
			confidence = 1.0
		}
		return confidence
	}
}

// recordDetection записывает результат детекции в историю
func (ad *AnomalyDetector) recordDetection(featureVector *types.FeatureVector, score float64, isAnomaly bool, confidence float64, anomalies []*types.Anomaly) {
	ad.historyMutex.Lock()
	defer ad.historyMutex.Unlock()

	result := DetectionResult{
		FeatureVector: featureVector,
		AnomalyScore:  score,
		IsAnomaly:     isAnomaly,
		Confidence:    confidence,
		Timestamp:     time.Now(),
		Anomalies:     anomalies,
	}

	ad.detectionHistory = append(ad.detectionHistory, result)

	// Ограничиваем размер истории
	if len(ad.detectionHistory) > ad.maxHistory {
		ad.detectionHistory = ad.detectionHistory[len(ad.detectionHistory)-ad.maxHistory:]
	}
}

// GetDetectionHistory возвращает историю детекций
func (ad *AnomalyDetector) GetDetectionHistory() []DetectionResult {
	ad.historyMutex.RLock()
	defer ad.historyMutex.RUnlock()

	// Возвращаем копию истории
	history := make([]DetectionResult, len(ad.detectionHistory))
	copy(history, ad.detectionHistory)
	return history
}

// GetRecentAnomalies возвращает недавние аномалии
func (ad *AnomalyDetector) GetRecentAnomalies(since time.Time) []*types.Anomaly {
	ad.historyMutex.RLock()
	defer ad.historyMutex.RUnlock()

	var recentAnomalies []*types.Anomaly

	for _, detection := range ad.detectionHistory {
		if detection.Timestamp.After(since) && detection.IsAnomaly {
			recentAnomalies = append(recentAnomalies, detection.Anomalies...)
		}
	}

	return recentAnomalies
}

// GetStatistics возвращает статистики детектора
func (ad *AnomalyDetector) GetStatistics() map[string]interface{} {
	ad.historyMutex.RLock()
	defer ad.historyMutex.RUnlock()

	// Вычисляем статистики за последний период
	lastHour := time.Now().Add(-1 * time.Hour)
	recentDetections := 0
	recentAnomalies := 0

	for _, detection := range ad.detectionHistory {
		if detection.Timestamp.After(lastHour) {
			recentDetections++
			if detection.IsAnomaly {
				recentAnomalies++
			}
		}
	}

	// Вычисляем показатели
	var anomalyRate float64
	if ad.totalDetections > 0 {
		anomalyRate = float64(ad.anomaliesFound) / float64(ad.totalDetections)
	}

	var recentAnomalyRate float64
	if recentDetections > 0 {
		recentAnomalyRate = float64(recentAnomalies) / float64(recentDetections)
	}

	return map[string]interface{}{
		"total_detections":     ad.totalDetections,
		"anomalies_found":      ad.anomaliesFound,
		"false_positives":      ad.falsePositives,
		"anomaly_rate":         anomalyRate,
		"recent_detections":    recentDetections,
		"recent_anomalies":     recentAnomalies,
		"recent_anomaly_rate":  recentAnomalyRate,
		"history_size":         len(ad.detectionHistory),
		"confidence_threshold": ad.confidenceThreshold,
	}
}

// GetStatus возвращает статус детектора
func (ad *AnomalyDetector) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"total_detections":     ad.totalDetections,
		"anomalies_found":      ad.anomaliesFound,
		"history_size":         len(ad.detectionHistory),
		"confidence_threshold": ad.confidenceThreshold,
		"model_available":      ad.modelManager != nil,
	}
}

// UpdateThreshold обновляет порог детекции
func (ad *AnomalyDetector) UpdateThreshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1, got: %f", threshold)
	}

	ad.config.ML.Threshold = threshold
	log.Printf("AnomalyDetector: updated threshold to %.3f", threshold)
	return nil
}

// MarkFalsePositive помечает детекцию как ложноположительную
func (ad *AnomalyDetector) MarkFalsePositive(anomalyID string) error {
	ad.historyMutex.Lock()
	defer ad.historyMutex.Unlock()

	// Ищем аномалию в истории
	for i := range ad.detectionHistory {
		for _, anomaly := range ad.detectionHistory[i].Anomalies {
			if anomaly.ID == anomalyID {
				// Помечаем как ложноположительную
				anomaly.Status = "false_positive"
				anomaly.Metadata["marked_false_positive"] = time.Now()

				ad.falsePositives++
				log.Printf("AnomalyDetector: marked anomaly %s as false positive", anomalyID)
				return nil
			}
		}
	}

	return fmt.Errorf("anomaly with ID %s not found", anomalyID)
}

// EvaluatePerformance оценивает производительность детектора
func (ad *AnomalyDetector) EvaluatePerformance() map[string]float64 {
	ad.historyMutex.RLock()
	defer ad.historyMutex.RUnlock()

	if len(ad.detectionHistory) == 0 {
		return map[string]float64{
			"precision": 0,
			"recall":    0,
			"f1_score":  0,
		}
	}

	// Вычисляем метрики производительности
	truePositives := float64(ad.anomaliesFound - ad.falsePositives)
	falsePositives := float64(ad.falsePositives)
	totalDetections := float64(ad.totalDetections)

	// Простая оценка precision (точность)
	var precision float64
	if ad.anomaliesFound > 0 {
		precision = truePositives / float64(ad.anomaliesFound)
	}

	// Простая оценка recall (полнота) - предполагаем что нашли большинство аномалий
	recall := truePositives / (truePositives + falsePositives + 1) // +1 для избежания деления на 0

	// F1 Score
	var f1Score float64
	if precision+recall > 0 {
		f1Score = 2 * (precision * recall) / (precision + recall)
	}

	return map[string]float64{
		"precision":           precision,
		"recall":              recall,
		"f1_score":            f1Score,
		"anomaly_rate":        float64(ad.anomaliesFound) / totalDetections,
		"false_positive_rate": falsePositives / totalDetections,
	}
}

// CleanupHistory очищает старую историю
func (ad *AnomalyDetector) CleanupHistory(maxAge time.Duration) {
	ad.historyMutex.Lock()
	defer ad.historyMutex.Unlock()

	cutoff := time.Now().Add(-maxAge)

	var filtered []DetectionResult
	for _, detection := range ad.detectionHistory {
		if detection.Timestamp.After(cutoff) {
			filtered = append(filtered, detection)
		}
	}

	oldSize := len(ad.detectionHistory)
	ad.detectionHistory = filtered

	log.Printf("AnomalyDetector: cleaned up detection history, removed %d old entries",
		oldSize-len(ad.detectionHistory))
}
