#!/bin/bash
set -e

echo "================================================"
echo " MIORU Backend VPS Setup"
echo " Ubuntu 22.04 | Go 1.22 | PostgreSQL 16 | Nginx"
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

# ── Secrets (generated once; preserved across re-runs via /opt/mioru/.env) ──
mkdir -p /opt/mioru
if [ -f /opt/mioru/.env ]; then
    echo "Existing /opt/mioru/.env found — reusing its secrets"
    set -a; . /opt/mioru/.env; set +a
    DB_PASS=$(printf '%s' "$DATABASE_URL" | sed -n 's#.*//mioru:\([^@]*\)@.*#\1#p')
    NEW_SECRETS=0
else
    DB_PASS=$(openssl rand -hex 24)
    SECRET_KEY=$(openssl rand -base64 48 | tr -d '\n/+=')
    NEW_SECRETS=1
fi

# Create database and user
su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='mioru'\"" | grep -q 1 || \
    su - postgres -c "psql -c \"CREATE USER mioru WITH PASSWORD '${DB_PASS}';\""
su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname='mioru'\"" | grep -q 1 || \
    su - postgres -c "psql -c \"CREATE DATABASE mioru OWNER mioru;\""
su - postgres -c "psql -c \"GRANT ALL ON DATABASE mioru TO mioru;\""
su - postgres -c "psql -c \"ALTER USER mioru CREATEDB;\""

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

# ── App user (non-root service account) ──
if ! id mioru &>/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin mioru
    echo "Created system user 'mioru'"
fi

# ── App directory ──
mkdir -p /opt/mioru /opt/mioru/uploads /opt/mioru/logs
chown -R mioru:mioru /opt/mioru

# ── Environment (written once with random secrets; preserved on re-run) ──
if [ "$NEW_SECRETS" = "1" ]; then
cat > /opt/mioru/.env << ENVEOF
# Database (DB is on loopback; sslmode=disable is acceptable over localhost)
DATABASE_URL=postgres://mioru:${DB_PASS}@localhost:5432/mioru?sslmode=disable

# JWT
SECRET_KEY=${SECRET_KEY}

# Server
PORT=8000
UPLOAD_DIR=/opt/mioru/uploads

# CORS (comma-separated origins — must list every allowed front-end origin)
CORS_ORIGINS=https://mioru.store,https://www.mioru.store,https://admin.mioru.store,https://www.admin.mioru.store

# First admin bootstrap — set on first run, then remove (see README)
# BOOTSTRAP_ADMIN_USERNAME=admin
# BOOTSTRAP_ADMIN_EMAIL=admin@example.com
# BOOTSTRAP_ADMIN_PASSWORD=
ENVEOF
echo "Generated /opt/mioru/.env with random secrets"
fi
chmod 600 /opt/mioru/.env
chown mioru:mioru /opt/mioru/.env

# ── Systemd service ──
cat > /etc/systemd/system/mioru.service << 'UNITEOF'
[Unit]
Description=MIORU Backend API
After=network.target postgresql.service

[Service]
Type=simple
User=mioru
Group=mioru
WorkingDirectory=/opt/mioru
EnvironmentFile=/opt/mioru/.env
ExecStart=/opt/mioru/mioru
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=/opt/mioru

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
echo " 2. Fix ownership: chown mioru:mioru /opt/mioru/mioru"
echo "    (Secrets in /opt/mioru/.env are auto-generated — no manual passwords needed.)"
echo " 3. First admin: uncomment BOOTSTRAP_ADMIN_* in /opt/mioru/.env (see README), then start."
echo " 4. Start: systemctl start mioru"
echo " 5. DNS: api.mioru.store → 92.246.137.159"
echo " 6. SSL (TLS at the edge): apt install certbot python3-certbot-nginx && certbot --nginx"
echo ""
echo "Check status: systemctl status mioru"
echo "Check logs: journalctl -u mioru -f"
