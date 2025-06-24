#!/bin/bash

# Security Audit Script for AIOps Platform Monitoring Stack
# This script checks for common security issues in the monitoring stack

set -e

echo "=== AIOps Platform Monitoring Stack Security Audit ==="
echo "Date: $(date)"
echo

# Function to check if a command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Check required tools
for cmd in kubectl jq curl grep; do
  if ! command_exists $cmd; then
    echo "Error: Required command '$cmd' not found"
    exit 1
  fi
done

echo "=== 1. Authentication and Authorization ==="

# Check Grafana authentication
echo "Checking Grafana authentication settings..."
GRAFANA_POD=$(kubectl get pods -n monitoring -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.name}')
GRAFANA_AUTH_CONFIG=$(kubectl exec -n monitoring ${GRAFANA_POD} -- cat /etc/grafana/grafana.ini | grep -A 10 "\[auth\]")
echo "$GRAFANA_AUTH_CONFIG"
echo

# Check Prometheus RBAC
echo "Checking Prometheus RBAC settings..."
kubectl get clusterrole,clusterrolebinding,role,rolebinding -n monitoring | grep prometheus
echo

echo "=== 2. Network Security ==="

# Check service exposure
echo "Checking service exposure..."
kubectl get svc -n monitoring -o custom-columns=NAME:.metadata.name,TYPE:.spec.type,EXTERNAL-IP:.spec.externalIP
echo

# Check network policies
echo "Checking network policies..."
kubectl get networkpolicy -n monitoring
if [ $? -ne 0 ]; then
  echo "Warning: No network policies found in monitoring namespace"
fi
echo

echo "=== 3. Storage Security ==="

# Check volume permissions
echo "Checking volume permissions..."
for pod in $(kubectl get pods -n monitoring -o jsonpath='{.items[*].metadata.name}'); do
  echo "Pod: $pod"
  kubectl get pod $pod -n monitoring -o jsonpath='{.spec.securityContext}' | jq '.'
  echo
done

# Check container security contexts
echo "Checking container security contexts..."
for pod in $(kubectl get pods -n monitoring -o jsonpath='{.items[*].metadata.name}'); do
  echo "Pod: $pod"
  kubectl get pod $pod -n monitoring -o jsonpath='{.spec.containers[*].securityContext}' | jq '.'
  echo
done

echo "=== 4. Secret Management ==="

# Check for secrets
echo "Checking secrets..."
kubectl get secrets -n monitoring
echo

# Check for hardcoded credentials
echo "Checking for hardcoded credentials in ConfigMaps..."
kubectl get configmap -n monitoring -o json | jq -r '.items[] | select(.data != null) | .metadata.name' | while read cm; do
  echo "ConfigMap: $cm"
  HAS_CREDS=$(kubectl get configmap $cm -n monitoring -o json | jq -r '.data | to_entries[] | .value' | grep -i -E 'password|secret|key|token|credential' || echo "")
  if [ ! -z "$HAS_CREDS" ]; then
    echo "Warning: Potential credentials found in ConfigMap $cm"
  else
    echo "OK: No obvious credentials found"
  fi
done
echo

echo "=== 5. API Security ==="

# Check Prometheus API access
echo "Checking Prometheus API access..."
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090 &
PF_PID=$!
sleep 5
curl -s http://localhost:9090/api/v1/status/config | jq '.status'
kill $PF_PID || true
echo

# Check Loki API access
echo "Checking Loki API access..."
kubectl port-forward -n monitoring svc/loki 3100:3100 &
PF_PID=$!
sleep 5
curl -s http://localhost:3100/ready
kill $PF_PID || true
echo

echo "=== 6. Resource Limits ==="

# Check resource limits
echo "Checking resource limits..."
for pod in $(kubectl get pods -n monitoring -o jsonpath='{.items[*].metadata.name}'); do
  echo "Pod: $pod"
  kubectl get pod $pod -n monitoring -o jsonpath='{.spec.containers[*].resources}' | jq '.'
  echo
done

echo "=== Security Audit Summary ==="
echo "Completed security audit of the monitoring stack."
echo "Review the findings above and address any warnings or issues."
echo "For a complete security assessment, consider using specialized security scanning tools."
echo
echo "Audit completed at $(date)" 