# Grafana Dashboards Documentation

## Overview

This document provides information about the Grafana dashboards deployed as part of the AIOps platform monitoring stack. The dashboards are designed to provide comprehensive visibility into both the Kubernetes infrastructure and the AIOps anomaly detection system.

## Dashboard Categories

### 1. Kubernetes System Dashboards

These dashboards provide visibility into the underlying Kubernetes infrastructure:

| Dashboard | Description | Source |
|-----------|-------------|--------|
| **Kubernetes / Views / Global** | Overview of cluster-wide metrics including CPU, memory, and network usage | Community (dotdc) |
| **Kubernetes / Views / Nodes** | Detailed metrics for individual nodes in the cluster | Community (dotdc) |
| **Kubernetes / Views / Namespaces** | Resource usage and metrics broken down by namespace | Community (dotdc) |
| **Kubernetes / Views / Pods** | Detailed pod-level metrics for application monitoring | Community (dotdc) |

### 2. AIOps Specific Dashboards

These dashboards are custom-built for the AIOps anomaly detection system:

| Dashboard | Description |
|-----------|-------------|
| **AIOps Anomaly Detector** | Displays anomaly scores and detection latency metrics from the anomaly detector service |

## Accessing Dashboards

Grafana dashboards can be accessed through:

```bash
# Port-forward the Grafana service
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

# Access in browser
http://localhost:3000
```

Default login credentials:
- Username: `admin`
- Password: `aiops-admin-2024`

## Dashboard Auto-Discovery

The Grafana deployment is configured with dashboard auto-discovery, which automatically loads dashboards from ConfigMaps with the label `grafana_dashboard: "1"`. This allows for dashboard-as-code management where dashboards are stored in version control and deployed alongside the application.

### ConfigMap Structure

Dashboards are stored in ConfigMaps with the following structure:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: dashboard-name
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  dashboard-file.json: |-
    {
      # Dashboard JSON content
    }
```

## Alerting Integration

The dashboards are integrated with Prometheus alerting rules, which are configured to trigger alerts based on metrics visualized in the dashboards. Alert thresholds include:

- **High Anomaly Score**: Triggers when the anomaly score exceeds 0.7 for 5 minutes
- **Critical Anomaly Score**: Triggers when the anomaly score exceeds 0.9 for 2 minutes
- **High Detection Latency**: Triggers when detection latency exceeds 1 second for 5 minutes

## Customizing Dashboards

To customize dashboards:

1. Edit the dashboard through the Grafana UI
2. Export the dashboard JSON
3. Update the corresponding ConfigMap in the `kubernetes/monitoring/` directory
4. Apply the changes with `kubectl apply -f kubernetes/monitoring/dashboard-configmap.yaml`

## Troubleshooting

If dashboards are not appearing in Grafana:

1. Check if the ConfigMap has the correct label: `grafana_dashboard: "1"`
2. Verify the Grafana sidecar logs for any errors:
   ```bash
   kubectl logs -n monitoring deployment/prometheus-grafana -c grafana-sc-dashboard
   ```
3. Check if the dashboard JSON is valid by attempting to import it manually through the Grafana UI

## Adding New Dashboards

To add a new dashboard:

1. Create a new ConfigMap with the dashboard JSON
2. Apply the ConfigMap to the cluster
3. The dashboard will be automatically discovered and added to Grafana

Example:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-new-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  my-dashboard.json: |-
    {
      # Dashboard JSON content
    }
``` 