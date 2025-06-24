#!/bin/bash

# Grafana Backup Script for AIOps Platform
# This script creates a backup of Grafana dashboards and configuration

set -e

# Configuration
BACKUP_DIR="/backup/grafana"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/grafana-backup-${TIMESTAMP}.tar.gz"
TEMP_DIR="/tmp/grafana-backup-${TIMESTAMP}"
GRAFANA_POD=$(kubectl get pods -n monitoring -l app.kubernetes.io/name=grafana -o jsonpath='{.items[0].metadata.name}')
GRAFANA_API="http://localhost:3000/api"
GRAFANA_ADMIN_USER="admin"
GRAFANA_ADMIN_PASSWORD="aiops-admin-2024"

# Create backup directory if it doesn't exist
mkdir -p ${BACKUP_DIR}
mkdir -p ${TEMP_DIR}/{dashboards,datasources,folders,users,orgs,alerts}

echo "Starting Grafana backup at $(date)"
echo "Backing up data from pod ${GRAFANA_POD}"

# Port-forward to Grafana
kubectl port-forward -n monitoring ${GRAFANA_POD} 3000:3000 &
PF_PID=$!

# Wait for port-forward to establish
sleep 5

# Function to get API data
get_grafana_data() {
  local endpoint=$1
  local output_file=$2
  
  curl -s -k -u "${GRAFANA_ADMIN_USER}:${GRAFANA_ADMIN_PASSWORD}" \
    "${GRAFANA_API}/${endpoint}" > "${output_file}"
}

# Backup dashboards
echo "Backing up dashboards..."
get_grafana_data "search?type=dash-db" "${TEMP_DIR}/dashboards.json"
for dashboard_uid in $(jq -r '.[] | select(.type == "dash-db") | .uid' "${TEMP_DIR}/dashboards.json"); do
  echo "Backing up dashboard: ${dashboard_uid}"
  get_grafana_data "dashboards/uid/${dashboard_uid}" "${TEMP_DIR}/dashboards/${dashboard_uid}.json"
done

# Backup datasources
echo "Backing up datasources..."
get_grafana_data "datasources" "${TEMP_DIR}/datasources.json"
for ds_id in $(jq -r '.[] | .id' "${TEMP_DIR}/datasources.json"); do
  echo "Backing up datasource: ${ds_id}"
  get_grafana_data "datasources/${ds_id}" "${TEMP_DIR}/datasources/${ds_id}.json"
done

# Backup folders
echo "Backing up folders..."
get_grafana_data "folders" "${TEMP_DIR}/folders.json"
for folder_uid in $(jq -r '.[] | .uid' "${TEMP_DIR}/folders.json"); do
  echo "Backing up folder: ${folder_uid}"
  get_grafana_data "folders/${folder_uid}" "${TEMP_DIR}/folders/${folder_uid}.json"
done

# Backup users
echo "Backing up users..."
get_grafana_data "users" "${TEMP_DIR}/users.json"

# Backup organizations
echo "Backing up organizations..."
get_grafana_data "orgs" "${TEMP_DIR}/orgs.json"

# Backup alert rules
echo "Backing up alert rules..."
get_grafana_data "alert-rules" "${TEMP_DIR}/alerts/rules.json"

# Create archive
echo "Creating backup archive..."
tar -czf ${BACKUP_FILE} -C ${TEMP_DIR} .

# Clean up
kill ${PF_PID} || true
rm -rf ${TEMP_DIR}

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
find ${BACKUP_DIR} -name "grafana-backup-*.tar.gz" -type f -mtime +7 -delete

echo "Backup process completed" 