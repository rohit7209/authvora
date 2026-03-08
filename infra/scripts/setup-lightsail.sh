#!/usr/bin/env bash
# =============================================================================
# Authvora — Lightsail Instance Setup Script
# Run this once on a fresh Ubuntu 22.04/24.04 Lightsail instance
#
# Usage: ssh into your instance, then:
#   curl -sL https://raw.githubusercontent.com/authvora/authvora/main/infra/scripts/setup-lightsail.sh | bash
#   OR copy this script and run: bash setup-lightsail.sh
# =============================================================================
set -euo pipefail

DOMAIN="${DOMAIN:-api.authvora.com}"
APP_DIR="/opt/authvora"
REPO_URL="${REPO_URL:-https://github.com/authvora/authvora.git}"

echo "========================================="
echo " Authvora Lightsail Setup"
echo " Domain: ${DOMAIN}"
echo "========================================="

# --- System updates ---
echo "[1/7] Updating system packages..."
sudo apt-get update -qq
sudo apt-get upgrade -y -qq

# --- Docker ---
echo "[2/7] Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sudo sh
    sudo usermod -aG docker "$USER"
    echo "  Docker installed. You may need to log out and back in for group changes."
else
    echo "  Docker already installed."
fi

# --- Docker Compose ---
echo "[3/7] Verifying Docker Compose..."
if ! docker compose version &> /dev/null; then
    sudo apt-get install -y -qq docker-compose-plugin
fi
docker compose version

# --- Nginx ---
echo "[4/7] Installing Nginx..."
sudo apt-get install -y -qq nginx
sudo systemctl enable nginx

# --- Certbot ---
echo "[5/7] Installing Certbot..."
sudo apt-get install -y -qq certbot python3-certbot-nginx

# --- Clone repo ---
echo "[6/7] Setting up application..."
sudo mkdir -p "${APP_DIR}"
sudo chown "$USER:$USER" "${APP_DIR}"

if [ -d "${APP_DIR}/.git" ]; then
    echo "  Repo exists, pulling latest..."
    cd "${APP_DIR}" && git pull
else
    git clone "${REPO_URL}" "${APP_DIR}"
fi

# --- Create .env if missing ---
if [ ! -f "${APP_DIR}/infra/docker/.env" ]; then
    echo "[!] Creating .env from production template..."
    cp "${APP_DIR}/.env.production" "${APP_DIR}/infra/docker/.env"

    PG_PASS=$(openssl rand -hex 24)
    REDIS_PASS=$(openssl rand -hex 24)

    sed -i "s|CHANGE_ME_use_openssl_rand_hex_32|${PG_PASS}|" "${APP_DIR}/infra/docker/.env"
    sed -i "0,/CHANGE_ME_use_openssl_rand_hex_32/! s|CHANGE_ME_use_openssl_rand_hex_32|${REDIS_PASS}|" "${APP_DIR}/infra/docker/.env"

    echo "  Generated .env with random credentials."
    echo "  IMPORTANT: Review and update ${APP_DIR}/infra/docker/.env before starting!"
fi

# --- Nginx config ---
echo "[7/7] Configuring Nginx..."
sudo cp "${APP_DIR}/infra/nginx/authvora.conf" /etc/nginx/sites-available/authvora
sudo ln -sf /etc/nginx/sites-available/authvora /etc/nginx/sites-enabled/authvora
sudo rm -f /etc/nginx/sites-enabled/default

sudo sed -i "s|api.authvora.com|${DOMAIN}|g" /etc/nginx/sites-available/authvora

sudo nginx -t && sudo systemctl reload nginx

# --- Backup cron ---
echo "Setting up daily backup cron..."
CRON_CMD="0 3 * * * cd ${APP_DIR} && source infra/docker/.env && bash infra/scripts/backup-postgres.sh >> /var/log/authvora-backup.log 2>&1"
(crontab -l 2>/dev/null | grep -v "backup-postgres" ; echo "${CRON_CMD}") | crontab -

echo ""
echo "========================================="
echo " Setup complete!"
echo "========================================="
echo ""
echo " Next steps:"
echo ""
echo " 1. Point your DNS:"
echo "    ${DOMAIN} → $(curl -s ifconfig.me)"
echo ""
echo " 2. Edit your .env:"
echo "    nano ${APP_DIR}/infra/docker/.env"
echo ""
echo " 3. Get SSL certificate (after DNS propagates):"
echo "    sudo certbot --nginx -d ${DOMAIN}"
echo ""
echo " 4. Start Authvora:"
echo "    cd ${APP_DIR}/infra/docker"
echo "    docker compose -f docker-compose.prod.yml up -d --build"
echo ""
echo " 5. Verify:"
echo "    curl https://${DOMAIN}/health"
echo ""
echo " 6. View logs:"
echo "    docker compose -f docker-compose.prod.yml logs -f"
echo ""
