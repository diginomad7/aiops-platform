package ml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aiops-platform/internal/types"
)

// RundeckClient handles communication with Rundeck orchestrator
type RundeckClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// RundeckJobExecution represents a job execution request
type RundeckJobExecution struct {
	JobID   string            `json:"jobId"`
	Options map[string]string `json:"options,omitempty"`
}

// RundeckResponse represents the response from Rundeck API
type RundeckResponse struct {
	ID          int    `json:"id"`
	Status      string `json:"status"`
	Project     string `json:"project"`
	ExecutionID string `json:"executionId"`
}

// NewRundeckClient creates a new Rundeck client instance
func NewRundeckClient(baseURL, username, password string) *RundeckClient {
	return &RundeckClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// TriggerRemediation sends anomaly data to Rundeck for remediation execution
func (rc *RundeckClient) TriggerRemediation(anomaly types.Anomaly) error {
	// Map anomaly to appropriate Rundeck job
	jobID := rc.getJobIDForAnomaly(anomaly)
	if jobID == "" {
		return fmt.Errorf("no suitable job found for anomaly type: %s", anomaly.Type)
	}

	// Extract resource name and namespace from labels or metadata
	resourceName := ""
	namespace := "default"

	// Try to get resource name from labels
	if name, ok := anomaly.Labels["pod"]; ok {
		resourceName = name
	} else if name, ok := anomaly.Labels["deployment"]; ok {
		resourceName = name
	} else if name, ok := anomaly.Labels["service"]; ok {
		resourceName = name
	} else {
		resourceName = anomaly.Source // Use source as fallback
	}

	// Try to get namespace from labels
	if ns, ok := anomaly.Labels["namespace"]; ok {
		namespace = ns
	}

	// Prepare job execution options
	options := map[string]string{
		"anomaly_type":  string(anomaly.Type),
		"resource_name": resourceName,
		"namespace":     namespace,
		"severity":      string(anomaly.Severity),
		"anomaly_id":    anomaly.ID,
		"metric":        anomaly.Metric,
		"description":   anomaly.Description,
	}

	// Execute job
	return rc.executeJob(jobID, options)
}

// getJobIDForAnomaly maps anomaly types to Rundeck job IDs
func (rc *RundeckClient) getJobIDForAnomaly(anomaly types.Anomaly) string {
	switch anomaly.Type {
	case "cpu", "memory", "network", "disk":
		return "ml-triggered-remediation"
	case "pod_restart":
		return "restart-high-cpu-pod"
	case "scale_deployment":
		return "scale-high-memory-deployment"
	default:
		return "ml-triggered-remediation" // Default to generic job
	}
}

// executeJob executes a Rundeck job with given options
func (rc *RundeckClient) executeJob(jobID string, options map[string]string) error {
	url := fmt.Sprintf("%s/api/18/job/%s/run", rc.baseURL, jobID)

	// Prepare request payload
	execution := RundeckJobExecution{
		JobID:   jobID,
		Options: options,
	}

	jsonData, err := json.Marshal(execution)
	if err != nil {
		return fmt.Errorf("failed to marshal job execution data: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(rc.username, rc.password)

	// Execute request
	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rundeck API returned status %d", resp.StatusCode)
	}

	// Parse response
	var rundeckResp RundeckResponse
	if err := json.NewDecoder(resp.Body).Decode(&rundeckResp); err != nil {
		return fmt.Errorf("failed to decode Rundeck response: %w", err)
	}

	fmt.Printf("Successfully triggered Rundeck job %s, execution ID: %s\n", jobID, rundeckResp.ExecutionID)
	return nil
}

// HealthCheck verifies connectivity to Rundeck
func (rc *RundeckClient) HealthCheck() error {
	url := fmt.Sprintf("%s/api/18/system/info", rc.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.SetBasicAuth(rc.username, rc.password)

	resp, err := rc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rundeck health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// BatchTriggerRemediation handles multiple anomalies efficiently
func (rc *RundeckClient) BatchTriggerRemediation(anomalies []types.Anomaly) error {
	if len(anomalies) == 0 {
		return nil
	}

	// Group anomalies by severity to prioritize critical ones
	critical := []types.Anomaly{}
	high := []types.Anomaly{}
	others := []types.Anomaly{}

	for _, anomaly := range anomalies {
		switch anomaly.Severity {
		case "critical":
			critical = append(critical, anomaly)
		case "high":
			high = append(high, anomaly)
		default:
			others = append(others, anomaly)
		}
	}

	// Process in priority order
	for _, group := range [][]types.Anomaly{critical, high, others} {
		for _, anomaly := range group {
			if err := rc.TriggerRemediation(anomaly); err != nil {
				fmt.Printf("Failed to trigger remediation for anomaly %s: %v\n", anomaly.ID, err)
				// Continue processing other anomalies
			}
		}
	}

	return nil
}
