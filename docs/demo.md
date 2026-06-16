# asyou 本地 Demo — frps + frpc 隧道暴露

## 环境准备

确保以下组件已就绪：

| 组件 | 路径 | 版本 |
|---|---|---|
| `frps` (server) | `/tmp/frps` | v0.69.1 |
| `frpc` (client) | `/tmp/frpc` | v0.69.1 |
| `asyou-server` | `/tmp/asyou-server` | latest |
| `asyou` CLI | `/tmp/asyou` | latest |

安装命令：

```bash
# 下载 frp 二进制
cd /tmp
curl -sL "https://github.com/fatedier/frp/releases/download/v0.69.1/frp_0.69.1_linux_amd64.tar.gz" -o frp.tar.gz
tar xzf frp.tar.gz
cp frp_0.69.1_linux_amd64/frps /tmp/frps
cp frp_0.69.1_linux_amd64/frpc /tmp/frpc
chmod +x /tmp/frps /tmp/frpc
rm -rf frp_0.69.1_linux_amd64 frp.tar.gz

# 编译 asyou 服务端
cd /mnt/d/project/asyou/server && go build -o /tmp/asyou-server ./cmd/server

# 编译 CLI
cd /mnt/d/project/asyou/cli && go build -o /tmp/asyou .
```

---

## 架构

```
┌───────────────────┐     frpc -c tunnel.ini     ┌───────────────────┐
│  本地 HTTP 服务   │◄──────────────────────────►│   frps 服务器     │
│   (port 9999)     │                            │   (port 7000)     │
└─────────┬─────────┘                            └─────────┬─────────┘
          │                                                │
          │  1. 注册节点                                     │
          │  2. 创建隧道                                     │
          │  3. 启动 frpc                                    │
          ▼                                                 ▼
   ┌───────────────────┐                          ┌───────────────────┐
   │  asyou 管理服务    │                          │  浏览器 / curl    │
   │   (port 8080)      │                          │  (公网访问隧道)   │
   └───────────────────┘                          └───────────────────┘
```

---

## 完整演示流程

### 步骤 1：启动 frps

```bash
/tmp/frps --log_file /tmp/frps.log --log_level info
```

预期输出：
```
frps tcp listen on 0.0.0.0:7000
frps started successfully
```

### 步骤 2：启动 asyou 管理服务

```bash
rm -f /mnt/d/project/asyou/asyou.db  # 清理旧数据库
cd /mnt/d/project/asyou/server && /tmp/asyou-server
```

预期输出：
```
migrations dir: /mnt/d/project/asyou/migrations
starting server :8080
```

### 步骤 3：启动本地 HTTP 测试服务

```bash
cd /tmp && python3 -m http.server 9999
```

验证：
```bash
curl -s http://localhost:9999/ | head -3
# → <!DOCTYPE HTML> ...
```

### 步骤 4：注册用户

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@test.com","password":"demo123","display_name":"Demo User"}'
```

预期输出：
```json
{"id":1,"email":"demo@test.com","display_name":"Demo User","role":"user"}
```

### 步骤 5：登录获取 Token

```bash
LOGIN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@test.com","password":"demo123"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
echo "TOKEN=$TOKEN"
```

### 步骤 6：注册 frps 节点

```bash
curl -s -X POST http://localhost:8080/api/v1/nodes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"local-frps",
    "host":"127.0.0.1",
    "bind_port":7000,
    "region":"local",
    "country":"CN",
    "city":"Local"
  }'
```

验证：
```bash
curl -s http://localhost:8080/api/v1/nodes -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

### 步骤 7：创建隧道

```bash
curl -s -X POST http://localhost:8080/api/v1/proxies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"demo-tunnel","type":"tcp","local_port":9999,"node_id":1}'
```

### 步骤 8：启动隧道

```bash
curl -s -X POST http://localhost:8080/api/v1/proxies/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"start"}'
```

> 此操作会触发 asyou 服务端启动 `frpc -c <临时配置文件>`，连接到步骤 1 的 frps。

### 步骤 9：查询隧道状态

```bash
curl -s http://localhost:8080/api/v1/proxies/1 \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

预期输出（`status: "running"`）：
```json
{
    "id": 1,
    "name": "demo-tunnel",
    "type": "tcp",
    "local_ip": "127.0.0.1",
    "local_port": 9999,
    "status": "running",
    "annotations": ""
}
```

### 步骤 10：通过 frps 验证隧道可达

```bash
# 通过 frps（端口 7000）访问本地服务
curl -s http://127.0.0.1:<remote_port>/
# remote_port 由 frps 分配，可从 proxy 详情中获取
```

### 步骤 11：停止隧道

```bash
curl -s -X POST http://localhost:8080/api/v1/proxies/1/action \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"stop"}'
```

### 步骤 12：使用 CLI 查看

```bash
/tmp/asyou login demo@test.com demo123
/tmp/asyou list
```

预期输出：
```
ID   Name                 Type     Port   Status
1    demo-tunnel          tcp      9999   stopped
```

---

## 一键 Demo 脚本

```bash
#!/bin/bash
# save as demo.sh && bash demo.sh

set -e
BASE="http://localhost:8080"

# 1. Register
curl -s -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@test.com","password":"demo123"}' || true

# 2. Login
LOGIN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"demo@test.com","password":"demo123"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

# 3. Create Node
curl -s -X POST "$BASE/api/v1/nodes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"local-frps","host":"127.0.0.1","bind_port":7000}'

# 4. Create Proxy
curl -s -X POST "$BASE/api/v1/proxies" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"demo-tunnel","type":"tcp","local_port":9999,"node_id":1}'

# 5. Start
curl -s -X POST "$BASE/api/v1/proxies/1/action" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"start"}'

# 6. Check status
sleep 2
curl -s "$BASE/api/v1/proxies/1" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

---

## 常见问题

| 问题 | 原因 | 解决 |
|---|---|---|
| `frpc: executable file not found` | frpc 不在 PATH | `findFrpc()` 会检查 `/tmp/frpc`，确认文件存在 |
| `port already in use` | 端口被占用 | `fuser -k 7000/tcp` 或 `fuser -k 8080/tcp` |
| `connection refused` | 服务未启动 | 检查各组件日志 |
| 隧道状态卡在 `stopped` | frpc 进程异常退出 | 查看 `annotations` 字段中的错误信息 |
