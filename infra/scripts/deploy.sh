#!/usr/bin/env bash
# =============================================================================
# Authvora — Deploy / Redeploy
# Run from anywhere on the Lightsail instance to pull latest and restart
#
# Usage: bash /opt/authvora/infra/scripts/deploy.sh
# =============================================================================
set -euo pipefail

APP_DIR="/opt/authvora"
COMPOSE_FILE="${APP_DIR}/infra/docker/docker-compose.prod.yml"

cd "${APP_DIR}"

echo "[$(date)] Pulling latest code..."
git pull origin main

echo "[$(date)] Building and restarting services..."
cd infra/docker
docker compose -f docker-compose.prod.yml up -d --build --remove-orphans

echo "[$(date)] Cleaning up old images..."
docker image prune -f

echo ""
echo "[$(date)] Deploy complete. Service status:"
docker compose -f docker-compose.prod.yml ps
