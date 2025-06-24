#!/bin/bash

# Loki Backup Script for AIOps Platform
# This script creates a backup of Loki data

set -e

# Configuration
BACKUP_DIR="/backup/loki"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/loki-backup-${TIMESTAMP}.tar.gz"
LOKI_POD=$(kubectl get pods -n monitoring -l app=loki -o jsonpath='{.items[0].metadata.name}')
LOKI_DATA_DIR="/data/loki"

# Create backup directory if it doesn't exist
mkdir -p ${BACKUP_DIR}

echo "Starting Loki backup at $(date)"
echo "Backing up data from pod ${LOKI_POD}"

# Flush any pending writes to disk
echo "Flushing pending writes..."
kubectl exec -n monitoring ${LOKI_POD} -- curl -s -XPOST http://localhost:3100/flush

# Wait for flush to complete
sleep 10

# Create backup
echo "Creating backup archive..."
kubectl exec -n monitoring ${LOKI_POD} -- tar -czf - -C ${LOKI_DATA_DIR} . > ${BACKUP_FILE}

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
find ${BACKUP_DIR} -name "loki-backup-*.tar.gz" -type f -mtime +7 -delete

echo "Backup process completed" 