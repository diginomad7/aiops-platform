# AIOps ML Pipeline Documentation

## Overview

The AIOps ML Pipeline is a comprehensive machine learning system designed for real-time anomaly detection in IT infrastructure. It processes metrics from various sources, extracts meaningful features, and uses trained models to identify anomalous behavior patterns.

## Architecture

### Core Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Data          │    │   Feature        │    │   Model         │
│   Processor     │───▶│   Engine         │───▶│   Manager       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Anomaly       │    │   Pipeline       │    │   Monitoring    │
│   Detector      │◀───│   Orchestrator   │───▶│   & Metrics     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### 1. Data Processor

**Purpose**: Preprocesses raw metrics for ML consumption

**Features**:
- Data normalization using Z-score standardization
- Outlier detection and filtering using IQR method
- Buffering for batch processing
- Training data collection and retention

**Key Methods**:
- `ProcessMetrics()`: Normalizes and filters incoming metrics
- `GetBufferedData()`: Returns processed data for ML pipeline
- `GetTrainingData()`: Provides historical data for model retraining

### 2. Feature Engine

**Purpose**: Extracts meaningful features from processed metrics

**Feature Types**:
- **Statistical Features**: Mean, std deviation, min, max, quantiles, skewness, kurtosis
- **Temporal Features**: Time intervals, regularity patterns
- **Trend Features**: Linear trends, monotonicity, variability
- **Correlation Features**: Cross-metric correlations

**Feature Vector Example**:
```json
{
  "values": [0.85, 0.12, 0.95, 0.23, ...],
  "labels": {"metric": "cpu_usage", "instance": "web-01"},
  "timestamp": "2024-01-15T14:30:00Z",
  "metric_name": "cpu_usage"
}
```

### 3. Model Manager

**Purpose**: Manages ML model lifecycle

**Supported Models**:
- **Isolation Forest**: Unsupervised anomaly detection using tree isolation

**Features**:
- Model training and retraining
- Model persistence and versioning
- Performance evaluation
- Model metadata management

**Model Configuration**:
```yaml
ml:
  model_type: "isolation_forest"
  storage_path: "/data/models"
  training_enabled: true
  threshold: 0.6
  feature_window: 20
```

### 4. Anomaly Detector

**Purpose**: Detects anomalies using trained models

**Detection Process**:
1. Receive feature vector
2. Get anomaly score from model
3. Compare against threshold
4. Generate anomaly object if detected
5. Record detection history

**Anomaly Object**:
```json
{
  "id": "anomaly_cpu_usage_1642261800",
  "type": "metric",
  "severity": "high",
  "title": "Metric Anomaly: cpu_usage",
  "description": "Anomalous behavior detected...",
  "source": "ml_pipeline",
  "metric": "cpu_usage",
  "value": 0.87,
  "threshold": 0.6,
  "detected_at": "2024-01-15T14:30:00Z",
  "status": "active"
}
```

## Configuration

### ML Configuration Options

```yaml
ml:
  model_path: "models/current_model.json"
  training_enabled: true
  threshold: 0.6
  feature_window: 20
  model_type: "isolation_forest"
  storage_path: "models"
```

**Parameters**:
- `model_path`: Path to the current model file
- `training_enabled`: Enable automatic model retraining
- `threshold`: Anomaly detection threshold (0.0-1.0)
- `feature_window`: Number of data points for feature extraction
- `model_type`: Type of ML model ("isolation_forest")
- `storage_path`: Directory for model storage

### Isolation Forest Parameters

```go
Parameters: map[string]float64{
  "tree_count":    100,   // Number of isolation trees
  "max_depth":     10,    // Maximum tree depth
  "sample_ratio":  0.8,   // Sample ratio for training
}
```

## Deployment

### Kubernetes Deployment

The ML pipeline is deployed as a containerized service in Kubernetes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aiops-ml-detector
  namespace: monitoring
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: aiops-detector
        image: aiops-detector:latest
        ports:
        - containerPort: 8080  # HTTP API
        - containerPort: 8081  # Metrics
        volumeMounts:
        - name: model-storage
          mountPath: /data/models
```

### Storage Requirements

- **Persistent Volume**: 5Gi for model storage
- **Memory**: 1Gi limit, 256Mi request
- **CPU**: 500m limit, 100m request

## API Endpoints

### Health Check
```
GET /health
```
Returns the health status of the ML pipeline.

### Metrics
```
GET /metrics
```
Returns Prometheus-formatted metrics.

### Pipeline Status
```
GET /ml/status
```
Returns detailed ML pipeline status.

## Metrics

The ML pipeline exposes the following Prometheus metrics:

### Core Metrics
- `aiops_ml_pipeline_running`: Pipeline running status (0/1)
- `aiops_ml_processed_samples_total`: Total processed samples
- `aiops_ml_detected_anomalies_total`: Total detected anomalies
- `aiops_ml_processing_latency_ms`: Processing latency in milliseconds
- `aiops_ml_model_accuracy`: Model accuracy score (0.0-1.0)
- `aiops_ml_error_count_total`: Total error count

### Component Metrics
- Data processor buffer size
- Feature extraction count
- Model training status
- Detection history size

## Monitoring and Alerting

### Grafana Dashboard

The ML pipeline includes a comprehensive Grafana dashboard showing:
- Pipeline status and health
- Processing performance metrics
- Anomaly detection rates
- Model accuracy trends
- Component health status

### Prometheus Alerts

Recommended alert rules:

```yaml
- alert: MLPipelineDown
  expr: aiops_ml_pipeline_running == 0
  for: 1m
  annotations:
    summary: "ML Pipeline is down"

- alert: HighMLProcessingLatency
  expr: aiops_ml_processing_latency_ms > 1000
  for: 5m
  annotations:
    summary: "ML processing latency is high"

- alert: LowModelAccuracy
  expr: aiops_ml_model_accuracy < 0.7
  for: 10m
  annotations:
    summary: "ML model accuracy is low"
```

## Performance Tuning

### Feature Engineering
- Adjust `feature_window` based on data frequency
- Monitor feature extraction latency
- Consider feature selection for high-dimensional data

### Model Training
- Tune isolation forest parameters:
  - `tree_count`: More trees = better accuracy, higher memory
  - `max_depth`: Controls overfitting vs underfitting
  - `sample_ratio`: Affects training time and accuracy

### Data Processing
- Adjust buffer sizes for throughput
- Configure outlier detection sensitivity
- Set appropriate data retention periods

## Troubleshooting

### Common Issues

1. **Model Not Loading**
   - Check model file permissions
   - Verify storage mount
   - Review model file format

2. **High Processing Latency**
   - Reduce feature window size
   - Optimize data preprocessing
   - Check resource limits

3. **Low Detection Accuracy**
   - Retrain model with recent data
   - Adjust detection threshold
   - Review feature engineering

### Debugging

Enable debug logging:
```yaml
logging:
  level: "debug"
```

Check component status:
```bash
kubectl exec -n monitoring <pod-name> -- curl http://localhost:8080/ml/status
```

Review metrics:
```bash
kubectl exec -n monitoring <pod-name> -- curl http://localhost:8081/metrics
```

## Testing

### Automated Testing

Run the ML pipeline test suite:
```bash
./scripts/test-ml-pipeline.sh
```

### Manual Testing

1. Deploy ML components
2. Verify metrics collection
3. Send test data
4. Check anomaly detection
5. Validate model persistence

## Security Considerations

### Data Protection
- Metrics data is processed in-memory
- Models are stored in persistent volumes
- No sensitive data is logged

### Access Control
- RBAC for Kubernetes resources
- Service account isolation
- Network policies for pod communication

### Container Security
- Non-root user execution
- Read-only root filesystem
- Capability dropping
- Security context enforcement

## Future Enhancements

### Planned Features
- Multiple model types (LSTM, SVM)
- Online learning capabilities
- Federated learning support
- Advanced feature engineering
- Model explainability features

### Integration Roadmap
- Real-time streaming processing
- Advanced correlation analysis
- Predictive maintenance
- Capacity planning integration 