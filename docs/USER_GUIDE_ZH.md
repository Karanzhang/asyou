# asyou 用户指南

**版本：** v0.1  
**最后更新：** 2026-06-16

---

## 目录

1. [简介](#1-简介)
2. [快速开始](#2-快速开始)
3. [安装](#3-安装)
4. [Web 仪表盘](#4-web-仪表盘)
   - [4.1 启动仪表盘](#41-启动仪表盘)
   - [4.2 首次登录 / 注册](#42-首次登录--注册)
   - [4.3 界面布局](#43-界面布局)
   - [4.4 隧道管理（主页）](#44-隧道管理主页)
   - [4.5 隧道详情](#45-隧道详情)
   - [4.6 节点管理](#46-节点管理)
   - [4.7 API 密钥管理](#47-api-密钥管理)
   - [4.8 审计日志](#48-审计日志)
   - [4.9 实时更新（SSE）](#49-实时更新sse)
   - [4.10 常见场景](#410-常见场景)
5. [CLI 参考](#5-cli-参考)
6. [桌面客户端](#6-桌面客户端)
7. [API 使用](#7-api-使用)
8. [SDK 快速入门](#8-sdk-快速入门)
9. [节点管理](#9-节点管理)
10. [隧道生命周期](#10-隧道生命周期)
11. [流量监控](#11-流量监控)
12. [TLS 证书](#12-tls-证书)
13. [多租户与角色](#13-多租户与角色)
14. [全局调度](#14-全局调度)
15. [实时事件](#15-实时事件)
16. [故障排除](#16-故障排除)
17. [常见问题](#17-常见问题)

---

## 1. 简介

asyou 是一个基于 [frp](https://github.com/fatedier/frp) 的自托管隧道管理平台。它允许您通过安全的隧道将本地服务（Web 应用、API、数据库等）暴露到互联网，并提供认证、监控和多节点编排等管理能力。

### 1.1 它能解决什么问题？

- **开发者预览**：与客户或队友分享您的本地开发服务器
- **Webhook 测试**：将本地 webhook 暴露给 GitHub、Stripe、Slack 等服务
- **IoT 访问**：远程访问 NAT 后面的设备
- **多区域**：通过地理上最近的 frps 节点路由流量

### 1.2 核心概念

| 概念 | 说明 |
|------|------|
| **隧道 (Tunnel/Proxy)** | 将流量从公网 frps 端口转发到本地服务的连接 |
| **节点 (Node)** | 接受隧道连接的 frps 服务器 |
| **frpc** | 在您的机器上运行的 frp 客户端二进制文件，由 asyou 管理 |
| **frps** | 接收隧道流量的 frp 服务器二进制文件 |
| **ACME** | 通过 Let's Encrypt 自动颁发证书 |
| **SSE** | 服务器推送事件，用于实时状态更新 |

---

## 2. 快速开始

### 2.1 一分钟演示

```bash
# 1. 确保 frpc 可用
/tmp/frpc --version

# 2. 启动 asyou 服务器
cd server && go run ./cmd/server &

# 3. 注册用户
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"me@example.com","password":"secret123","display_name":"Me"}'

# 4. 登录
LOGIN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"me@example.com","password":"secret123"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

# 5. 使用 CLI 暴露服务
# (先构建 CLI)
cd cli && go build -o /tmp/asyou .
/tmp/asyou login me@example.com secret123
/tmp/asyou expose 3000 --n my-app
```

您的 3000 端口本地服务现已通过 frps 隧道转发到公网。

---

## 3. 安装

### 3.1 环境要求

| 要求 | 版本 | 检查方法 |
|------|------|----------|
| Go | 1.20+ | `go version` |
| Node.js（Web UI 需要） | 18+ | `node --version` |
| frp 二进制文件 | 0.69.x | `/tmp/frpc --version` |

### 3.2 获取 frp 二进制文件

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

### 3.3 构建服务器

```bash
cd server && go build -o /tmp/asyou-server ./cmd/server
```

### 3.4 构建 CLI

```bash
cd cli && go build -o /tmp/asyou .
```

### 3.5 构建 Web 仪表盘

```bash
cd web && npm install && npm run build
```

### 3.6 构建桌面客户端 (Wails)

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

cd desktop && wails build
# 二进制文件: desktop/build/bin/asyou-desktop.exe
```

---

## 4. Web 仪表盘

asyou 的仪表盘是一个基于 React + Vite 构建的 Web 管理界面，提供隧道、节点、API 密钥和审计日志的图形化管理，并通过 SSE（Server-Sent Events）实现实时状态更新。

> **先决条件**：仪表盘需要 asyou 管理服务器（`server`）正在运行。参见 [2. 快速开始](#2-快速开始) 启动服务端。

### 4.1 启动仪表盘

```bash
cd web && npm run dev
# 开发服务器 → http://localhost:5173
```

在开发模式下，Vite 会自动将 `/api` 请求代理到 `http://localhost:8080`（asyou 管理服务器）。  
生产环境请参考 [DEPLOY.md](./DEPLOY.md) 使用 Nginx 反向代理配合静态构建产物。

```bash
# 生产构建
cd web && npm run build
# 输出目录：web/dist/
```

### 4.2 首次登录 / 注册

打开 `http://localhost:5173`，您将看到**登录/注册页面**：

| 场景 | 操作 |
|------|------|
| **已有账号** | 输入 Email + Password → 点击 **Sign In** |
| **首次使用** | 点击 **Register** 切换到注册模式 → 填写 Email、Password、Display Name → 点击 **Register** |
| **登录成功** | 自动跳转到仪表盘首页，右上角显示用户邮箱和退出按钮 |

**认证机制**：登录后，JWT Token 保存在 `localStorage` 中。后续所有 API 请求会自动携带 `Authorization: Bearer <token>` 请求头。

### 4.3 界面布局

登录后进入仪表盘，左侧为导航侧边栏，右侧为主内容区。

**侧边栏** — 四个主要页面：

| 图标 | 页面 | 路由 | 说明 |
|------|------|------|------|
| ◉ | **隧道 (Proxies)** | `/` | 首页 — 隧道列表、创建和流量图 |
| ◎ | **节点 (Nodes)** | `/nodes` | frps 节点管理与健康监控 |
| ◎ | **审计 (Audit Logs)** | `/audit-logs` | 操作历史追溯 |
| ◎ | **密钥 (API Keys)** | `/api-keys` | API 密钥管理 |

**顶栏**（所有页面可见）：当前用户邮箱 + 退出登录按钮。

---

### 4.4 隧道管理（主页）

**隧道（Proxies）** 是仪表盘的核心页面，默认路由为 `/`。

#### 4.4.1 统计栏

页面顶部展示四组汇总数据，在页面加载时从 API 获取并实时更新：

| 指标 | 含义 |
|------|------|
| **Total Proxies** | 隧道总数（当前用户，管理员可见全部） |
| **Running** | 状态为 `running` 的隧道数 |
| **Stopped** | 状态为 `stopped` 或 `error` 的隧道数 |
| **Nodes** | 已注册的 frps 节点总数 |

#### 4.4.2 隧道列表

统计栏下方为隧道表格，每行代表一个隧道：

| 列 | 说明 |
|----|------|
| **Name** | 隧道名称（点击进入详情页） |
| **Type** | 协议：`tcp` / `http` / `https` / `udp` |
| **Local Address** | 本地服务地址，例如 `127.0.0.1:3000` |
| **Remote Port** | frps 分配的远程端口（如果已启动） |
| **Status** | 状态标签：`running`（绿色）、`stopped`（灰色）、`error`（红色） |
| **Actions** | ▶ 启动 / ⏹ 停止 / ↻ 重载 按钮 |

> **实时更新**：隧道列表监听 `proxy_update` SSE 事件。当其他客户端（CLI、桌面端）创建、删除、启动或停止隧道时，列表会自动刷新，无需手动刷新页面。

#### 4.4.3 创建隧道

点击 **"+ New Tunnel"** 按钮展开内联创建表单：

| 字段 | 必填 | 说明 |
|------|------|------|
| **Name** | 是 | 隧道名称（例如 `my-web-app`） |
| **Type** | 是 | 协议类型：TCP / HTTP / HTTPS / UDP |
| **Local IP** | 否 | 本地 IP，默认为 `127.0.0.1` |
| **Local Port** | 是 | 本地服务监听的端口 |
| **Remote Port** | 否 | 在 frps 上申请的特定端口（留空则自动分配） |
| **Subdomain** | 否 | HTTP 类型隧道的子域名 |
| **Node** | 是 | 目标 frps 节点（下拉列表数据来自 API） |

点击 **Create** — 成功后，隧道列表中会出现一条新记录，初始状态为 `stopped`。

> **一键启动**：创建后需要手动点击 ▶ 启动隧道。CLI 的 `asyou expose` 命令将创建和启动合并为一步。

#### 4.4.4 隧道操作

| 操作 | 按钮 | 说明 |
|------|------|------|
| **启动 (Start)** | ▶ | 启动 frpc 进程；状态变为 `running` |
| **停止 (Stop)** | ⏹ | 终止 frpc 进程；状态变为 `stopped` |
| **重载 (Reload)** | ↻ | 重新读取配置并重启 frpc，配置变更后使用 |
| **删除 (Delete)** | 🗑 | 永久删除隧道（在详情页中操作） |

> 每次操作：
> 1. 调用对应的 API（`POST /proxies/:id/start|stop|reload`）
> 2. 服务器管理 frpc 进程
> 3. 通过 SSE 广播状态变更
> 4. 前端自动刷新列表

---

### 4.5 隧道详情

点击隧道列表中的任意 **Name** 进入详情页（路由 `/proxies/:id`）。

#### 4.5.1 信息面板

| 字段 | 说明 |
|------|------|
| **ID** | 隧道唯一标识 |
| **Name** | 隧道名称 |
| **Type** | 协议类型 |
| **Local Address** | 完整的本地地址，格式为 `ip:port` |
| **Remote Port** | 公网访问端口 |
| **Node** | 目标 frps 节点名称 |
| **Subdomain** | HTTP 类型隧道的子域名（如适用） |
| **Status** | 当前状态（颜色标识） |
| **Created At** | 创建时间 |
| **Updated At** | 最后更新时间 |

#### 4.5.2 错误标注

如果隧道启动失败，服务器会记录错误信息并在详情页以 **红色** 显示。常见错误：

- `frpc binary not found` — 服务器未安装 frpc
- `connection refused` — 本地服务未运行或地址错误
- `port already in use` — 远程端口已被占用
- `authentication failed` — frps auth token 不匹配

错误信息通过 API 的 `annotations` 字段获取，格式为 JSON 对象，例如 `{"error": "connection refused: 127.0.0.1:3000"}`。

#### 4.5.3 操作按钮

详情页同样提供 ▶ 启动 / ⏹ 停止 / ↻ 重载 三个操作按钮，以及红色的 **Delete** 删除按钮（删除后自动返回列表页）。

#### 4.5.4 流量图

详情页底部展示该隧道的**实时流量图**（使用 Recharts 渲染）。

- **数据来源**：SSE `stats_update` 事件 + 通过 `GET /proxies/:id/stats` 加载的历史数据
- **刷新间隔**：服务器每 10 秒轮询一次 frps 管理 API
- **指标**：**KB/s In**（入站流量，绿色折线）和 **KB/s Out**（出站流量，蓝色折线）
- **时间范围**：展示最近几分钟的数据

---

### 4.6 节点管理

导航到侧边栏 **Nodes**（路由 `/nodes`）。

#### 4.6.1 节点列表

展示所有已注册的 frps 节点：

| 列 | 说明 |
|----|------|
| **ID** | 节点 ID |
| **Name** | 节点名称 |
| **Host** | 节点主机地址 |
| **API Port** | frps 管理 API 端口 |
| **Bind Port** | frps 隧道绑定端口 |
| **Status** | 在线状态（通过心跳判断） |
| **Actions** | 删除按钮 |

#### 4.6.2 添加节点

| 字段 | 必填 | 说明 |
|------|------|------|
| **Name** | 是 | 节点名称 |
| **Host** | 是 | IP 地址或域名 |
| **Bind Port** | 是 | frps `bind_port`，默认 `7000` |
| **API Port** | 否 | frps 管理 API 端口（`--admin_port`） |
| **Auth Token** | 否 | frps `token` 认证（如果已配置） |
| **TLS** | 否 | 是否启用 TLS |
| **Region** | 否 | 地理区域（例如 `us-east`、`ap-southeast`） |
| **Country** | 否 | 国家 |
| **City** | 否 | 城市 |
| **Lat/Lng** | 否 | GPS 坐标，用于地理邻近度调度 |

---

### 4.7 API 密钥管理

导航到侧边栏 **API Keys**（路由 `/api-keys`）。

#### 4.7.1 创建密钥

1. 点击 **"+ New API Key"**
2. 输入名称（例如 `ci-token`）
3. 点击 **Create**
4. **密钥仅显示一次！** 请立即复制并安全保存

```
⚠️ 安全提示：关闭弹窗后将无法再次查看密钥明文。
请像对待密码一样妥善保管 API Key。
```

#### 4.7.2 密钥列表

| 列 | 说明 |
|----|------|
| **Name** | 密钥名称 |
| **Prefix** | 密钥前缀（明文前 8 位），用于识别 |
| **Created At** | 创建时间 |
| **Revoked** | 是否已吊销 |
| **Actions** | 吊销按钮 |

#### 4.7.3 吊销密钥

如果密钥泄露或不再需要，点击 **Revoke** 吊销。吊销后：

- 密钥不可恢复
- 使用该密钥的 API 请求将收到 `401 Unauthorized`
- 列表中状态标记为 `Revoked: Yes`

> **API Key 用途**：用于 CLI 和 SDK 的无交互认证。  
> 参见 [5. CLI 参考](#5-cli-参考) 和 [8. SDK 快速入门](#8-sdk-快速入门)。

---

### 4.8 审计日志

导航到侧边栏 **Audit Logs**（路由 `/audit-logs`）。

#### 4.8.1 日志列表

| 列 | 说明 |
|----|------|
| **Time** | 操作时间 |
| **Action** | 操作类型（见下方） |
| **Resource** | 资源类型 + ID |
| **Actor** | 操作者（用户 ID 或 API Key 名称） |
| **IP** | 请求来源 IP |

#### 4.8.2 操作类型

| 操作 | 说明 |
|------|------|
| `user.register` | 用户注册 |
| `user.login` | 用户登录 |
| `proxy.create` | 隧道已创建 |
| `proxy.start` | 隧道已启动 |
| `proxy.stop` | 隧道已停止 |
| `proxy.reload` | 隧道已重载 |
| `proxy.delete` | 隧道已删除 |
| `api_key.create` | API 密钥已创建 |
| `api_key.revoke` | API 密钥已吊销 |
| `cert.provision` | TLS 证书已颁发 |
| `cert.delete` | 证书已删除 |

> **注意**：审计日志对所有用户可见，但普通用户只能看到自己的操作，管理员可以看到所有用户的操作（多租户）。

---

### 4.9 实时更新（SSE）

仪表盘的所有页面通过 SSE（Server-Sent Events）接收实时数据推送，无需 WebSocket 或轮询。

#### 连接方式

浏览器自动通过 `EventSource` 连接到：

```
GET /api/v1/events?token=<jwt_token>
```

#### 事件类型

| 事件名 | 触发时机 | 自动刷新的页面 |
|--------|----------|---------------|
| `proxy_update` | 隧道创建/启动/停止/重载/删除 | 隧道列表、详情页 |
| `stats_update` | 每 10 秒推送统计数据 | 流量图 |
| `*`（通配符） | 所有事件 | — |

#### 自动恢复

SSE 连接内置自动重连机制。当网络断开或服务器重启时，浏览器会自动重新连接，页面数据会自动同步。

---

### 4.10 常见场景

#### 4.10.1 通过仪表盘一键暴露

```bash
# 在仪表盘中完成的等效步骤：
# 1. 登录仪表盘
# 2. 填写隧道信息（Name=my-app, Type=tcp, Local Port=3000, 选择一个 Node）
# 3. 点击 Create
# 4. 在列表中点击 ▶ 启动
# 5. 访问 http://<node-host>:<remote-port>
```

#### 4.10.2 排查隧道启动失败

1. 点击隧道名称进入详情页
2. 检查 **Status** 是否为红色 `error`
3. 查看错误标注内容（例如 `connection refused`）
4. 确认本地服务正在运行：`curl http://localhost:3000`
5. 确认服务器已安装 frpc：`which frpc`
6. 重新启动隧道

#### 4.10.3 监控实时流量

- 首页底部展示所有隧道的**汇总流量图**
- 点击隧道名称查看其**独立流量图**
- 指标包括入站和出站速率（KB/s），每 10 秒刷新一次

#### 4.10.4 管理多节点

- 提前在每个节点机器上部署并启动 frps
- 在仪表盘 **Nodes** 页面注册每个 frps
- 创建隧道时选择不同的 Node，实现流量分发

---

## 5. CLI 参考

### 5.1 安装

```bash
cd cli && go build -o /usr/local/bin/asyou .
```

### 5.2 命令

#### `asyou login [--s <url>] <email> <password>`

登录并将凭据保存到 `~/.config/asyou/cli-config.json`。

```bash
# 默认服务器 (localhost:8080)
asyou login admin@example.com mypassword

# 自定义服务器
asyou login --s https://asyou.example.com admin@example.com mypassword
```

#### `asyou logout`

删除已保存的凭据。

```bash
asyou logout
```

#### `asyou expose [--n <name>] [--node <id>] <local_port>`

创建并启动隧道 — "一键暴露"。

```bash
# 暴露 3000 端口，自动生成名称
asyou expose 3000

# 自定义名称并指定节点
asyou expose 8080 --n my-web-app --node 2
```

#### `asyou list`

列出所有隧道及其状态。

```bash
$ asyou list
ID   Name                 Type     Port   Status
1    my-web-app           tcp      8080   running
2    api-backend          tcp      4000   stopped
```

#### `asyou delete <id>`

按 ID 删除隧道。

```bash
asyou delete 2
# → Tunnel #2 deleted
```

#### `asyou nodes`

列出所有已注册的 frps 节点。

```bash
$ asyou nodes
ID   Name                 Host             Port
1    tokyo-1              203.0.113.1      7000
2    frankfurt            198.51.100.1     7000
```

### 5.3 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1 | 一般错误（认证、网络等） |

---

## 6. 桌面客户端

### 6.1 构建

```bash
cd desktop && wails build
```

### 6.2 功能

- **一键隧道**：选择已发现的本地端口 → 自动创建并启动隧道
- **端口发现**：扫描 localhost 上的监听端口
- **隧道管理**：启动/停止隧道，查看状态
- **系统托盘**：最小化到托盘（通过 Wails 实现）
- **自动登录**：凭据持久化保存在配置文件中

### 6.3 快速隧道流程

1. 启动 `asyou-desktop.exe`
2. 使用 asyou 服务器凭据登录
3. 点击 **Scan Ports** 发现本地服务
4. 从列表中选择一个端口
5. 点击 **Create & Start** — 完成！

---

## 7. API 使用

### 7.1 认证

所有 API 调用（注册和登录除外）都需要通过以下方式进行认证：

**JWT Bearer Token**（24 小时有效期）：
```bash
# 获取 Token
curl -s http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"me@example.com","password":"pass"}' | jq .access_token

# 在请求中使用
curl http://localhost:8080/api/v1/proxies \
  -H "Authorization: Bearer $TOKEN"
```

**API Key**（持久化）：
```bash
# 通过 API 创建
curl -X POST http://localhost:8080/api/v1/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-key"}' | jq .token

# 在请求中使用
curl http://localhost:8080/api/v1/proxies \
  -H "X-Api-Key: asyou_abc123..."
```

### 7.2 常见工作流

#### 暴露服务

```bash
# 1. 注册节点（一次性操作）
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-node","host":"203.0.113.1","bind_port":7000}'

# 2. 创建隧道
curl -X POST http://localhost:8080/api/v1/proxies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-app","type":"tcp","local_port":3000,"node_id":1}'

# 3. 启动隧道
curl -X POST http://localhost:8080/api/v1/proxies/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"start"}'

# 4. 查看状态
curl http://localhost:8080/api/v1/proxies/1 \
  -H "Authorization: Bearer $TOKEN" | jq .status
# → "running"
```

#### 监控流量

```bash
# 获取统计数据
curl http://localhost:8080/api/v1/proxies/1/stats?limit=10 \
  -H "Authorization: Bearer $TOKEN" | jq
```

#### 监听实时事件

```bash
# 使用 curl（SSE）
curl -N http://localhost:8080/api/v1/events?token=$JWT

# 使用 web/desktop 中的 hooks（自动）
```

### 7.3 错误处理

始终检查响应体中的错误信息：
```json
{
  "error": "failed to start proxy: start frpc: exec: \"frpc\": executable file not found in $PATH",
  "code": "INTERNAL"
}
```

常见错误：

| HTTP 状态码 | 错误码 | 含义 |
|-------------|--------|------|
| 400 | `BAD_REQUEST` | 缺少或无效的参数 |
| 401 | `UNAUTHORIZED` | 缺少/过期/无效的 Token |
| 403 | `FORBIDDEN` | 资源不属于您 |
| 404 | `NOT_FOUND` | 资源未找到 |
| 500 | `INTERNAL` | 服务器错误（检查日志） |

---

## 8. SDK 快速入门

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

    // 一键暴露
    proxy, _ := client.CreateProxy("my-app", "tcp", 3000, 0)
    client.ProxyAction(proxy.ID, "start")
    fmt.Printf("Tunnel #%d: %s\n", proxy.ID, proxy.Status)

    // 列出隧道
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

# 一键暴露
proxy = client.expose(3000, name="my-app")
print(f"Tunnel #{proxy.id}: {proxy.status}")

# 列出隧道
for p in client.list_proxies():
    print(f"[{p.id}] {p.name}: {p.status}")
```

### 8.3 Node.js SDK

```typescript
import { Client } from 'asyou-sdk'

const client = new Client('http://localhost:8080')
await client.login('me@example.com', 'password')

// 一键暴露
const proxy = await client.expose(3000, 'my-app')
console.log(`Tunnel #${proxy.id}: ${proxy.status}`)

// 列出隧道
const proxies = await client.listProxies()
for (const p of proxies) {
    console.log(`[${p.id}] ${p.name}: ${p.status}`)
}
```

---

## 9. 节点管理

### 9.1 什么是节点？

**节点**是一个接受隧道连接的 frps 服务器。您需要至少注册一个节点才能使隧道工作。在多区域部署中，您可以在世界各地拥有多个节点。

### 9.2 注册节点

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

### 9.3 节点心跳

节点定期发送心跳报告健康状态。可以在 frps 机器上执行：

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

### 9.4 查找最佳节点

自动节点选择：

```bash
# 基于地理偏好
curl "http://localhost:8080/api/v1/nodes/best?region=ap-northeast&lat=35.68&lng=139.65"
```

响应中包含 `score` 字段，显示计算出的评分。

---

## 10. 隧道生命周期

### 10.1 状态流转

```
stopped ──► running ──► stopped
   ▲          │
   │          ├── (进程退出) ──► stopped
   │          └── (重载) ──► stop ──► start ──► running
   │
   └── (删除) ──► 已移除
```

### 10.2 启动

当您启动隧道时：

1. asyou 服务器在 `/tmp/asyou-proxy-{id}-*.ini` 生成 frpc 配置文件
2. 启动 `frpc -c <config>` 作为子进程
3. 成功时：隧道状态 → `"running"`
4. 失败时：错误信息记录在 `annotations` 字段中，格式为 JSON：
   ```json
   {"error": "start frpc: exec: \"frpc\": not found", "when": "start"}
   ```

### 10.3 停止

1. 向 frpc 进程发送 SIGKILL 信号
2. 状态 → `"stopped"`
3. 清理配置文件

### 10.4 重载

1. 停止当前 frpc 进程
2. 生成新的配置（拾取任何变更）
3. 启动新的 frpc 进程
4. 状态 → `"running"`

### 10.5 错误处理

如果启动/停止/重载失败，错误信息会以 JSON 格式存储：

```json
{
  "error": "frpc executable not found in $PATH",
  "when": "start"
}
```

通过 API 查看：
```bash
curl http://localhost:8080/api/v1/proxies/1 \
  -H "Authorization: Bearer $TOKEN" | jq .annotations
```

---

## 11. 流量监控

### 11.1 REST API

```bash
# 获取最近 60 条统计数据
curl "http://localhost:8080/api/v1/proxies/1/stats?limit=60" \
  -H "Authorization: Bearer $TOKEN" | jq
```

响应：
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

### 11.2 从 frps 节点上报统计数据

```bash
curl -X POST http://<asyou-server>:8080/api/v1/proxies/1/stats \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"bytes_in": 1024, "bytes_out": 512, "conn_count": 3}'
```

### 11.3 通过 SSE 实时获取

Web 仪表盘会自动通过 SSE 每 10 秒接收统计数据。您也可以直接连接：

```bash
curl -N "http://localhost:8080/api/v1/events?token=$JWT"
```

### 11.4 Prometheus 指标

```bash
curl http://localhost:8080/api/v1/metrics
```

返回示例：
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

## 12. TLS 证书

### 12.1 自动颁发（ACME）

```bash
POST /api/v1/certs/provision
Authorization: Bearer $TOKEN
Content-Type: application/json

{
    "proxy_id": 1,
    "domain": "app.example.com"
}
```

**要求：**
- 域名必须解析到您的 frps 服务器的公网 IP
- frps 服务器必须能从公网访问 80 端口（HTTP-01 验证）
- 响应中包含过期日期和颁发机构

### 12.2 管理证书

```bash
# 列出所有证书
curl http://localhost:8080/api/v1/certs \
  -H "Authorization: Bearer $TOKEN" | jq

# 获取证书详情
curl http://localhost:8080/api/v1/certs/1 \
  -H "Authorization: Bearer $TOKEN" | jq

# 删除证书
curl -X DELETE http://localhost:8080/api/v1/certs/1 \
  -H "Authorization: Bearer $TOKEN"
```

### 12.3 使用外部 ACME 客户端

对于生产环境，您可能更倾向于使用 certbot 或 acme.sh：

```bash
# 使用 certbot 获取证书
certbot certonly --standalone -d app.example.com

# 然后通过 API 上传（未来功能）
# 目前请手动存储证书文件并配置 frps
```

---

## 13. 多租户与角色

### 13.1 用户角色

| 角色 | 权限 |
|------|------|
| `user` | 管理自己的隧道、查看自己的审计日志、管理自己的 API 密钥 |
| `admin` | 拥有用户的所有权限 + 查看所有资源 + 管理所有租户 |

### 13.2 租户隔离

- 普通用户**只能看到自己**的隧道、证书和 API 密钥
- 管理员可以**看到所有**用户的资源
- 所有查询自动按 `user_id` 进行作用域限制
- 隧道名称在**同一用户下必须唯一**

### 13.3 管理员 API

管理员可以访问任何用户的资源：
```bash
# 管理员查看所有隧道
curl http://localhost:8080/api/v1/proxies \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

---

## 14. 全局调度

### 14.1 工作原理

调度器根据以下指标对每个活跃节点进行评分：

- **当前负载**：连接数与最大连接数的比率
- **延迟**：最近一次心跳的延迟
- **地理邻近度**：与首选位置的距离
- **区域匹配**：精确的区域偏好匹配
- **权重**：管理员配置的乘数

### 14.2 获取最佳节点

```bash
# 简单：仅获取最佳节点
curl http://localhost:8080/api/v1/nodes/best

# 带地理偏好（例如用户在东京）
curl "http://localhost:8080/api/v1/nodes/best?region=ap-northeast&lat=35.68&lng=139.65"

# 覆盖最大距离
curl "http://localhost:8080/api/v1/nodes/best?lat=35.68&lng=139.65&max_distance=2000"
```

### 14.3 节点权重

为性能更强的节点设置更高的权重：
```bash
curl -X PUT http://localhost:8080/api/v1/nodes/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"weight": 2.0}'
```

---

## 15. 实时事件

### 15.1 SSE 流

服务器通过 Server-Sent Events 在 `/api/v1/events` 推送事件。

**事件类型：**

| 类型 | 触发时机 | 数据 |
|------|----------|------|
| `connected` | 连接建立 | `{ user_id }` |
| `proxy_update` | 状态变更 | `{ id, status, annotations, timestamp }` |
| `stats_update` | 每 10 秒 | `[{ proxy_id, bytes_in, bytes_out, conn_count }]` |

### 15.2 编程方式连接

```javascript
// 浏览器 JavaScript
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

## 16. 故障排除

### 16.0 版本不匹配

**问题：** frpc 版本与 frps 版本不匹配，导致连接失败。

**检查：**
```bash
# 查看服务器推荐的版本
asyou version

# 查看本地安装的版本
asyou check

# 或手动检查
/tmp/frpc --version
# 从服务器获取期望版本
curl http://localhost:8080/api/v1/version
```

**解决方案：**
1. **从 GitHub Releases 下载匹配的 frpc**：
   ```bash
   # 与节点报告的 frps 版本匹配
   VER="0.69.1"  # 替换为 'asyou version' 显示的版本
   cd /tmp
   curl -sL "https://github.com/fatedier/frp/releases/download/v${VER}/frp_${VER}_linux_amd64.tar.gz" -o frp.tar.gz
   tar xzf frp.tar.gz
   cp "frp_${VER}_linux_amd64/frpc" /usr/local/bin/frpc
   rm -rf "frp_${VER}_linux_amd64" frp.tar.gz
   ```

2. **服务器自动检测** frpc 路径（`findFrpc()`）。它会检查：
   - `frpc`（PATH 环境变量）
   - `/usr/local/bin/frpc`
   - `/usr/bin/frpc`
   - `/tmp/frpc`
   - `./frpc`（当前目录）

3. **如果管理多台机器**，设置共享下载脚本：
   ```bash
   #!/bin/bash
   # 保存为 sync-frpc.sh
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

4. **自动版本同步**（通过心跳）：
   节点发送心跳时可以包含 `frp_version`：
   ```bash
   curl -X POST http://localhost:8080/api/v1/nodes/1/heartbeat \
     -H "X-Node-Token: <token>" \
     -H "Content-Type: application/json" \
     -d '{"frp_version": "0.69.1", "connections": 5}'
   ```
   服务器记录此版本并将其用作新 frpc 实例的推荐版本。

### 16.1 "frpc: executable file not found"

```
Error: failed to start proxy: start frpc: exec: "frpc": executable file not found in $PATH
```

**解决方案：**
1. 安装 frpc：`cp /tmp/frpc /usr/local/bin/frpc`
2. 或将 frpc 放在以下位置之一：`/tmp/frpc`、`./frpc`、`/usr/bin/frpc`
3. 服务器会通过 `findFrpc()` 自动检查这些位置

### 16.2 "connection refused"

```
Error: connect: connection refused
```

**检查：**
1. 您的本地服务是否在指定端口上运行？
2. `curl http://localhost:8802/` — 有响应吗？
3. frps 是否在运行？`ps aux | grep frps`

### 16.3 "port already in use"

```
Error: listen tcp :8080: bind: address already in use
```

**解决方案：**
```bash
# 查找占用端口的进程
fuser 8080/tcp

# 终止它
fuser -k 8080/tcp
```

### 16.4 隧道状态卡在 "stopped"

1. 查看隧道注释中的错误详情：
   ```bash
   curl http://localhost:8080/api/v1/proxies/1 -H "Bearer $TOKEN" | jq .annotations
   ```
2. 确认 frpc 二进制文件存在于已知位置
3. 检查服务器日志：`cat /tmp/asyou-server.log`

### 16.5 认证错误

```
{"code":"UNAUTHORIZED","error":"missing or invalid authorization header"}
```

**解决方案：**
1. Token 过期（24 小时）— 重新登录
2. 缺少 `Bearer ` 前缀 — 确保格式为 `Authorization: Bearer <token>`
3. API Key 错误 — 从仪表盘创建一个新的

### 16.6 SSE 连接丢失

SSE 会在 3 秒后自动重连。如果持续失败：
1. 检查 `/api/v1/events` 是否可以访问
2. 确保 Token 未过期
3. 如果使用自定义前端，检查 CORS 问题

---

## 17. 常见问题

**问：我可以在没有 frp 的情况下使用 asyou 吗？**  
答：不可以。asyou 是 frp 之上的管理层。您需要 frps 和 frpc 二进制文件。

**问：我需要公网 IP 吗？**  
答：只有 frps 服务器需要公网 IP。frpc 客户端（您的本地机器）可以在 NAT 后面。

**问：我可以创建多少个隧道？**  
答：仅受 frps 服务器容量限制（每个节点的 `max_connections`）。

**问：有用于实时更新的 WebSocket 吗？**  
答：asyou 使用 SSE（Server-Sent Events）而不是 WebSocket。SSE 更简单，使用标准 HTTP，可与任何现有基础设施配合使用。

**问：我可以使用 PostgreSQL 代替 SQLite 吗？**  
答：目前仅支持 SQLite。PostgreSQL 支持可以添加 — SQL 是标准的，查询也很简单。

**问：如何备份数据库？**  
答：在服务器没有写入时复制 `asyou.db`。服务器默认以 WAL 模式使用 SQLite。

**问：我可以运行多个 asyou 服务器吗？**  
答：目前不支持。设计假设单一管理服务器配合多个 frps 节点。

**问：如何更新 frp？**  
答：替换 `frps` 和 `frpc` 二进制文件为新版本。asyou 服务器会自动检测二进制文件路径。

**问：有速率限制吗？**  
答：目前没有。这计划在未来的版本中添加。

**问：我可以使用自定义域名吗？**  
答：可以。创建 HTTP 隧道时设置 `custom_domains`，并使用证书颁发接口获取 TLS 证书。
