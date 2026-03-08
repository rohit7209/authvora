#!/usr/bin/env bash
# =============================================================================
# Authvora — Deploy / Redeploy
# Run from anywhere on the Lightsail instance to pull latest and restart
#
# Usage: bash /opt/authvora/infra/scripts/deploy.sh
# =============================================================================
set -euo pipefail

APP_DIR="/opt/authvora"

cd "${APP_DIR}"

echo "[$(date)] Pulling latest code..."
git pull origin main

echo "[$(date)] Building and restarting services..."
cd infra/docker
docker compose -f docker-compose.prod.yml up -d --build --remove-orphans

echo "[$(date)] Updating Nginx config..."
sudo cp "${APP_DIR}/infra/nginx/authvora.conf" /etc/nginx/conf.d/authvora.conf
sudo nginx -t && sudo systemctl reload nginx

echo "[$(date)] Updating frontend placeholder..."
sudo mkdir -p /var/www/authvora
sudo cp "${APP_DIR}/infra/frontend-placeholder/index.html" /var/www/authvora/index.html 2>/dev/null || true

echo "[$(date)] Cleaning up old images..."
docker image prune -f

echo "[$(date)] Waiting for services to be healthy..."
sleep 10

echo "[$(date)] Health check..."
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>/dev/null || echo "000")

if [ "$HEALTH_STATUS" = "200" ]; then
    echo "[$(date)] Health check PASSED (HTTP $HEALTH_STATUS)"
else
    echo "[$(date)] Health check FAILED (HTTP $HEALTH_STATUS)"
    echo "[$(date)] Service status:"
    docker compose -f docker-compose.prod.yml ps
    echo "[$(date)] Recent logs:"
    docker compose -f docker-compose.prod.yml logs --tail=20
    exit 1
fi

echo ""
echo "[$(date)] Deploy complete. Service status:"
docker compose -f docker-compose.prod.yml ps
