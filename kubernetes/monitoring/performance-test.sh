#!/bin/bash

# Performance Testing Script for AIOps Platform Monitoring Stack
# This script tests the performance of the monitoring stack components

set -e

echo "=== AIOps Platform Monitoring Stack Performance Test ==="
echo "Date: $(date)"
echo

# Function to check if a command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Check required tools
for cmd in kubectl jq curl grep awk; do
  if ! command_exists $cmd; then
    echo "Error: Required command '$cmd' not found"
    exit 1
  fi
done

# Create results directory
RESULTS_DIR="performance-results-$(date +%Y%m%d-%H%M%S)"
mkdir -p ${RESULTS_DIR}

echo "=== 1. Resource Usage Assessment ==="

# Function to get pod resource usage
get_pod_resources() {
  local namespace=$1
  local selector=$2
  local output_file=$3
  
  echo "Collecting resource usage for $selector in $namespace..."
  kubectl top pods -n $namespace -l $selector > $output_file
  cat $output_file
  echo
}

# Get resource usage for monitoring components
get_pod_resources "monitoring" "app.kubernetes.io/name=prometheus" "${RESULTS_DIR}/prometheus-resources.txt"
get_pod_resources "monitoring" "app.kubernetes.io/name=grafana" "${RESULTS_DIR}/grafana-resources.txt"
get_pod_resources "monitoring" "app=loki" "${RESULTS_DIR}/loki-resources.txt"
get_pod_resources "monitoring" "app=promtail" "${RESULTS_DIR}/promtail-resources.txt"

# Get node resource usage
echo "Collecting node resource usage..."
kubectl top nodes > "${RESULTS_DIR}/node-resources.txt"
cat "${RESULTS_DIR}/node-resources.txt"
echo

echo "=== 2. Storage Usage Assessment ==="

# Function to get PVC usage
get_pvc_usage() {
  local namespace=$1
  local pvc_name=$2
  local output_file=$3
  
  echo "Checking storage usage for PVC $pvc_name in $namespace..."
  kubectl get pvc $pvc_name -n $namespace -o json > $output_file
  cat $output_file | jq '.status.capacity, .spec.resources.requests'
  echo
}

# Get storage usage for monitoring components
get_pvc_usage "monitoring" "prometheus-prometheus-kube-prometheus-prometheus-db-prometheus-prometheus-kube-prometheus-prometheus-0" "${RESULTS_DIR}/prometheus-storage.json"
get_pvc_usage "monitoring" "prometheus-grafana" "${RESULTS_DIR}/grafana-storage.json"
get_pvc_usage "monitoring" "storage-loki-0" "${RESULTS_DIR}/loki-storage.json"

echo "=== 3. Query Performance Testing ==="

# Function to test Prometheus query performance
test_prometheus_query() {
  local query=$1
  local description=$2
  local output_file=$3
  local iterations=5
  
  echo "Testing Prometheus query: $description"
  kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090 &
  PF_PID=$!
  sleep 5
  
  echo "Running query $iterations times..."
  total_time=0
  for i in $(seq 1 $iterations); do
    start_time=$(date +%s.%N)
    curl -s -o /dev/null "http://localhost:9090/api/v1/query?query=${query}"
    end_time=$(date +%s.%N)
    query_time=$(echo "$end_time - $start_time" | bc)
    total_time=$(echo "$total_time + $query_time" | bc)
    echo "Iteration $i: $query_time seconds"
  done
  
  avg_time=$(echo "$total_time / $iterations" | bc -l)
  echo "Average query time: $avg_time seconds"
  echo "$description,$avg_time" >> $output_file
  
  kill $PF_PID || true
  echo
}

# Function to test Loki query performance
test_loki_query() {
  local query=$1
  local description=$2
  local output_file=$3
  local iterations=5
  
  echo "Testing Loki query: $description"
  kubectl port-forward -n monitoring svc/loki 3100:3100 &
  PF_PID=$!
  sleep 5
  
  echo "Running query $iterations times..."
  total_time=0
  for i in $(seq 1 $iterations); do
    start_time=$(date +%s.%N)
    curl -s -o /dev/null -G --data-urlencode "query=${query}" "http://localhost:3100/loki/api/v1/query_range" --data-urlencode "start=$(date -d '1 hour ago' +%s000000000)" --data-urlencode "end=$(date +%s000000000)" --data-urlencode "step=60s"
    end_time=$(date +%s.%N)
    query_time=$(echo "$end_time - $start_time" | bc)
    total_time=$(echo "$total_time + $query_time" | bc)
    echo "Iteration $i: $query_time seconds"
  done
  
  avg_time=$(echo "$total_time / $iterations" | bc -l)
  echo "Average query time: $avg_time seconds"
  echo "$description,$avg_time" >> $output_file
  
  kill $PF_PID || true
  echo
}

# Initialize query results files
echo "Query,Average Time (seconds)" > "${RESULTS_DIR}/prometheus-query-performance.csv"
echo "Query,Average Time (seconds)" > "${RESULTS_DIR}/loki-query-performance.csv"

# Test Prometheus queries
test_prometheus_query "up" "Simple up query" "${RESULTS_DIR}/prometheus-query-performance.csv"
test_prometheus_query "sum(rate(node_cpu_seconds_total{mode!='idle'}[5m])) by (instance)" "CPU usage by instance" "${RESULTS_DIR}/prometheus-query-performance.csv"
test_prometheus_query "sum by(container) (rate(container_cpu_usage_seconds_total{namespace='monitoring'}[5m]))" "Container CPU by namespace" "${RESULTS_DIR}/prometheus-query-performance.csv"
test_prometheus_query "histogram_quantile(0.99, sum(rate(prometheus_http_request_duration_seconds_bucket[5m])) by (le))" "Complex histogram query" "${RESULTS_DIR}/prometheus-query-performance.csv"

# Test Loki queries
test_loki_query "{namespace=\"monitoring\"}" "Simple namespace query" "${RESULTS_DIR}/loki-query-performance.csv"
test_loki_query "{namespace=\"monitoring\"} |= \"error\"" "Filter by content" "${RESULTS_DIR}/loki-query-performance.csv"
test_loki_query "{namespace=\"monitoring\"} | json" "JSON parsing" "${RESULTS_DIR}/loki-query-performance.csv"
test_loki_query "sum(count_over_time({namespace=\"monitoring\"}[5m])) by (pod)" "Aggregation query" "${RESULTS_DIR}/loki-query-performance.csv"

echo "=== 4. Dashboard Loading Performance ==="

# Function to test Grafana dashboard loading
test_dashboard_loading() {
  local dashboard_uid=$1
  local dashboard_name=$2
  local output_file=$3
  local iterations=3
  
  echo "Testing dashboard loading: $dashboard_name ($dashboard_uid)"
  kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80 &
  PF_PID=$!
  sleep 5
  
  # Get login session
  echo "Logging in to Grafana..."
  CSRF_TOKEN=$(curl -s -c cookies.txt http://localhost:3000/login | grep -oP 'csrf_token = "\K[^"]+')
  curl -s -b cookies.txt -c cookies.txt -X POST -H "X-Grafana-Org-Id: 1" -H "Content-Type: application/x-www-form-urlencoded" -d "user=admin&password=aiops-admin-2024&csrf_token=${CSRF_TOKEN}" http://localhost:3000/login > /dev/null
  
  echo "Running dashboard load test $iterations times..."
  total_time=0
  for i in $(seq 1 $iterations); do
    start_time=$(date +%s.%N)
    curl -s -b cookies.txt -o /dev/null "http://localhost:3000/d/${dashboard_uid}"
    end_time=$(date +%s.%N)
    load_time=$(echo "$end_time - $start_time" | bc)
    total_time=$(echo "$total_time + $load_time" | bc)
    echo "Iteration $i: $load_time seconds"
  done
  
  avg_time=$(echo "$total_time / $iterations" | bc -l)
  echo "Average dashboard load time: $avg_time seconds"
  echo "$dashboard_name,$avg_time" >> $output_file
  
  rm -f cookies.txt
  kill $PF_PID || true
  echo
}

# Initialize dashboard results file
echo "Dashboard,Average Load Time (seconds)" > "${RESULTS_DIR}/dashboard-load-performance.csv"

# Test dashboard loading
test_dashboard_loading "k8s_views_global" "Kubernetes Global View" "${RESULTS_DIR}/dashboard-load-performance.csv"
test_dashboard_loading "k8s_views_nodes" "Kubernetes Nodes View" "${RESULTS_DIR}/dashboard-load-performance.csv"
test_dashboard_loading "aiops-anomaly-detector" "AIOps Anomaly Detector" "${RESULTS_DIR}/dashboard-load-performance.csv"
test_dashboard_loading "aiops-logs-overview" "AIOps Logs Overview" "${RESULTS_DIR}/dashboard-load-performance.csv"

echo "=== Performance Test Summary ==="
echo "Performance test completed successfully."
echo "Results saved to ${RESULTS_DIR}/"
echo
echo "Test completed at $(date)" 