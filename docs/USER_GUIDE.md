# asyou User Guide

**Version:** v0.1  
**Last updated:** 2026-06-16  

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Quick Start](#2-quick-start)
3. [Installation](#3-installation)
4. [Web Dashboard](#4-web-dashboard)
   - [4.1 Starting the Dashboard](#41-starting-the-dashboard)
   - [4.2 First Login / Register](#42-first-login--register)
   - [4.3 Layout](#43-layout)
   - [4.4 Tunnel Management (Home)](#44-tunnel-management-home)
   - [4.5 Tunnel Details](#45-tunnel-details)
   - [4.6 Node Management](#46-node-management)
   - [4.7 API Key Management](#47-api-key-management)
   - [4.8 Audit Logs](#48-audit-logs)
   - [4.9 Real-time Updates (SSE)](#49-real-time-updates-sse)
   - [4.10 Common Scenarios](#410-common-scenarios)
5. [CLI Reference](#5-cli-reference)
6. [Desktop App](#6-desktop-app)
7. [API Usage](#7-api-usage)
8. [SDK Quickstart](#8-sdk-quickstart)
9. [Node Management](#9-node-management)
10. [Tunnel Lifecycle](#10-tunnel-lifecycle)
11. [Traffic Monitoring](#11-traffic-monitoring)
12. [TLS Certificates](#12-tls-certificates)
13. [Multi-Tenancy & Roles](#13-multi-tenancy--roles)
14. [Global Scheduling](#14-global-scheduling)
15. [Real-time Events](#15-real-time-events)
16. [Troubleshooting](#16-troubleshooting)
17. [FAQ](#17-faq)

---

## 1. Introduction

asyou is a self-hosted tunnel management platform built on [frp](https://github.com/fatedier/frp). It allows you to expose local services (web apps, APIs, databases) to the internet through secure tunnels, with a management layer for authentication, monitoring, and multi-node orchestration.

### 1.1 What Problem Does It Solve?

- **Developer Preview**: Share your local development server with clients or teammates
- **Webhook Testing**: Expose local webhooks to services like GitHub, Stripe, Slack
- **IoT Access**: Remotely access devices behind NAT
- **Multi-Region**: Route traffic through the geographically closest frps node

### 1.2 Key Concepts

| Concept | Description |
|---|---|
| **Tunnel (Proxy)** | A connection that forwards traffic from a public frps port to your local service |
| **Node** | An frps server that accepts tunnel connections |
| **frpc** | The frp client binary that runs on your machine, managed by asyou |
| **frps** | The frp server binary that receives tunnel traffic |
| **ACME** | Certificate provisioning via Let's Encrypt |
| **SSE** | Server-Sent Events for real-time status updates |

---

## 2. Quick Start

### 2.1 One-Minute Demo

```bash
# 1. Make sure you have frpc available
/tmp/frpc --version

# 2. Start the asyou server
cd server && go run ./cmd/server &

# 3. Register a user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"me@example.com","password":"secret123","display_name":"Me"}'

# 4. Login
LOGIN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"me@example.com","password":"secret123"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

# 5. Use the CLI to expose a service
# (first build the CLI)
cd cli && go build -o /tmp/asyou .
/tmp/asyou login me@example.com secret123
/tmp/asyou expose 3000 --n my-app
```

Your local service on port 3000 is now tunneled through frps.

---

## 3. Installation

### 3.1 Prerequisites

| Requirement | Version | Check |
|---|---|---|
| Go | 1.20+ | `go version` |
| Node.js (for Web UI) | 18+ | `node --version` |
| frp binaries | 0.69.x | `/tmp/frpc --version` |

### 3.2 Get frp Binaries

```bash
VER="0.69.1"
cd /tmp
curl -sL "https://github.com/fatedier/frp/releases/download/v${VER}/frp_${VER}_linux_amd64.tar.gz" -o frp.tar.gz
tar xzf frp.tar.gz
cp "frp_${VER}_linux_amd64/frps" /tmp/frps
cp "frp_${VER}_linux_amd64/frpc" /tmp/frpc
chmod +x /tmp/frps /tmp/frpc
rm -rf "frp_${VER}_linux_amd64" frp.tar.gz
```

### 3.3 Build the Server

```bash
cd server && go build -o /tmp/asyou-server ./cmd/server
```

### 3.4 Build the CLI

```bash
cd cli && go build -o /tmp/asyou .
```

### 3.5 Build the Web Dashboard

```bash
cd web && npm install && npm run build
```

### 3.6 Build the Desktop App (Wails)

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

cd desktop && wails build
# Binary: desktop/build/bin/asyou-desktop.exe
```

---

## 4. Web Dashboard

asyou's Dashboard is a React + Vite based web management interface that provides graphical management of tunnels, nodes, API keys, and audit logs, with real-time status updates via SSE (Server-Sent Events).

> **Prerequisite**: The Dashboard requires the asyou management server (`server`) to be running. See [2. Quick Start](#2-quick-start) to start the server.

### 4.1 Starting the Dashboard

```bash
cd web && npm run dev
# Dev server → http://localhost:5173
```

In development mode, Vite automatically proxies `/api` requests to `http://localhost:8080` (the asyou management server).  
For production, refer to [DEPLOY.md](./DEPLOY.md) for Nginx reverse proxy setup with the static build output.

```bash
# Production build
cd web && npm run build
# Output: web/dist/
```

### 4.2 First Login / Register

Open `http://localhost:5173` to see the **Login / Register page**:

| Scenario | Action |
|----------|--------|
| **Already have an account** | Enter Email + Password → Click **Sign In** |
| **First time** | Click **Register** to switch to sign-up mode → fill in Email, Password, Display Name → Click **Register** |
| **Successful login** | Automatically redirects to the Dashboard home; your email is shown in the top-right with a logout button |

**Auth mechanism**: After login, the JWT token is stored in `localStorage`. All subsequent API requests automatically include the `Authorization: Bearer <token>` header.

### 4.3 Layout

After logging in, the Dashboard has a sidebar on the left and the main content area on the right.

**Sidebar** — four main pages:

| Icon | Page | Route | Description |
|------|------|-------|-------------|
| ◉ | **Proxies** | `/` | Home page — tunnel list, creation & traffic chart |
| ◎ | **Nodes** | `/nodes` | frps node management & health monitoring |
| ◎ | **Audit Logs** | `/audit-logs` | Operation history |
| ◎ | **API Keys** | `/api-keys` | API key management |

**Top bar** (visible on all pages): current user email + logout button.

---

### 4.4 Tunnel Management (Home)

**Proxies** is the core page of the Dashboard, default route `/`.

#### 4.4.1 Stats Bar

The top of the page shows four summary metrics, fetched from the API on page load and updated in real-time:

| Metric | Meaning |
|--------|---------|
| **Total Proxies** | Total tunnels (for the current user, or all tunnels for admins) |
| **Running** | Number of tunnels with `running` status |
| **Stopped** | Number of tunnels with `stopped` or `error` status |
| **Nodes** | Total registered frps nodes |

#### 4.4.2 Tunnel List

Below the stats bar is a table of tunnels, each row representing one tunnel:

| Column | Description |
|--------|-------------|
| **Name** | Tunnel name (click to go to detail page) |
| **Type** | Protocol: `tcp` / `http` / `https` / `udp` |
| **Local Address** | Local service address, e.g. `127.0.0.1:3000` |
| **Remote Port** | Port assigned by frps (if running) |
| **Status** | Status badge: `running` (green), `stopped` (gray), `error` (red) |
| **Actions** | ▶ Start / ⏹ Stop / ↻ Reload buttons |

> **Real-time updates**: The tunnel list listens for `proxy_update` SSE events. When other clients (CLI, Desktop) create, delete, start, or stop tunnels, the list refreshes automatically — no page reload needed.

#### 4.4.3 Creating a Tunnel

Click the **"+ New Tunnel"** button to expand the inline creation form:

| Field | Required | Description |
|-------|----------|-------------|
| **Name** | Yes | A friendly identifier (e.g. `my-web-app`) |
| **Type** | Yes | Protocol: TCP / HTTP / HTTPS / UDP |
| **Local IP** | No | Local IP, defaults to `127.0.0.1` |
| **Local Port** | Yes | The port your local service runs on |
| **Remote Port** | No | Request a specific port on frps (leave empty for auto-assignment) |
| **Subdomain** | No | Subdomain for HTTP tunnels |
| **Node** | Yes | Target frps node (dropdown populated from the API) |

Click **Create** — on success, a new entry appears in the tunnel list with `stopped` status.

> **One-step start**: After creation, you need to manually click ▶ to start the tunnel. The CLI's `asyou expose` command combines create + start in one step.

#### 4.4.4 Tunnel Operations

| Action | Button | Description |
|--------|--------|-------------|
| **Start** | ▶ | Launches the frpc process; status becomes `running` |
| **Stop** | ⏹ | Terminates the frpc process; status becomes `stopped` |
| **Reload** | ↻ | Re-reads config and restarts frpc; useful after config changes |
| **Delete** | 🗑 | Permanently removes the tunnel (available on the detail page) |

> Each operation:
> 1. Calls the corresponding API (`POST /proxies/:id/start|stop|reload`)
> 2. The server manages the frpc process
> 3. State changes are broadcast via SSE
> 4. The frontend automatically refreshes the list

---

### 4.5 Tunnel Details

Click any tunnel **Name** in the list to enter the detail page (route `/proxies/:id`).

#### 4.5.1 Info Panel

| Field | Description |
|-------|-------------|
| **ID** | Unique tunnel identifier |
| **Name** | Tunnel name |
| **Type** | Protocol type |
| **Local Address** | Full local address in `ip:port` format |
| **Remote Port** | Public-facing port |
| **Node** | Target frps node name |
| **Subdomain** | Subdomain for HTTP tunnels (if applicable) |
| **Status** | Current status (color-coded) |
| **Created At** | Creation timestamp |
| **Updated At** | Last update timestamp |

#### 4.5.2 Error Annotations

If a tunnel fails to start, the server records an error message displayed in **red** on the detail page. Common errors:

- `frpc binary not found` — frpc is not installed on the server
- `connection refused` — local service is not running or the address is wrong
- `port already in use` — the remote port is already taken
- `authentication failed` — frps auth token mismatch

Error info is retrieved from the API via the `annotations` field, formatted as a JSON object like `{"error": "connection refused: 127.0.0.1:3000"}`.

#### 4.5.3 Action Buttons

The detail page provides the same ▶ Start / ⏹ Stop / ↻ Reload buttons, plus a red **Delete** button (redirects to the list after deletion).

#### 4.5.4 Traffic Chart

The bottom of the detail page shows a **real-time traffic chart** (rendered with Recharts).

- **Data source**: SSE `stats_update` events + historical data loaded via `GET /proxies/:id/stats`
- **Refresh interval**: The server polls the frps admin API every 10 seconds
- **Metrics**: **KB/s In** (inbound traffic, green line) and **KB/s Out** (outbound traffic, blue line)
- **Time range**: Shows data from the most recent minutes

---

### 4.6 Node Management

Navigate to **Nodes** in the sidebar (route `/nodes`).

#### 4.6.1 Node List

Displays all registered frps nodes:

| Column | Description |
|--------|-------------|
| **ID** | Node ID |
| **Name** | Node name |
| **Host** | Node host address |
| **API Port** | frps admin API port |
| **Bind Port** | frps tunnel bind port |
| **Status** | Online status (determined by heartbeat) |
| **Actions** | Delete button |

#### 4.6.2 Adding a Node

| Field | Required | Description |
|-------|----------|-------------|
| **Name** | Yes | Node name |
| **Host** | Yes | IP address or domain |
| **Bind Port** | Yes | frps `bind_port`, default `7000` |
| **API Port** | No | frps admin API port (`--admin_port`) |
| **Auth Token** | No | frps `token` authentication (if configured) |
| **TLS** | No | Whether TLS is enabled |
| **Region** | No | Geographic region (e.g. `us-east`, `ap-southeast`) |
| **Country** | No | Country |
| **City** | No | City |
| **Lat/Lng** | No | GPS coordinates, used for geo-proximity scheduling |

---

### 4.7 API Key Management

Navigate to **API Keys** in the sidebar (route `/api-keys`).

#### 4.7.1 Creating a Key

1. Click **"+ New API Key"**
2. Enter a name (e.g. `ci-token`)
3. Click **Create**
4. **The key is shown only once!** Copy and save it immediately

```
⚠️ Security notice: The raw key cannot be viewed again after closing the dialog.
Treat your API key like a password.
```

#### 4.7.2 Key List

| Column | Description |
|--------|-------------|
| **Name** | Key name |
| **Prefix** | First 8 characters of the raw key (for identification) |
| **Created At** | Creation timestamp |
| **Revoked** | Whether the key has been revoked |
| **Actions** | Revoke button |

#### 4.7.3 Revoking a Key

If a key is compromised or no longer needed, click **Revoke**. After revocation:
- The key cannot be recovered
- API requests using this key receive `401 Unauthorized`
- The list shows `Revoked: Yes`

> **API Key usage**: Used for non-interactive authentication with CLI and SDK.  
> See [5. CLI Reference](#5-cli-reference) and [8. SDK Quickstart](#8-sdk-quickstart).

---

### 4.8 Audit Logs

Navigate to **Audit Logs** in the sidebar (route `/audit-logs`).

#### 4.8.1 Log List

| Column | Description |
|--------|-------------|
| **Time** | Operation timestamp |
| **Action** | Action type (see below) |
| **Resource** | Resource type + ID |
| **Actor** | Operator (user ID or API key name) |
| **IP** | Request source IP |

#### 4.8.2 Action Types

| Action | Description |
|--------|-------------|
| `user.register` | User registration |
| `user.login` | User login |
| `proxy.create` | Tunnel created |
| `proxy.start` | Tunnel started |
| `proxy.stop` | Tunnel stopped |
| `proxy.reload` | Tunnel reloaded |
| `proxy.delete` | Tunnel deleted |
| `api_key.create` | API key created |
| `api_key.revoke` | API key revoked |
| `cert.provision` | TLS certificate provisioned |
| `cert.delete` | Certificate deleted |

> **Note**: Audit logs are visible to all users, but regular users only see their own actions, while admins see all users' actions (multi-tenancy).

---

### 4.9 Real-time Updates (SSE)

All Dashboard pages receive real-time data pushes via SSE (Server-Sent Events), without needing WebSocket or polling.

#### Connection

The browser automatically connects via `EventSource` to:

```
GET /api/v1/events?token=<jwt_token>
```

#### Event Types

| Event Name | Trigger | Pages That Auto-Refresh |
|------------|---------|--------------------------|
| `proxy_update` | Tunnel create/start/stop/reload/delete | Tunnel list, detail page |
| `stats_update` | Stats push every 10 seconds | Traffic chart |
| `*` (wildcard) | All events | — |

#### Auto-Recovery

The SSE connection has built-in auto-reconnect. When the network drops or the server restarts, the browser automatically reconnects and page data syncs back.

---

### 4.10 Common Scenarios

#### 4.10.1 One-Click Expose via Dashboard

```bash
# Equivalent steps through the Dashboard:
# 1. Log into the Dashboard
# 2. Fill in tunnel details (Name=my-app, Type=tcp, Local Port=3000, pick a Node)
# 3. Click Create
# 4. Click ▶ in the list to start
# 5. Access http://<node-host>:<remote-port>
```

#### 4.10.2 Troubleshooting Tunnel Start Failures

1. Click the tunnel name to enter the detail page
2. Check if **Status** shows red `error`
3. Read the error annotation (e.g. `connection refused`)
4. Verify the local service is running: `curl http://localhost:3000`
5. Confirm frpc is installed on the server: `which frpc`
6. Retry starting the tunnel

#### 4.10.3 Monitoring Real-Time Traffic

- The home page shows an **aggregate traffic chart** for all tunnels at the bottom
- Click a tunnel name to see its **individual traffic chart**
- Metrics include inbound and outbound rates (KB/s), refreshed every 10 seconds

#### 4.10.4 Managing Multiple Nodes

- Deploy and start frps on each node machine ahead of time
- Register each frps on the Dashboard **Nodes** page
- Select different nodes when creating tunnels to distribute traffic

---

## 5. CLI Reference

### 5.1 Installation

```bash
cd cli && go build -o /usr/local/bin/asyou .
```

### 5.2 Commands

#### `asyou login [--s <url>] <email> <password>`

Login and save credentials to `~/.config/asyou/cli-config.json`.

```bash
# Default server (localhost:8080)
asyou login admin@example.com mypassword

# Custom server
asyou login --s https://asyou.example.com admin@example.com mypassword
```

#### `asyou logout`

Remove saved credentials.

```bash
asyou logout
```

#### `asyou expose [--n <name>] [--node <id>] <local_port>`

Create a tunnel and start it in one step — the "one-click expose".

```bash
# Expose port 3000 with auto-generated name
asyou expose 3000

# Expose with custom name on a specific node
asyou expose 8080 --n my-web-app --node 2
```

#### `asyou list`

List all tunnels with their status.

```bash
$ asyou list
ID   Name                 Type     Port   Status
1    my-web-app           tcp      8080   running
2    api-backend          tcp      4000   stopped
```

#### `asyou delete <id>`

Delete a tunnel by ID.

```bash
asyou delete 2
# → Tunnel #2 deleted
```

#### `asyou nodes`

List all registered frps nodes.

```bash
$ asyou nodes
ID   Name                 Host             Port
1    tokyo-1              203.0.113.1      7000
2    frankfurt            198.51.100.1     7000
```

### 5.3 Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | General error (auth, network, etc.) |

---

## 6. Desktop App

### 6.1 Building

```bash
cd desktop && wails build
```

### 6.2 Features

- **One-Click Tunnel**: Select a discovered local port → auto-creates and starts a tunnel
- **Port Discovery**: Scans localhost for listening ports
- **Tunnel Management**: Start/stop tunnels, view status
- **System Tray**: Minimize to tray (via Wails)
- **Auto-login**: Credentials persisted in config file

### 6.3 Quick Tunnel Flow

1. Launch `asyou-desktop.exe`
2. Login with your asyou server credentials
3. Click **Scan Ports** to discover local services
4. Click a port from the list
5. Click **Create & Start** — done!

---

## 7. API Usage

### 7.1 Authentication

All API calls (except register/login) require authentication via:

**JWT Bearer Token** (24h expiry):
```bash
# Obtain token
curl -s http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"me@example.com","password":"pass"}' | jq .access_token

# Use in requests
curl http://localhost:8080/api/v1/proxies \
  -H "Authorization: Bearer $TOKEN"
```

**API Key** (persistent):
```bash
# Create via API
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-key"}' | jq .token

# Use in requests
curl http://localhost:8080/api/v1/proxies \
  -H "X-Api-Key: asyou_abc123..."
```

### 7.2 Common Workflows

#### Expose a Service

```bash
# 1. Register node (one-time)
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-node","host":"203.0.113.1","bind_port":7000}'

# 2. Create proxy
curl -X POST http://localhost:8080/api/v1/proxies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-app","type":"tcp","local_port":3000,"node_id":1}'

# 3. Start tunnel
curl -X POST http://localhost:8080/api/v1/proxies/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"start"}'

# 4. Status
curl http://localhost:8080/api/v1/proxies/1 \
  -H "Authorization: Bearer $TOKEN" | jq .status
# → "running"
```

#### Monitor Traffic

```bash
# Get stats
curl http://localhost:8080/api/v1/proxies/1/stats?limit=10 \
  -H "Authorization: Bearer $TOKEN" | jq
```

#### Listen to Real-time Events

```bash
# Using curl (SSE)
curl -N http://localhost:8080/api/v1/events?token=$JWT

# Using the hooks in web/desktop (automatic)
```

### 7.3 Error Handling

Always check the response body for errors:
```json
{
  "error": "failed to start proxy: start frpc: exec: \"frpc\": executable file not found in $PATH",
  "code": "INTERNAL"
}
```

Common errors:

| HTTP Status | Code | Meaning |
|---|---|---|
| 400 | `BAD_REQUEST` | Missing or invalid parameters |
| 401 | `UNAUTHORIZED` | Missing/expired/invalid token |
| 403 | `FORBIDDEN` | Resource doesn't belong to you |
| 404 | `NOT_FOUND` | Resource not found |
| 500 | `INTERNAL` | Server error (check logs) |

---

## 8. SDK Quickstart

### 8.1 Go SDK

```go
package main

import (
    "fmt"
    "github.com/asyou/sdk-go"
)

func main() {
    client := asyou.NewClient("http://localhost:8080")
    client.Login("me@example.com", "password")

    // One-click expose
    proxy, _ := client.CreateProxy("my-app", "tcp", 3000, 0)
    client.ProxyAction(proxy.ID, "start")
    fmt.Printf("Tunnel #%d: %s\n", proxy.ID, proxy.Status)

    // List
    proxies, _ := client.ListProxies()
    for _, p := range proxies {
        fmt.Printf("%d %s %s\n", p.ID, p.Name, p.Status)
    }
}
```

### 8.2 Python SDK

```python
from asyou import Client

client = Client("http://localhost:8080")
client.login("me@example.com", "password")

# One-click expose
proxy = client.expose(3000, name="my-app")
print(f"Tunnel #{proxy.id}: {proxy.status}")

# List tunnels
for p in client.list_proxies():
    print(f"[{p.id}] {p.name}: {p.status}")
```

### 8.3 Node.js SDK

```typescript
import { Client } from 'asyou-sdk'

const client = new Client('http://localhost:8080')
await client.login('me@example.com', 'password')

// One-click expose
const proxy = await client.expose(3000, 'my-app')
console.log(`Tunnel #${proxy.id}: ${proxy.status}`)

// List
const proxies = await client.listProxies()
for (const p of proxies) {
    console.log(`[${p.id}] ${p.name}: ${p.status}`)
}
```

---

## 9. Node Management

### 9.1 What is a Node?

A **node** is an frps server that accepts incoming tunnel connections. You need at least one node registered for tunnels to work. In a multi-region setup, you can have many nodes around the world.

### 9.2 Registering a Node

```bash
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "tokyo-1",
    "host": "203.0.113.1",
    "bind_port": 7000,
    "region": "ap-northeast",
    "country": "JP",
    "city": "Tokyo",
    "latitude": 35.6762,
    "longitude": 139.6503,
    "max_connections": 200,
    "weight": 1.2
  }'
```

### 9.3 Node Heartbeat

Nodes send periodic heartbeats to report health. This can be done from the frps machine:

```bash
curl -X POST http://<asyou-server>:8080/api/v1/nodes/1/heartbeat \
  -H "X-Node-Token: <auth_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "connections": 15,
    "cpu_load": 0.45,
    "memory_usage": 0.6,
    "latency_ms": 23,
    "bandwidth": 850
  }'
```

### 9.4 Finding the Best Node

For automatic node selection:

```bash
# Based on geographic preference
curl "http://localhost:8080/api/v1/nodes/best?region=ap-northeast&lat=35.68&lng=139.65"
```

The response includes the `score` field showing the computed rating.

---

## 10. Tunnel Lifecycle

### 10.1 States

```
stopped ──► running ──► stopped
   ▲          │
   │          ├── (process exits) ──► stopped
   │          └── (reload) ──► stop ──► start ──► running
   │
   └── (delete) ──► removed
```

### 10.2 Starting

When you start a tunnel:

1. asyou server generates an frpc config file at `/tmp/asyou-proxy-{id}-*.ini`
2. Launches `frpc -c <config>` as a child process
3. On success: proxy status → `"running"`
4. On failure: error captured in `annotations` field as JSON:
   ```json
   {"error": "start frpc: exec: \"frpc\": not found", "when": "start"}
   ```

### 10.3 Stopping

1. Sends SIGKILL to the frpc process
2. Status → `"stopped"`
3. Config file is cleaned up

### 10.4 Reloading

1. Stops the current frpc process
2. Generates a fresh config (picks up any changes)
3. Starts a new frpc process
4. Status → `"running"`

### 10.5 Error Handling

If start/stop/reload fails, the error is stored as a JSON annotation:

```json
{
  "error": "frpc executable not found in $PATH",
  "when": "start"
}
```

View it via the API:
```bash
curl http://localhost:8080/api/v1/proxies/1 \
  -H "Authorization: Bearer $TOKEN" | jq .annotations
```

---

## 11. Traffic Monitoring

### 11.1 REST API

```bash
# Get last 60 stats entries
curl "http://localhost:8080/api/v1/proxies/1/stats?limit=60" \
  -H "Authorization: Bearer $TOKEN" | jq
```

Response:
```json
[
    {
        "id": 1,
        "proxy_id": 1,
        "timestamp": "2026-06-16T10:00:00Z",
        "bytes_in": 1048576,
        "bytes_out": 524288,
        "conn_count": 5
    }
]
```

### 11.2 Ingesting Stats (from frps node)

```bash
curl -X POST http://<asyou-server>:8080/api/v1/proxies/1/stats \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bytes_in": 1024, "bytes_out": 512, "conn_count": 3}'
```

### 11.3 Real-time via SSE

The Web Dashboard automatically receives stats via SSE at 10-second intervals. You can also connect directly:

```bash
curl -N "http://localhost:8080/api/v1/events?token=$JWT"
```

### 11.4 Prometheus Metrics

```bash
curl http://localhost:8080/api/v1/metrics
```

Returns:
```
# HELP asyou_users_total Total number of registered users
# TYPE asyou_users_total gauge
asyou_users_total 42
# HELP asyou_nodes_total Total number of nodes
# TYPE asyou_nodes_total gauge
asyou_nodes_total 3
# HELP asyou_proxies_total Total number of proxies
# TYPE asyou_proxies_total gauge
asyou_proxies_total 15
# HELP asyou_proxies_running Number of running proxies
# TYPE asyou_proxies_running gauge
asyou_proxies_running 8
```

---

## 12. TLS Certificates

### 12.1 Automated Provisioning (ACME)

```bash
POST /api/v1/certs/provision
Authorization: Bearer $TOKEN
Content-Type: application/json

{
    "proxy_id": 1,
    "domain": "app.example.com"
}
```

**Requirements:**
- The domain must resolve to your frps server's public IP
- Your frps server must be reachable on port 80 (for HTTP-01 challenge)
- Response includes the expiry date and issuer

### 12.2 Managing Certificates

```bash
# List all certificates
curl http://localhost:8080/api/v1/certs \
  -H "Authorization: Bearer $TOKEN" | jq

# Get certificate details
curl http://localhost:8080/api/v1/certs/1 \
  -H "Authorization: Bearer $TOKEN" | jq

# Delete a certificate
curl -X DELETE http://localhost:8080/api/v1/certs/1 \
  -H "Authorization: Bearer $TOKEN"
```

### 12.3 Using External ACME Clients

For production, you may prefer using certbot or acme.sh:

```bash
# Get certificate with certbot
certbot certonly --standalone -d app.example.com

# Then upload via API (future feature)
# For now, store the certificate files manually and configure frps
```

---

## 13. Multi-Tenancy & Roles

### 13.1 User Roles

| Role | Permissions |
|---|---|
| `user` | Manage own proxies, view own audit logs, manage own API keys |
| `admin` | All user permissions + view all resources + manage all tenants |

### 13.2 Tenant Isolation

- Regular users **only see their own** proxies, certificates, and API keys
- Admins can **see all** resources across all users
- All queries are automatically scoped by `user_id`
- Proxy names must be **unique per user**

### 13.3 Admin API

Admins can access any user's resources:
```bash
# Admin sees all proxies
curl http://localhost:8080/api/v1/proxies \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## 14. Global Scheduling

### 14.1 How It Works

The scheduler scores each active node based on:
- **Current load**: Connection count vs maximum
- **Latency**: Recent heartbeat latency
- **Geographic proximity**: Distance from preferred location
- **Region match**: Exact region preference
- **Weight**: Admin-configured multiplier

### 14.2 Getting the Best Node

```bash
# Simple: just get the best node
curl http://localhost:8080/api/v1/nodes/best

# With geographic preference (e.g., user in Tokyo)
curl "http://localhost:8080/api/v1/nodes/best?region=ap-northeast&lat=35.68&lng=139.65"

# Override max distance
curl "http://localhost:8080/api/v1/nodes/best?lat=35.68&lng=139.65&max_distance=2000"
```

### 14.3 Node Weights

Set higher weights for more powerful nodes:
```bash
curl -X PUT http://localhost:8080/api/v1/nodes/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"weight": 2.0}'
```

---

## 15. Real-time Events

### 15.1 SSE Stream

The server pushes events via Server-Sent Events at `/api/v1/events`.

**Event types:**

| Type | Triggered | Data |
|---|---|---|
| `connected` | On connect | `{ user_id }` |
| `proxy_update` | Status change | `{ id, status, annotations, timestamp }` |
| `stats_update` | Every 10s | `[{ proxy_id, bytes_in, bytes_out, conn_count }]` |

### 15.2 Connecting Programmatically

```javascript
// Browser JavaScript
const token = localStorage.getItem('asyou_token')
const source = new EventSource(`/api/v1/events?token=${token}`)

source.addEventListener('proxy_update', (e) => {
    const data = JSON.parse(e.data)
    console.log('Proxy status changed:', data)
})

source.addEventListener('stats_update', (e) => {
    const data = JSON.parse(e.data)
    console.log('Stats:', data)
})
```

```python
import requests

response = requests.get(
    'http://localhost:8080/api/v1/events',
    params={'token': token},
    stream=True
)
for line in response.iter_lines():
    if line:
        print(line)
```

---

## 16. Troubleshooting

### 16.0 Version Mismatch

**Problem:** frpc version doesn't match frps version, causing connection failures.

**Check:**
```bash
# Check what version the server recommends
asyou version

# Check what version you have locally
asyou check

# Or manually
/tmp/frpc --version
# Get the expected version from the server
curl http://localhost:8080/api/v1/version
```

**Solutions:**
1. **Download matching frpc** from the GitHub releases page:
   ```bash
   # Match the frps version that your node reports
   VER="0.69.1"  # replace with version from 'asyou version'
   cd /tmp
   curl -sL "https://github.com/fatedier/frp/releases/download/v${VER}/frp_${VER}_linux_amd64.tar.gz" -o frp.tar.gz
   tar xzf frp.tar.gz
   cp "frp_${VER}_linux_amd64/frpc" /usr/local/bin/frpc
   rm -rf "frp_${VER}_linux_amd64" frp.tar.gz
   ```

2. **The server auto-detects** the frpc path via `findFrpc()`. It checks:
   - `frpc` (PATH)
   - `/usr/local/bin/frpc`
   - `/usr/bin/frpc`
   - `/tmp/frpc`
   - `./frpc` (current directory)

3. **If you manage multiple machines**, set up a shared download script:
   ```bash
   #!/bin/bash
   # save as sync-frpc.sh
   SERVER="http://your-asyou-server:8080"
   JSON=$(curl -s "$SERVER/api/v1/version")
   VER=$(echo "$JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['recommended_frpc_version'])")
   echo "Installing frpc v$VER..."
   curl -sL "https://github.com/fatedier/frp/releases/download/v${VER}/frp_${VER}_linux_amd64.tar.gz" -o /tmp/frp.tar.gz
   tar xzf /tmp/frp.tar.gz -C /tmp
   sudo cp "/tmp/frp_${VER}_linux_amd64/frpc" /usr/local/bin/frpc
   rm -rf "/tmp/frp_${VER}_linux_amd64" /tmp/frp.tar.gz
   /usr/local/bin/frpc --version
   ```

4. **Automatic version sync** (via heartbeat):
   When a node sends its heartbeat, it can include `frp_version`:
   ```bash
   curl -X POST http://localhost:8080/api/v1/nodes/1/heartbeat \
     -H "X-Node-Token: <token>" \
     -H "Content-Type: application/json" \
     -d '{"frp_version": "0.69.1", "connections": 5}'
   ```
   The server records this version and uses it as the recommended version for new frpc instances.

### 16.1 "frpc: executable file not found"

```
Error: failed to start proxy: start frpc: exec: "frpc": executable file not found in $PATH
```

**Solutions:**
1. Install frpc: `cp /tmp/frpc /usr/local/bin/frpc`
2. Or place frpc at one of: `/tmp/frpc`, `./frpc`, `/usr/bin/frpc`
3. The server checks these locations automatically via `findFrpc()`

### 16.2 "connection refused"

```
Error: connect: connection refused
```

**Check:**
1. Is your local service actually running on the specified port?
2. `curl http://localhost:8802/` — does it respond?
3. Is frps running? `ps aux | grep frps`

### 16.3 "port already in use"

```
Error: listen tcp :8080: bind: address already in use
```

**Solutions:**
```bash
# Find what's using the port
fuser 8080/tcp

# Kill it
fuser -k 8080/tcp
```

### 16.4 Tunnel status stuck on "stopped"

1. Check the proxy's annotations for error details:
   ```bash
   curl http://localhost:8080/api/v1/proxies/1 -H "Bearer $TOKEN" | jq .annotations
   ```
2. Verify frpc binary exists at a known location
3. Check server logs: `cat /tmp/asyou-server.log`

### 16.5 Authentication errors

```
{"code":"UNAUTHORIZED","error":"missing or invalid authorization header"}
```

**Solutions:**
1. Token expired (24h) — login again
2. Missing `Bearer ` prefix — ensure format is `Authorization: Bearer <token>`
3. Wrong API key — create a new one from the dashboard

### 16.6 Connection lost to SSE

SSE auto-reconnects after 3 seconds. If it fails continuously:
1. Check that `/api/v1/events` is accessible
2. Ensure the token hasn't expired
3. Check for CORS issues if using a custom frontend

---

## 17. FAQ

**Q: Can I use asyou without frp?**  
A: No. asyou is a management layer on top of frp. You need frps and frpc binaries.

**Q: Do I need a public IP?**  
A: Only the frps server needs a public IP. The frpc client (your local machine) can be behind NAT.

**Q: How many tunnels can I create?**  
A: Limited only by your frps server's capacity (`max_connections` per node).

**Q: Is there a WebSocket for real-time updates?**  
A: asyou uses SSE (Server-Sent Events) instead of WebSocket. SSE is simpler, uses standard HTTP, and works with any existing infrastructure.

**Q: Can I use PostgreSQL instead of SQLite?**  
A: Currently SQLite only. PostgreSQL support can be added — the SQL is standard and queries are simple.

**Q: How do I backup the database?**  
A: Copy `asyou.db` while the server is not writing. The server uses SQLite in WAL mode by default.

**Q: Can I run multiple asyou servers?**  
A: Not currently. The design assumes a single management server with multiple frps nodes.

**Q: How do I update frp?**  
A: Replace the `frps` and `frpc` binaries with the new version. The asyou server auto-detects the binary path.

**Q: Is there rate limiting?**  
A: Not yet. This is planned for a future release.

**Q: Can I use custom domains?**  
A: Yes. Set `custom_domains` when creating an HTTP proxy, and use the certificate provisioning endpoint to get TLS certs.
