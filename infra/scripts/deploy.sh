#!/usr/bin/env bash
# =============================================================================
# Authvora — Smart Deploy
# Detects which services changed and only rebuilds those.
#
# Usage:
#   bash /opt/authvora/infra/scripts/deploy.sh          # smart (selective)
#   bash /opt/authvora/infra/scripts/deploy.sh --all     # force rebuild all
# =============================================================================
set -euo pipefail

APP_DIR="/opt/authvora"
COMPOSE_DIR="${APP_DIR}/infra/docker"
COMPOSE_FILE="docker-compose.prod.yml"
FORCE_ALL="${1:-}"

cd "${APP_DIR}"

BEFORE_SHA=$(git rev-parse HEAD)

echo "[$(date)] Pulling latest code..."
git pull origin main

AFTER_SHA=$(git rev-parse HEAD)

if [ "$BEFORE_SHA" = "$AFTER_SHA" ] && [ "$FORCE_ALL" != "--all" ]; then
    echo "[$(date)] No changes detected. Nothing to deploy."
    exit 0
fi

CHANGED_FILES=$(git diff --name-only "$BEFORE_SHA" "$AFTER_SHA" 2>/dev/null || echo "--all")

SERVICES_TO_BUILD=()
NGINX_CHANGED=false
FRONTEND_CHANGED=false
COMPOSE_CHANGED=false
MIGRATION_CHANGED=false

if [ "$FORCE_ALL" = "--all" ] || [ "$CHANGED_FILES" = "--all" ]; then
    SERVICES_TO_BUILD=("auth-service" "api-gateway" "risk-engine" "policy-engine" "mcp-server")
    NGINX_CHANGED=true
    FRONTEND_CHANGED=true
    COMPOSE_CHANGED=true
else
    while IFS= read -r file; do
        case "$file" in
            services/auth-service/*)    SERVICES_TO_BUILD+=("auth-service") ;;
            services/api-gateway/*)     SERVICES_TO_BUILD+=("api-gateway") ;;
            services/risk-engine/*)     SERVICES_TO_BUILD+=("risk-engine") ;;
            services/policy-engine/*)   SERVICES_TO_BUILD+=("policy-engine") ;;
            services/mcp-server/*)      SERVICES_TO_BUILD+=("mcp-server") ;;
            infra/docker/*)             COMPOSE_CHANGED=true ;;
            infra/nginx/*)              NGINX_CHANGED=true ;;
            infra/migrations/*)         MIGRATION_CHANGED=true ;;
            infra/frontend-placeholder/*) FRONTEND_CHANGED=true ;;
        esac
    done <<< "$CHANGED_FILES"
fi

# Deduplicate services list
SERVICES_TO_BUILD=($(printf '%s\n' "${SERVICES_TO_BUILD[@]}" 2>/dev/null | sort -u || true))

cd "$COMPOSE_DIR"

if [ "$COMPOSE_CHANGED" = true ]; then
    echo "[$(date)] Docker Compose config changed — restarting all services..."
    docker compose -f "$COMPOSE_FILE" up -d --build --remove-orphans
elif [ ${#SERVICES_TO_BUILD[@]} -gt 0 ]; then
    echo "[$(date)] Rebuilding changed services: ${SERVICES_TO_BUILD[*]}"
    docker compose -f "$COMPOSE_FILE" up -d --build --no-deps "${SERVICES_TO_BUILD[@]}"
else
    echo "[$(date)] No service changes detected."
fi

if [ "$MIGRATION_CHANGED" = true ]; then
    echo "[$(date)] Migration files changed — restart postgres to apply..."
    docker compose -f "$COMPOSE_FILE" restart postgres
fi

if [ "$NGINX_CHANGED" = true ]; then
    echo "[$(date)] Updating Nginx config..."
    sudo cp "${APP_DIR}/infra/nginx/authvora.conf" /etc/nginx/conf.d/authvora.conf
    sudo nginx -t && sudo systemctl reload nginx
fi

if [ "$FRONTEND_CHANGED" = true ]; then
    echo "[$(date)] Updating frontend..."
    sudo mkdir -p /var/www/authvora
    sudo cp "${APP_DIR}/infra/frontend-placeholder/index.html" /var/www/authvora/index.html 2>/dev/null || true
fi

echo "[$(date)] Cleaning up old images..."
docker image prune -f

echo "[$(date)] Waiting for services..."
sleep 10

echo "[$(date)] Health check..."
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>/dev/null || echo "000")

if [ "$HEALTH_STATUS" = "200" ]; then
    echo "[$(date)] Health check PASSED (HTTP $HEALTH_STATUS)"
else
    echo "[$(date)] Health check FAILED (HTTP $HEALTH_STATUS)"
    echo "[$(date)] Service status:"
    docker compose -f "$COMPOSE_FILE" ps
    echo "[$(date)] Recent logs:"
    docker compose -f "$COMPOSE_FILE" logs --tail=20
    exit 1
fi

echo ""
echo "[$(date)] Deploy complete. Service status:"
docker compose -f "$COMPOSE_FILE" ps
