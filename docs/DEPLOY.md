# asyou Public Server Deployment Guide

Deploy the asyou management server + frps to a public server, making your tunnel management platform globally accessible.

---

## 1. Architecture Overview

```
                          ┌──────────────────────────────────────┐
                          │         asyou Management Server      │
                          │  :8080 (API + Web Dashboard)         │
                          │  ┌────────┐ ┌────────┐ ┌──────────┐ │
                          │  │ Auth   │ │ Proxy  │ │Scheduler │ │
                          │  │ JWT    │ │ CRUD   │ │Geo+Weight│ │
                          │  └────────┘ └────┬───┘ └────┬─────┘ │
                          └──────────────────┼───────────┼───────┘
                                             │           │
                    ┌────────────────────────┼───────────┼──────────────┐
                    │                        │           │              │
               ┌────▼────┐            ┌──────▼────┐ ┌───▼────┐  ┌─────▼─────┐
               │ VPS 1   │            │  VPS 2    │ │ VPS 3  │  │  ...      │
               │ frps    │            │  frps     │ │ frps   │  │           │
               │ :7001   │            │  :7002    │ │ :7003  │  │           │
               │ HK      │            │  Tokyo    │ │ US-West│  │           │
               └────┬────┘            └──────┬────┘ └───┬────┘  └─────┬─────┘
                    │                        │          │              │
                    └──────────┬─────────────┴──────────┴──────────────┘
                               │
                 ┌─────────────┼────────────────┐
                 │             │                │
            Local Machine  Local Machine    Browser
            (frpc)         (frpc)           (Dashboard)
```

- **asyou server**: Management API + Web Dashboard + Node Scheduler
- **frps nodes**: Multiple tunnel ingress servers (different VPS, regions, or ports)
- **Scheduler**: Auto-selects best node based on weight, geo-proximity, capacity, latency
- **frpc**: Runs on machines that need to expose services, connects to the assigned frps

---

## 2. Prerequisites

### 2.1 Server Requirements

| Item | Minimum | Recommended |
|------|---------|-------------|
| CPU | 1 core | 2 cores |
| RAM | 512 MB | 2 GB |
| Disk | 1 GB | 10 GB SSD |
| OS | Ubuntu 20.04+ / Debian 11+ | Ubuntu 24.04 LTS |
| Network | Public IP, open ports | 50 Mbps+ |

### 2.2 Ports to Open

| Port | Service | Description |
|------|---------|-------------|
| 22 | SSH | Remote administration |
| 80 | HTTP | ACME validation + redirect to HTTPS |
| 443 | HTTPS | Web Dashboard + API |
| 7000-7100 | frps | Tunnel ports for multiple nodes |
| 8080 | asyou API | Internal / reverse proxy use |

---

## 3. One-Click Deployment Script

Run the following commands on your server sequentially:

```bash
# === 1. Install dependencies ===
sudo apt update && sudo apt install -y git golang-go nginx certbot

# === 2. Clone asyou source ===
git clone https://github.com/your-org/asyou.git /opt/asyou
cd /opt/asyou

# === 3. Download frp binaries ===
VER="0.69.1"
cd /tmp
curl -sL "https://github.com/fatedier/frp/releases/download/v${VER}/frp_${VER}_linux_amd64.tar.gz" -o frp.tar.gz
tar xzf frp.tar.gz
sudo cp "frp_${VER}_linux_amd64/frps" /usr/local/bin/frps
sudo cp "frp_${VER}_linux_amd64/frpc" /usr/local/bin/frpc
sudo chmod +x /usr/local/bin/frps /usr/local/bin/frpc
rm -rf "frp_${VER}_linux_amd64" frp.tar.gz

# === 4. Build asyou server ===
cd /opt/asyou/server
go build -o /usr/local/bin/asyou-server ./cmd/server

# === 5. Create data directory ===
sudo mkdir -p /var/lib/asyou
```

---

## 4. frps Configuration

Create the frps configuration file:

```bash
sudo tee /etc/frps.toml << 'EOF'
[common]
bind_port = 7000
bind_addr = "0.0.0.0"
allow_ports = "31000-31499"

# Optional: HTTP/HTTPS virtual host ports (for http/https proxy types)
# vhost_http_port = 80
# vhost_https_port = 443

# Optional: subdomain host — enables http://<subdomain>.<host>/ access
# Requires DNS wildcard record *.tunnel.example.com → this server
# subdomain_host = tunnel.example.com

# Optional: KCP/QUIC transport
# kcp_bind_port = 7000
# bind_udp_port = 7001

# Optional Dashboard
dashboard_port = 7500
dashboard_user = "admin"
dashboard_pwd = "CHANGE_THIS_PASSWORD"

# Logging
log_file = "/var/log/frps.log"
log_level = "info"
log_max_days = 7
EOF
```

Start frps:

```bash
frps -c /etc/frps.toml &
```

---

## 5. asyou Server Configuration

### 5.1 Database Initialization

```bash
# asyou server uses SQLite; the database is created automatically on first start
# Database location: /var/lib/asyou/asyou.db
```

### 5.2 Start asyou Server

```bash
# Foreground mode (for testing)
asyou-server &

# Verify
curl http://localhost:8080/
# → "asyou server is running..."
```

---

## 5.3 Single-Server Cluster Simulation

Run multiple frps instances on a single machine (different ports) to simulate a multi-node cluster for testing the scheduler, failover, etc.

### 5.3.1 Create frps Configurations

```bash
# Node 1 — Primary
sudo tee /etc/frps-node1.toml << 'EOF'
[common]
bind_addr = "0.0.0.0"
bind_port = 7001
allow_ports = "31600-31699"
token = "node1-secret"

dashboard_port = 7501
dashboard_user = "admin"
dashboard_pwd = "CHANGE_THIS"

log_file = "/var/log/frps-node1.log"
log_level = "info"
log_max_days = 7
EOF

# Node 2 — Standby
sudo tee /etc/frps-node2.toml << 'EOF'
[common]
bind_addr = "0.0.0.0"
bind_port = 7002
allow_ports = "31700-31799"
token = "node2-secret"

dashboard_port = 7502
dashboard_user = "admin"
dashboard_pwd = "CHANGE_THIS"

log_file = "/var/log/frps-node2.log"
log_level = "info"
log_max_days = 7
EOF

# Node 3 — Low priority (low weight, simulates poor performance)
sudo tee /etc/frps-node3.toml << 'EOF'
[common]
bind_addr = "0.0.0.0"
bind_port = 7003
allow_ports = "31800-31899"
token = "node3-secret"

dashboard_port = 7503
dashboard_user = "admin"
dashboard_pwd = "CHANGE_THIS"

log_file = "/var/log/frps-node3.log"
log_level = "info"
log_max_days = 7
EOF
```

### 5.3.2 Start All frps Instances

```bash
sudo frps -c /etc/frps-node1.toml &
sudo frps -c /etc/frps-node2.toml &
sudo frps -c /etc/frps-node3.toml &

# Verify all started
sudo ss -tlnp | grep frps
# Should show 7001 7002 7003 7501 7502 7503
```

### 5.3.3 Register Nodes with asyou

```bash
# First login (get token)
LOGIN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"your-password"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
AUTH="Authorization: Bearer $TOKEN"

# Register node 1 (weight 1.0 = normal)
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"hk-main","host":"YOUR_SERVER_IP","api_port":7501,"bind_port":7001,"auth_token":"node1-secret","region":"ap-east","country":"HK","city":"Hong Kong","latitude":22.3193,"longitude":114.1694,"max_connections":100,"weight":1.0}'

# Register node 2 (weight 0.8 = fewer tunnels assigned)
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"hk-standby","host":"YOUR_SERVER_IP","api_port":7502,"bind_port":7002,"auth_token":"node2-secret","region":"ap-east","country":"HK","city":"Hong Kong","latitude":22.3193,"longitude":114.1694,"max_connections":80,"weight":0.8}'

# Register node 3 (weight 0.3 = low priority)
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"hk-low","host":"YOUR_SERVER_IP","api_port":7503,"bind_port":7003,"auth_token":"node3-secret","region":"ap-east","country":"HK","city":"Hong Kong","latitude":22.3193,"longitude":114.1694,"max_connections":50,"weight":0.3}'
```

### 5.3.4 Verification

```bash
# List nodes via API
curl -s http://localhost:8080/api/v1/nodes -H "$AUTH" | python3 -m json.tool

# List nodes via CLI
./asyou nodes

# Expected output:
# ID   Name          Host              Port
# 1    hk-main       YOUR_SERVER_IP    7001
# 2    hk-standby    YOUR_SERVER_IP    7002
# 3    hk-low        YOUR_SERVER_IP    7003

# Create a tunnel (no node specified — scheduler auto-selects best node)
./asyou expose 3000 -n my-app
# → auto-selected node #1 "hk-main" (highest weight)
```

The scheduler assigns tunnels to frps instances based on `weight`, `max_connections`, latency, etc. Adjust these parameters to simulate different scenarios.

---

## 6. systemd Services (Auto-Start on Boot)

### 6.1 frps Service

```bash
sudo tee /etc/systemd/system/frps.service << 'EOF'
[Unit]
Description=frps tunnel server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/frps -c /etc/frps.toml
Restart=always
RestartSec=5
User=nobody
Group=nogroup

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now frps
sudo systemctl status frps
```

### 6.2 asyou Service

```bash
sudo tee /etc/systemd/system/asyou-server.service << 'EOF'
[Unit]
Description=asyou tunnel management server
After=network.target frps.service
Wants=frps.service

[Service]
Type=simple
ExecStart=/usr/local/bin/asyou-server
WorkingDirectory=/var/lib/asyou
Restart=always
RestartSec=5
User=nobody
Group=nogroup
Environment=HOME=/var/lib/asyou

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now asyou-server
sudo systemctl status asyou-server
```

### 6.3 Viewing Logs

```bash
sudo journalctl -u asyou-server -f
sudo journalctl -u frps -f
```

---

## 7. Nginx Reverse Proxy (HTTPS Domain Access)

### 7.1 Nginx Configuration

Assuming your domain is `asyou.example.com`:

```bash
sudo tee /etc/nginx/sites-available/asyou << 'EOF'
server {
    listen 80;
    server_name asyou.example.com;

    # ACME challenge directory
    location ^~ /.well-known/acme-challenge/ {
        root /var/www/html;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name asyou.example.com;

    # SSL certificate (provisioned with certbot)
    ssl_certificate /etc/letsencrypt/live/asyou.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/asyou.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    # Proxy asyou API
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Proxy SSE (buffering must be disabled)
    location /api/v1/events {
        proxy_pass http://127.0.0.1:8080;
        proxy_buffering off;
        proxy_cache off;
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        chunked_transfer_encoding on;
    }

    # Serve Web Dashboard (static files from web/dist)
    location / {
        root /opt/asyou/web/dist;
        index index.html;
        try_files $uri $uri/ /index.html;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/asyou /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

### 7.2 Provision SSL Certificate

```bash
sudo apt install -y python3-certbot-nginx
sudo certbot --nginx -d asyou.example.com --non-interactive --agree-tos -m admin@example.com

# Auto-renewal (configured by default)
sudo certbot renew --dry-run
```

### 7.3 Build Web Dashboard (Optional)

Skip this step if you don't need the Web UI.

```bash
cd /opt/asyou/web
npm install
npm run build
# Static files output to /opt/asyou/web/dist
# Nginx is already configured to serve these files
```

---

## 8. Register & Initialize

After the services are running, execute the following via SSH or directly on the server:

```bash
# Register an admin account
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"your-strong-password","display_name":"Admin"}'

# Register the local frps node
LOGIN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"your-strong-password"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"public-frps","host":"YOUR_SERVER_PUBLIC_IP","bind_port":7000,"subdomain_host":"tunnel.example.com"}'

# Verify
curl http://localhost:8080/api/v1/nodes -H "Authorization: Bearer $TOKEN"
```

---

## 9. Client Installation (frpc + CLI)

Install the asyou client on machines that need to expose services.

### 9.1 One-Click Install Script

```bash
# Run on the local machine
# Automatically installs frpc + asyou CLI

bash <(curl -sL https://raw.githubusercontent.com/your-org/asyou/main/scripts/install-client.sh)
```

Or install manually:

### 9.2 Install frpc

```bash
# Get the recommended frpc version from the server
VER=$(curl -s https://asyou.example.com/api/v1/version | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['recommended_frpc_version'])")

# Download matching version
cd /tmp
curl -sL "https://github.com/fatedier/frp/releases/download/v${VER}/frp_${VER}_linux_amd64.tar.gz" -o frp.tar.gz
tar xzf frp.tar.gz
sudo cp "frp_${VER}_linux_amd64/frpc" /usr/local/bin/frpc
sudo chmod +x /usr/local/bin/frpc
rm -rf "frp_${VER}_linux_amd64" frp.tar.gz

# Verify
frpc --version
```

### 9.3 Install asyou CLI

**Option A: Build from source (recommended)**

```bash
# Requires Go 1.20+
sudo apt install -y golang-go git   # Ubuntu/Debian
# or brew install go                # macOS

git clone https://github.com/your-org/asyou.git /tmp/asyou-src
cd /tmp/asyou-src/cli
go build -o /usr/local/bin/asyou .
rm -rf /tmp/asyou-src

# Verify
asyou --help
```

**Option B: Download pre-built binary**

```bash
# (Coming soon — please use Option A for now)
```

### 9.4 Configure & Login

```bash
# Login to the public asyou server
asyou login --s https://asyou.example.com admin@example.com your-password

# Verify login
asyou list
# → Shows the tunnel list on the server (empty initially)
```

### 9.5 One-Click Expose a Local Service

```bash
# Expose a web service running on local port 3000
asyou expose 3000 --n my-app

# Check status
asyou list
# → ID  Name    Type  Port   Status
# → 1   my-app  tcp   3000   running

# View the local frpc process
ps aux | grep frpc
# → /usr/local/bin/frpc -c /tmp/asyou-proxy-1-xxx.ini
```

### 9.6 Check Version Consistency

```bash
asyou version
# → asyou CLI version:     0.1.0
# → Server version:         0.1.0
# → Recommended frpc:       0.69.1

asyou check
# → Expected frpc: 0.69.1
# → Actual frpc:   0.69.1
# → ✓ Version OK
```

### 9.7 Using Docker (Optional)

```dockerfile
# Dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN cd cli && go build -o /asyou .

FROM ubuntu:22.04
RUN apt update && apt install -y curl && \
    VER=$(curl -s https://asyou.example.com/api/v1/version | python3 -c \
      "import sys,json; print(json.load(sys.stdin)['recommended_frpc_version'])") && \
    curl -sL "https://github.com/fatedier/frp/releases/download/v${VER}/frp_${VER}_linux_amd64.tar.gz" \
      -o /tmp/frp.tar.gz && tar xzf /tmp/frp.tar.gz -C /tmp && \
    cp "/tmp/frp_${VER}_linux_amd64/frpc" /usr/local/bin/frpc && \
    rm -rf /tmp/frp*
COPY --from=builder /app/asyou /usr/local/bin/asyou
ENTRYPOINT ["asyou"]
```

### 9.8 Windows Client

```powershell
# Run PowerShell as Administrator

# 1. Download frpc
$VER = "0.69.1"  # Get from https://asyou.example.com/api/v1/version
$url = "https://github.com/fatedier/frp/releases/download/v$VER/frp_$VER_windows_amd64.zip"
Invoke-WebRequest $url -OutFile "$env:TEMP\frp.zip"
Expand-Archive "$env:TEMP\frp.zip" -DestinationPath "$env:TEMP\frp"
Copy-Item "$env:TEMP\frp\frp_$VER_windows_amd64\frpc.exe" "C:\Windows\System32\frpc.exe"

# 2. Build CLI (requires Go)
cd cli
go build -o "$env:USERPROFILE\go\bin\asyou.exe"

# 3. Login
asyou login --s https://asyou.example.com admin@example.com your-password

# 4. Expose local service
asyou expose 3000 --n my-app
```

---

## 10. Security Hardening

### 10.1 Change the Default JWT Secret

Edit `server/internal/handlers/auth.go` and update the JWT key:

```go
var jwtKey = []byte("replace-this-with-a-random-string")
```

Then rebuild:

```bash
cd /opt/asyou/server && go build -o /usr/local/bin/asyou-server ./cmd/server
sudo systemctl restart asyou-server
```

### 10.2 Firewall

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP (ACME)
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 7000/tcp  # frps
sudo ufw enable
```

### 10.3 Automatic Backups

```bash
sudo tee /etc/cron.daily/asyou-backup << 'EOF'
#!/bin/bash
cp /var/lib/asyou/asyou.db /var/backups/asyou/asyou-$(date +%Y%m%d).db
find /var/backups/asyou -name "asyou-*.db" -mtime +30 -delete
EOF
sudo chmod +x /etc/cron.daily/asyou-backup
```

---

## 11. Verification Checklist

After deployment, verify each item:

| Check | Command | Expected Result |
|-------|---------|-----------------|
| asyou running | `systemctl status asyou-server` | `active (running)` |
| frps running | `systemctl status frps` | `active (running)` |
| API reachable | `curl https://asyou.example.com/api/v1/version` | Returns JSON |
| Registration | `curl .../auth/register` | `201 Created` |
| Node created | `curl .../nodes` | Returns node list |
| SSL valid | `curl -I https://asyou.example.com` | `HTTP/2 200` |
| Port open | `nc -zv SERVER_IP 7000` | `open` |

---

## 12. Monitoring

### Prometheus Metrics

```bash
curl https://asyou.example.com/api/v1/metrics
```

Sample output:
```
# HELP asyou_nodes_total Total number of nodes
# TYPE asyou_nodes_total gauge
asyou_nodes_total 1
# HELP asyou_proxies_running Number of running proxies
# TYPE asyou_proxies_running gauge
asyou_proxies_running 3
```

Configure Prometheus to scrape this endpoint periodically, and visualize with Grafana.

---

## 13. FAQ

**Q: Getting "connection refused" after deployment?**  
A: Check if the firewall has opened the required ports: `sudo ufw status`. Check if the service is running: `systemctl status asyou-server`.

**Q: API returns 502 Bad Gateway?**  
A: The Nginx proxy configuration is incorrect. Verify that `proxy_pass http://127.0.0.1:8080;` points to the same port asyou is actually listening on.

**Q: SSE connection fails?**  
A: Make sure the Nginx config for `/api/v1/events` includes `proxy_buffering off;`.

**Q: How do I migrate the database to a new server?**  
A: Copy the `/var/lib/asyou/asyou.db` file to the same directory on the new server.

**Q: How do I update the asyou version?**  
A:
```bash
cd /opt/asyou && git pull
cd server && go build -o /usr/local/bin/asyou-server ./cmd/server
sudo systemctl restart asyou-server
```
