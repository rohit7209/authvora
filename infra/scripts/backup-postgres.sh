#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="/opt/authvora/backups"
RETENTION_DAYS=7
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/authvora_${TIMESTAMP}.sql.gz"

mkdir -p "${BACKUP_DIR}"

echo "[$(date)] Starting backup..."

docker exec authvora-postgres pg_dump \
  -U "${POSTGRES_USER:-authvora}" \
  -d "${POSTGRES_DB:-authvora}" \
  --no-owner \
  --clean \
  --if-exists \
  | gzip > "${BACKUP_FILE}"

FILESIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "[$(date)] Backup complete: ${BACKUP_FILE} (${FILESIZE})"

echo "[$(date)] Removing backups older than ${RETENTION_DAYS} days..."
find "${BACKUP_DIR}" -name "authvora_*.sql.gz" -mtime +${RETENTION_DAYS} -delete

REMAINING=$(ls -1 "${BACKUP_DIR}"/authvora_*.sql.gz 2>/dev/null | wc -l)
echo "[$(date)] Done. ${REMAINING} backups retained."
