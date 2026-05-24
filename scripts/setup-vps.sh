#!/bin/bash
set -e

echo "================================================"
echo " MIORU Backend VPS Setup"
echo " Ubuntu 22.04 | Go 1.22 | PostgreSQL 16 | Redis | Nginx"
echo "================================================"

# ── System ──
apt update && apt upgrade -y

# Swap 2GB
if [ ! -f /swapfile ]; then
    fallocate -l 2G /swapfile
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo '/swapfile none swap sw 0 0' >> /etc/fstab
    echo "Swap 2GB created"
fi

# ── Go 1.22 ──
if ! command -v go &>/dev/null; then
    wget -q https://go.dev/dl/go1.22.10.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.22.10.linux-amd64.tar.gz
    rm go1.22.10.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    echo "Go $(go version) installed"
fi

# ── PostgreSQL 16 ──
if ! command -v psql &>/dev/null; then
    sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg
    apt update
    apt install -y postgresql-16
    echo "PostgreSQL 16 installed"
fi

# Create database and user
su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='mioru'\"" | grep -q 1 || \
    su - postgres -c "psql -c \"CREATE USER mioru WITH PASSWORD 'change-me-in-production';\""
su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname='mioru'\"" | grep -q 1 || \
    su - postgres -c "psql -c \"CREATE DATABASE mioru OWNER mioru;\""
su - postgres -c "psql -c \"GRANT ALL ON DATABASE mioru TO mioru;\""
su - postgres -c "psql -c \"ALTER USER mioru CREATEDB;\""

# ── Redis ──
if ! command -v redis-server &>/dev/null; then
    apt install -y redis-server
    systemctl enable redis-server
    systemctl start redis-server
    echo "Redis installed"
fi

# ── Nginx ──
if ! command -v nginx &>/dev/null; then
    apt install -y nginx
    systemctl enable nginx
    echo "Nginx installed"
fi

# ── Firewall ──
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# ── App directory ──
mkdir -p /opt/mioru /opt/mioru/uploads /opt/mioru/logs
chown -R root:root /opt/mioru

# ── Environment ──
cat > /opt/mioru/.env << 'ENVEOF'
# Database
DATABASE_URL=postgres://mioru:change-me-in-production@localhost:5432/mioru?sslmode=disable

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# JWT
SECRET_KEY=change-me-to-random-string-64-chars-min-like-this-one-here-ok

# Server
PORT=8000
UPLOAD_DIR=/opt/mioru/uploads

# CORS (comma-separated origins)
CORS_ORIGINS=https://mioru.store,https://www.mioru.store
ENVEOF

# ── Systemd service ──
cat > /etc/systemd/system/mioru.service << 'UNITEOF'
[Unit]
Description=MIORU Backend API
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/mioru
EnvironmentFile=/opt/mioru/.env
ExecStart=/opt/mioru/mioru
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNITEOF

systemctl daemon-reload

# ── Nginx config ──
cat > /etc/nginx/sites-available/mioru << 'NGXEOF'
server {
    listen 80;
    server_name api.mioru.store;

    client_max_body_size 32M;

    location /uploads/ {
        alias /opt/mioru/uploads/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
NGXEOF

ln -sf /etc/nginx/sites-available/mioru /etc/nginx/sites-enabled/mioru
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

echo ""
echo "================================================"
echo " DONE"
echo "================================================"
echo ""
echo "Next steps:"
echo " 1. Copy the Go binary: scp backend/api/mioru root@92.246.137.159:/opt/mioru/"
echo " 2. Set STRONG passwords in /opt/mioru/.env"
echo " 3. Start: systemctl start mioru"
echo " 4. DNS: api.mioru.store → 92.246.137.159"
echo " 5. SSL: apt install certbot python3-certbot-nginx && certbot --nginx"
echo ""
echo "Check status: systemctl status mioru"
echo "Check logs: journalctl -u mioru -f"
