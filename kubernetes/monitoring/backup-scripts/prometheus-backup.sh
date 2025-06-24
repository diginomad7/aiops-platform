#!/bin/bash

# Prometheus Backup Script for AIOps Platform
# This script creates a backup of Prometheus data

set -e

# Configuration
BACKUP_DIR="/backup/prometheus"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/prometheus-backup-${TIMESTAMP}.tar.gz"
PROMETHEUS_POD=$(kubectl get pods -n monitoring -l app.kubernetes.io/name=prometheus -o jsonpath='{.items[0].metadata.name}')
PROMETHEUS_DATA_DIR="/prometheus"

# Create backup directory if it doesn't exist
mkdir -p ${BACKUP_DIR}

echo "Starting Prometheus backup at $(date)"
echo "Backing up data from pod ${PROMETHEUS_POD}"

# Create snapshot
echo "Creating Prometheus snapshot..."
SNAPSHOT_NAME=$(kubectl exec -n monitoring ${PROMETHEUS_POD} -c prometheus -- curl -s -XPOST http://localhost:9090/api/v1/admin/tsdb/snapshot | jq -r '.data.name')

if [ -z "${SNAPSHOT_NAME}" ]; then
  echo "Failed to create snapshot"
  exit 1
fi

echo "Snapshot created: ${SNAPSHOT_NAME}"

# Wait for snapshot to complete
sleep 5

# Copy snapshot data
echo "Copying snapshot data..."
kubectl exec -n monitoring ${PROMETHEUS_POD} -c prometheus -- tar -czf - -C ${PROMETHEUS_DATA_DIR}/snapshots ${SNAPSHOT_NAME} > ${BACKUP_FILE}

# Verify backup file
if [ -f "${BACKUP_FILE}" ]; then
  BACKUP_SIZE=$(du -h ${BACKUP_FILE} | cut -f1)
  echo "Backup completed successfully at $(date)"
  echo "Backup file: ${BACKUP_FILE}"
  echo "Backup size: ${BACKUP_SIZE}"
else
  echo "Backup failed: Backup file not created"
  exit 1
fi

# Clean up old backups (keep last 7 days)
find ${BACKUP_DIR} -name "prometheus-backup-*.tar.gz" -type f -mtime +7 -delete

echo "Backup process completed" 