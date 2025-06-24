package ml

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"aiops-platform/internal/types"
)

// ModelManager управляет ML моделями для детекции аномалий
type ModelManager struct {
	config       *types.Config
	currentModel *AnomalyModel
	modelInfo    *types.ModelInfo
	modelMutex   sync.RWMutex

	// Настройки
	storagePath  string
	autoSave     bool
	saveInterval time.Duration
}

// AnomalyModel представляет модель детекции аномалий
type AnomalyModel struct {
	Type        string             `json:"type"`
	Parameters  map[string]float64 `json:"parameters"`
	Statistics  *ModelStatistics   `json:"statistics"`
	Thresholds  map[string]float64 `json:"thresholds"`
	TrainedAt   time.Time          `json:"trained_at"`
	SampleCount int                `json:"sample_count"`
}

// ModelStatistics содержит статистики модели
type ModelStatistics struct {
	FeatureMeans []float64 `json:"feature_means"`
	FeatureStds  []float64 `json:"feature_stds"`
	FeatureCount int       `json:"feature_count"`

	// Для Isolation Forest
	Trees     []IsolationTree `json:"trees,omitempty"`
	TreeCount int             `json:"tree_count"`
	MaxDepth  int             `json:"max_depth"`

	// Общие параметры
	Threshold float64   `json:"threshold"`
	TrainTime time.Time `json:"train_time"`
}

// IsolationTree представляет дерево изоляции
type IsolationTree struct {
	Root *TreeNode `json:"root"`
}

// TreeNode узел дерева изоляции
type TreeNode struct {
	FeatureIndex int       `json:"feature_index"`
	SplitValue   float64   `json:"split_value"`
	Left         *TreeNode `json:"left,omitempty"`
	Right        *TreeNode `json:"right,omitempty"`
	IsLeaf       bool      `json:"is_leaf"`
	PathLength   int       `json:"path_length"`
}

// NewModelManager создает новый менеджер моделей
func NewModelManager(config *types.Config) (*ModelManager, error) {
	storagePath := config.ML.StoragePath
	if storagePath == "" {
		storagePath = "models"
	}

	// Создаем директорию для моделей
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create models directory: %w", err)
	}

	manager := &ModelManager{
		config:       config,
		storagePath:  storagePath,
		autoSave:     true,
		saveInterval: 1 * time.Hour,
	}

	return manager, nil
}

// InitializeModel инициализирует новую модель
func (mm *ModelManager) InitializeModel() error {
	mm.modelMutex.Lock()
	defer mm.modelMutex.Unlock()

	modelType := mm.config.ML.ModelType
	if modelType == "" {
		modelType = "isolation_forest"
	}

	mm.currentModel = &AnomalyModel{
		Type:        modelType,
		Parameters:  make(map[string]float64),
		Statistics:  &ModelStatistics{},
		Thresholds:  make(map[string]float64),
		TrainedAt:   time.Now(),
		SampleCount: 0,
	}

	// Инициализируем параметры модели
	switch modelType {
	case "isolation_forest":
		mm.currentModel.Parameters["tree_count"] = 100
		mm.currentModel.Parameters["max_depth"] = 10
		mm.currentModel.Parameters["sample_ratio"] = 0.8
		mm.currentModel.Thresholds["anomaly_threshold"] = mm.config.ML.Threshold
		if mm.currentModel.Thresholds["anomaly_threshold"] == 0 {
			mm.currentModel.Thresholds["anomaly_threshold"] = 0.6
		}
	default:
		return fmt.Errorf("unsupported model type: %s", modelType)
	}

	mm.modelInfo = &types.ModelInfo{
		ID:        fmt.Sprintf("model_%d", time.Now().Unix()),
		Type:      modelType,
		Version:   "1.0",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Accuracy:  0.0,
		Parameters: map[string]interface{}{
			"type": modelType,
		},
		FilePath: filepath.Join(mm.storagePath, "current_model.json"),
	}

	log.Printf("ModelManager: initialized new %s model", modelType)
	return nil
}

// LoadModel загружает модель из файла
func (mm *ModelManager) LoadModel() error {
	mm.modelMutex.Lock()
	defer mm.modelMutex.Unlock()

	modelPath := filepath.Join(mm.storagePath, "current_model.json")

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", modelPath)
	}

	data, err := ioutil.ReadFile(modelPath)
	if err != nil {
		return fmt.Errorf("failed to read model file: %w", err)
	}

	var model AnomalyModel
	if err := json.Unmarshal(data, &model); err != nil {
		return fmt.Errorf("failed to unmarshal model: %w", err)
	}

	mm.currentModel = &model

	// Загружаем информацию о модели
	infoPath := filepath.Join(mm.storagePath, "model_info.json")
	if infoData, err := ioutil.ReadFile(infoPath); err == nil {
		var info types.ModelInfo
		if json.Unmarshal(infoData, &info) == nil {
			mm.modelInfo = &info
		}
	}

	log.Printf("ModelManager: loaded model type %s with %d samples",
		mm.currentModel.Type, mm.currentModel.SampleCount)
	return nil
}

// SaveModel сохраняет модель в файл
func (mm *ModelManager) SaveModel() error {
	mm.modelMutex.RLock()
	defer mm.modelMutex.RUnlock()

	if mm.currentModel == nil {
		return fmt.Errorf("no model to save")
	}

	modelPath := filepath.Join(mm.storagePath, "current_model.json")

	data, err := json.MarshalIndent(mm.currentModel, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal model: %w", err)
	}

	if err := ioutil.WriteFile(modelPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	// Сохраняем информацию о модели
	if mm.modelInfo != nil {
		mm.modelInfo.UpdatedAt = time.Now()
		infoData, _ := json.MarshalIndent(mm.modelInfo, "", "  ")
		infoPath := filepath.Join(mm.storagePath, "model_info.json")
		ioutil.WriteFile(infoPath, infoData, 0644)
	}

	log.Printf("ModelManager: saved model to %s", modelPath)
	return nil
}

// TrainModel обучает модель на новых данных
func (mm *ModelManager) TrainModel(features [][]float64) error {
	mm.modelMutex.Lock()
	defer mm.modelMutex.Unlock()

	if mm.currentModel == nil {
		return fmt.Errorf("model not initialized")
	}

	if len(features) == 0 {
		return fmt.Errorf("no training data provided")
	}

	log.Printf("ModelManager: training model on %d samples", len(features))

	switch mm.currentModel.Type {
	case "isolation_forest":
		return mm.trainIsolationForest(features)
	default:
		return fmt.Errorf("unsupported model type: %s", mm.currentModel.Type)
	}
}

// trainIsolationForest обучает модель Isolation Forest
func (mm *ModelManager) trainIsolationForest(features [][]float64) error {
	if len(features) == 0 {
		return fmt.Errorf("no features provided")
	}

	featureCount := len(features[0])
	treeCount := int(mm.currentModel.Parameters["tree_count"])
	maxDepth := int(mm.currentModel.Parameters["max_depth"])
	sampleRatio := mm.currentModel.Parameters["sample_ratio"]

	// Вычисляем статистики признаков
	mm.currentModel.Statistics.FeatureCount = featureCount
	mm.currentModel.Statistics.FeatureMeans = make([]float64, featureCount)
	mm.currentModel.Statistics.FeatureStds = make([]float64, featureCount)

	// Вычисляем средние и стандартные отклонения
	for i := 0; i < featureCount; i++ {
		sum := 0.0
		for j := 0; j < len(features); j++ {
			if i < len(features[j]) {
				sum += features[j][i]
			}
		}
		mm.currentModel.Statistics.FeatureMeans[i] = sum / float64(len(features))
	}

	for i := 0; i < featureCount; i++ {
		variance := 0.0
		for j := 0; j < len(features); j++ {
			if i < len(features[j]) {
				diff := features[j][i] - mm.currentModel.Statistics.FeatureMeans[i]
				variance += diff * diff
			}
		}
		variance /= float64(len(features) - 1)
		mm.currentModel.Statistics.FeatureStds[i] = math.Sqrt(variance)
	}

	// Обучаем деревья изоляции
	mm.currentModel.Statistics.Trees = make([]IsolationTree, treeCount)
	mm.currentModel.Statistics.TreeCount = treeCount
	mm.currentModel.Statistics.MaxDepth = maxDepth

	sampleSize := int(float64(len(features)) * sampleRatio)
	if sampleSize < 1 {
		sampleSize = len(features)
	}

	for i := 0; i < treeCount; i++ {
		// Случайная выборка данных
		sample := mm.sampleFeatures(features, sampleSize)

		// Строим дерево
		tree := &IsolationTree{
			Root: mm.buildIsolationTree(sample, 0, maxDepth),
		}
		mm.currentModel.Statistics.Trees[i] = *tree
	}

	mm.currentModel.SampleCount = len(features)
	mm.currentModel.TrainedAt = time.Now()
	mm.currentModel.Statistics.TrainTime = time.Now()

	log.Printf("ModelManager: trained isolation forest with %d trees", treeCount)
	return nil
}

// buildIsolationTree строит дерево изоляции рекурсивно
func (mm *ModelManager) buildIsolationTree(features [][]float64, depth, maxDepth int) *TreeNode {
	if len(features) <= 1 || depth >= maxDepth {
		return &TreeNode{
			IsLeaf:     true,
			PathLength: depth,
		}
	}

	// Случайно выбираем признак и значение для разделения
	featureIndex := mm.randomInt(len(features[0]))

	// Находим минимум и максимум для выбранного признака
	minVal, maxVal := mm.getFeatureRange(features, featureIndex)
	if minVal == maxVal {
		return &TreeNode{
			IsLeaf:     true,
			PathLength: depth,
		}
	}

	splitValue := minVal + (maxVal-minVal)*mm.randomFloat()

	// Разделяем данные
	left, right := mm.splitFeatures(features, featureIndex, splitValue)

	return &TreeNode{
		FeatureIndex: featureIndex,
		SplitValue:   splitValue,
		Left:         mm.buildIsolationTree(left, depth+1, maxDepth),
		Right:        mm.buildIsolationTree(right, depth+1, maxDepth),
		IsLeaf:       false,
	}
}

// PredictAnomaly предсказывает аномалию для вектора признаков
func (mm *ModelManager) PredictAnomaly(features []float64) (float64, error) {
	mm.modelMutex.RLock()
	defer mm.modelMutex.RUnlock()

	if mm.currentModel == nil {
		return 0, fmt.Errorf("model not initialized")
	}

	switch mm.currentModel.Type {
	case "isolation_forest":
		return mm.predictIsolationForest(features)
	default:
		return 0, fmt.Errorf("unsupported model type: %s", mm.currentModel.Type)
	}
}

// predictIsolationForest предсказывает аномалию используя Isolation Forest
func (mm *ModelManager) predictIsolationForest(features []float64) (float64, error) {
	if mm.currentModel.Statistics == nil || len(mm.currentModel.Statistics.Trees) == 0 {
		return 0, fmt.Errorf("model not trained")
	}

	// Нормализуем признаки
	normalizedFeatures := make([]float64, len(features))
	for i, feature := range features {
		if i < len(mm.currentModel.Statistics.FeatureMeans) {
			mean := mm.currentModel.Statistics.FeatureMeans[i]
			std := mm.currentModel.Statistics.FeatureStds[i]
			if std > 0 {
				normalizedFeatures[i] = (feature - mean) / std
			} else {
				normalizedFeatures[i] = feature - mean
			}
		} else {
			normalizedFeatures[i] = feature
		}
	}

	// Вычисляем среднюю длину пути по всем деревьям
	totalPathLength := 0.0
	treeCount := len(mm.currentModel.Statistics.Trees)

	for _, tree := range mm.currentModel.Statistics.Trees {
		pathLength := mm.getPathLength(tree.Root, normalizedFeatures, 0)
		totalPathLength += pathLength
	}

	avgPathLength := totalPathLength / float64(treeCount)

	// Вычисляем аномальность (чем короче путь, тем больше аномалия)
	// Нормализуем относительно ожидаемой длины пути
	expectedPathLength := mm.expectedIsolationDepth(mm.currentModel.SampleCount)
	anomalyScore := math.Pow(2, -avgPathLength/expectedPathLength)

	return anomalyScore, nil
}

// getPathLength получает длину пути в дереве изоляции
func (mm *ModelManager) getPathLength(node *TreeNode, features []float64, currentDepth int) float64 {
	if node.IsLeaf {
		return float64(currentDepth)
	}

	if node.FeatureIndex < len(features) {
		if features[node.FeatureIndex] < node.SplitValue {
			return mm.getPathLength(node.Left, features, currentDepth+1)
		} else {
			return mm.getPathLength(node.Right, features, currentDepth+1)
		}
	}

	return float64(currentDepth)
}

// GetStatus возвращает статус менеджера моделей
func (mm *ModelManager) GetStatus() map[string]interface{} {
	mm.modelMutex.RLock()
	defer mm.modelMutex.RUnlock()

	status := map[string]interface{}{
		"model_loaded": mm.currentModel != nil,
		"storage_path": mm.storagePath,
	}

	if mm.currentModel != nil {
		status["model_type"] = mm.currentModel.Type
		status["sample_count"] = mm.currentModel.SampleCount
		status["trained_at"] = mm.currentModel.TrainedAt
	}

	if mm.modelInfo != nil {
		status["model_id"] = mm.modelInfo.ID
		status["model_version"] = mm.modelInfo.Version
		status["accuracy"] = mm.modelInfo.Accuracy
	}

	return status
}

// Вспомогательные функции

func (mm *ModelManager) sampleFeatures(features [][]float64, sampleSize int) [][]float64 {
	if sampleSize >= len(features) {
		return features
	}

	sample := make([][]float64, sampleSize)
	for i := 0; i < sampleSize; i++ {
		idx := mm.randomInt(len(features))
		sample[i] = features[idx]
	}
	return sample
}

func (mm *ModelManager) getFeatureRange(features [][]float64, featureIndex int) (float64, float64) {
	if len(features) == 0 || featureIndex >= len(features[0]) {
		return 0, 0
	}

	min := features[0][featureIndex]
	max := features[0][featureIndex]

	for _, feature := range features {
		if featureIndex < len(feature) {
			if feature[featureIndex] < min {
				min = feature[featureIndex]
			}
			if feature[featureIndex] > max {
				max = feature[featureIndex]
			}
		}
	}

	return min, max
}

func (mm *ModelManager) splitFeatures(features [][]float64, featureIndex int, splitValue float64) ([][]float64, [][]float64) {
	var left, right [][]float64

	for _, feature := range features {
		if featureIndex < len(feature) {
			if feature[featureIndex] < splitValue {
				left = append(left, feature)
			} else {
				right = append(right, feature)
			}
		}
	}

	return left, right
}

func (mm *ModelManager) expectedIsolationDepth(n int) float64 {
	if n <= 1 {
		return 0
	}
	// H(n-1) = ln(n-1) + 0.5772156649 (Euler's constant approximation)
	return 2.0*(math.Log(float64(n-1))+0.5772156649) - 2.0*float64(n-1)/float64(n)
}

// Простые генераторы случайных чисел
var randomSeed int64 = 1

func (mm *ModelManager) randomInt(max int) int {
	randomSeed = randomSeed*1103515245 + 12345
	return int((randomSeed / 65536) % int64(max))
}

func (mm *ModelManager) randomFloat() float64 {
	randomSeed = randomSeed*1103515245 + 12345
	return float64((randomSeed/65536)%32768) / 32768.0
}
