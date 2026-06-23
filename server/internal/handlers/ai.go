package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode"
)

type knowledgeEntry struct {
	Keywords []string
	Answer   string
}

var knowledgeBase = []knowledgeEntry{
	{
		Keywords: []string{"quick start", "快速开始", "start", "begin"},
		Answer: "**Quick Start:**\n" +
			"1. Go to **Proxies** and click **New Tunnel**\n" +
			"2. Fill in a name and local port (e.g. 3000 for a dev server)\n" +
			"3. Select TCP or HTTP type, then create\n" +
			"4. Open the tunnel detail page, download frpc for your OS\n" +
			"5. Run the provided frpc command to start the tunnel\n" +
			"6. Access your service via host:remote_port (TCP) or the subdomain URL (HTTP)",
	},
	{
		Keywords: []string{"tunnel", "proxy", "create", "new"},
		Answer: "**Creating a Tunnel:**\n" +
			"- Go to **Proxies** -> **New Tunnel**\n" +
			"- **Name**: Any descriptive name (unique to your account)\n" +
			"- **Type**: TCP for most services (SSH, RDP, databases), HTTP/HTTPS for web apps\n" +
			"- **Local Port**: The port your service runs on locally\n" +
			"- **Remote Port**: The port on the frps server (leave empty for auto-assignment)\n" +
			"- **Subdomain**: Only for HTTP/HTTPS -- requires frps subdomain_host and DNS wildcard\n" +
			"- **Node**: The frps server to use (auto-selected if only one node)",
	},
	{
		Keywords: []string{"frpc", "client", "run", "start", "本地", "local"},
		Answer: "**Running frpc Locally:**\n" +
			"After creating a tunnel, open its detail page. You'll find:\n" +
			"1. **Download frpc**: Links for Windows/Linux/macOS/ARM\n" +
			"2. **frpc command**: A ready-to-run command\n" +
			"3. **Config preview**: The generated frpc.ini - click Copy or Download\n" +
			"4. **Run script**: One-click script that auto-downloads frpc, writes the config, and starts\n\n" +
			"The tunnel status in the dashboard shows **running** only when frpc is actively connected to frps.",
	},
	{
		Keywords: []string{"type", "类型", "tcp", "http", "https", "udp"},
		Answer: "**Proxy Types:**\n\n" +
			"- **TCP**: SSH, RDP, databases, any TCP service -- access via host:remote_port\n" +
			"- **HTTP**: Web apps, APIs, dev servers -- access via http://subdomain.host\n" +
			"- **HTTPS**: Web apps with TLS -- access via https://subdomain.host\n" +
			"- **UDP**: Game servers, DNS, streaming -- access via host:remote_port (UDP)\n\n" +
			"For HTTP/HTTPS, frps must have vhost_http_port / vhost_https_port configured.",
	},
	{
		Keywords: []string{"subdomain", "域名"},
		Answer: "**Subdomain (HTTP/HTTPS only):**\n" +
			"Subdomain gives you a friendly URL instead of a port number.\n\n" +
			"**Prerequisites:**\n" +
			"- frps must have subdomain_host configured (e.g. tunnel.example.com)\n" +
			"- DNS wildcard record *.tunnel.example.com -> your frps server IP\n" +
			"- frps must enable vhost_http_port and/or vhost_https_port\n\n" +
			"Example: Create HTTP tunnel with subdomain myapp -> visit http://myapp.tunnel.example.com\n\n" +
			"Subdomains are unique per subdomain_host -- you cannot use the same subdomain on two tunnels that share the same host.",
	},
	{
		Keywords: []string{"port", "端口", "remote", "local port", "remote port"},
		Answer: "**Local Port vs Remote Port:**\n" +
			"- **Local Port**: The port your service runs on your machine (e.g. 3000 for Node.js, 8080 for Java)\n" +
			"- **Remote Port**: The port on the frps server that forwards traffic to your local service\n" +
			"  - Leave empty for **auto-assignment** from the node's port range\n" +
			"  - Or specify one manually (must be within the node's allowed range)\n\n" +
			"Example: Local port 3000 -> remote port 31001 -> access at frps-host:31001",
	},
	{
		Keywords: []string{"node", "节点", "frps", "server", "服务器"},
		Answer: "**Nodes (frps Servers):**\n" +
			"Nodes are frps instances that accept tunnel connections.\n\n" +
			"- Register a node with its host, bind_port, and dashboard credentials\n" +
			"- Each node has a **port range** (e.g. 31000-31499) for tunnel ports\n" +
			"- Node status is checked live via the frps admin API (dashboard)\n" +
			"- You can set subdomain_host per node for HTTP/HTTPS tunnels\n" +
			"- Configure geo fields (region, city, lat/lng) for scheduler\n" +
			"- **Weight** controls how many tunnels the scheduler assigns (higher = more)\n\n" +
			"To add a node: go to **Nodes** -> **Add Node**.",
	},
	{
		Keywords: []string{"scheduler", "调度", "select", "auto", "best"},
		Answer: "**Node Scheduler:**\n" +
			"When creating a tunnel without specifying a node, the scheduler auto-selects the best one.\n\n" +
			"Factors considered:\n" +
			"- **Weight**: Higher weight nodes get more tunnels\n" +
			"- **Max Connections**: Capacity limit\n" +
			"- **Region/City**: (future) geo-proximity\n" +
			"- **Active status**: Only online nodes are considered\n\n" +
			"You can also manually choose a node when creating a tunnel.",
	},
	{
		Keywords: []string{"api key", "apikey", "api_key", "token", "密钥"},
		Answer: "**API Keys:**\n" +
			"API keys allow programmatic access without JWT login.\n\n" +
			"1. Go to **API Keys** and click **Create Key**\n" +
			"2. Give it a label and optionally set an expiration\n" +
			"3. **Copy the key immediately** -- it's shown only once!\n\n" +
			"Usage:\n" +
			"curl -H \"X-Api-Key: your-api-key\" https://your-server.com/api/v1/proxies\n\n" +
			"To revoke: click the delete button on the API Keys page.",
	},
	{
		Keywords: []string{"audit", "日志", "log", "history", "历史"},
		Answer: "**Audit Logs:**\n" +
			"All actions are recorded in the audit log:\n" +
			"- Create/update/delete tunnels\n" +
			"- Node management\n" +
			"- User actions\n\n" +
			"Each entry shows: **Time**, **Action**, **Resource**, **Actor**, **IP Address**, and **Details**.\n\n" +
			"Go to **Audit Logs** to view the history.",
	},
	{
		Keywords: []string{"status", "stopped", "running", "状态", "离线", "online", "offline"},
		Answer: "**Tunnel Status:**\n" +
			"- **running**: frpc is actively connected to frps -- your service is accessible\n" +
			"- **stopped**: frpc is not running on the client machine\n\n" +
			"If your tunnel shows \"stopped\" but you have frpc running:\n" +
			"1. Check the frpc command and config are correct\n" +
			"2. Verify the frps server is reachable (host + bind_port)\n" +
			"3. Check firewall rules on both sides\n\n" +
			"**Node Status:**\n" +
			"- **Online**: frps dashboard API is reachable\n" +
			"- **Offline**: Cannot connect -- check if frps is running and dashboard_port is correct",
	},
	{
		Keywords: []string{"frp", "version", "版本", "update", "更新"},
		Answer: "**frp Version Management:**\n" +
			"- The server stores the frp version detected from each node's frps dashboard\n" +
			"- The CLI **asyou check** command verifies your local frpc matches the recommended version\n" +
			"- To update frp: download the latest release from GitHub and replace the frps/frpc binaries\n\n" +
			"Check frpc version:\n" +
			"  asyou check\n\n" +
			"Expected output:\n" +
			"  Expected frpc: 0.69.1\n" +
			"  Actual frpc:   0.69.1\n" +
			"  OK Version OK",
	},
	{
		Keywords: []string{"trouble", "error", "错误", "fail", "失败", "cannot", "connect", "refused", "拒绝"},
		Answer: "**Common Issues:**\n\n" +
			"**Connection refused**\n" +
			"- Check frps is running: systemctl status frps\n" +
			"- Check firewall: sudo ufw status\n" +
			"- Verify node's host and bind_port are correct\n" +
			"- If using local frpc, check server_addr and server_port in the config\n\n" +
			"**Subdomain not working**\n" +
			"- Ensure frps has subdomain_host configured\n" +
			"- Ensure vhost_http_port / vhost_https_port is set\n" +
			"- DNS wildcard record must exist\n" +
			"- Tunnel type must be http or https\n\n" +
			"**Tunnel shows stopped**\n" +
			"- frpc is not running on your machine\n" +
			"- Start it manually or use the run script from the tunnel detail page\n\n" +
			"**already in use / already exists**\n" +
			"- The tunnel name is unique per user -- choose a different name\n" +
			"- Subdomain must be unique across nodes sharing the same subdomain_host",
	},
	{
		Keywords: []string{"cli", "command line", "命令行", "asyou"},
		Answer: "**CLI Usage:**\n\n" +
			"Login:\n" +
			"  asyou login --s https://your-server.com admin@example.com\n\n" +
			"Expose a TCP service:\n" +
			"  asyou expose 3000 -n my-app\n\n" +
			"Expose with HTTP + subdomain:\n" +
			"  asyou expose 8080 --type http --subdomain myapp -n my-web\n\n" +
			"List tunnels:\n" +
			"  asyou list\n\n" +
			"Delete a tunnel:\n" +
			"  asyou delete 1\n\n" +
			"Start frpc for an existing tunnel:\n" +
			"  asyou start 1\n\n" +
			"List nodes:\n" +
			"  asyou nodes\n\n" +
			"Check frpc version:\n" +
			"  asyou check",
	},
	{
		Keywords: []string{"config", "配置", "ini", "toml", "frpc.ini", "frps.toml"},
		Answer: "**Configuration Files:**\n" +
			"The frpc config is auto-generated for each tunnel. Get it from the tunnel detail page.\n\n" +
			"**frpc.ini** (auto-generated):\n" +
			"  [common]\n" +
			"  server_addr = your-frps-host\n" +
			"  server_port = 7000\n\n" +
			"  [proxy_1]\n" +
			"  type = tcp\n" +
			"  local_ip = 127.0.0.1\n" +
			"  local_port = 3000\n" +
			"  remote_port = 31001\n\n" +
			"**frps configuration** (on the server):\n" +
			"  [common]\n" +
			"  bind_port = 7000\n" +
			"  allow_ports = 31000-31499\n" +
			"  dashboard_port = 7500\n" +
			"  dashboard_user = admin\n" +
			"  dashboard_pwd = your-password\n" +
			"  # subdomain_host = tunnel.example.com\n" +
			"  # vhost_http_port = 80",
	},
	{
		Keywords: []string{"script", "run script", "bat", "sh", "下载", "download", "windows", "linux", "macos"},
		Answer: "**Download & Run Scripts:**\n" +
			"The tunnel detail page provides:\n" +
			"- **frp download links**: Windows (amd64), Linux (amd64), macOS (amd64), Linux (arm64)\n" +
			"- **frpc command**: Ready to copy-paste\n" +
			"- **Config download**: frpc.ini with Copy or Download button\n" +
			"- **Run scripts**: One-click scripts that:\n" +
			"  - Auto-detect your OS\n" +
			"  - Download the matching frpc binary\n" +
			"  - Write the config inline (Base64 encoded)\n" +
			"  - Start frpc\n\n" +
			"Windows users get a .bat file, Linux/macOS get a .sh file.",
	},
	{
		Keywords: []string{"monitor", "traffic", "流量", "metrics", "统计", "stats"},
		Answer: "**Traffic Monitoring:**\n" +
			"The tunnel detail page shows a traffic chart with:\n" +
			"- **Inbound traffic** (bytes received)\n" +
			"- **Outbound traffic** (bytes sent)\n" +
			"- Data is polled periodically from the frps admin API\n\n" +
			"The **Nodes** page shows aggregated stats:\n" +
			"- Total clients connected\n" +
			"- Total proxies\n" +
			"- Current connections\n" +
			"- Traffic in/out per node",
	},
	{
		Keywords: []string{"auth", "login", "password", "密码", "登录", "register", "注册"},
		Answer: "**Authentication:**\n" +
			"- **JWT tokens**: Issued on login, valid for 24 hours\n" +
			"- **API Keys**: Long-lived keys for programmatic access\n" +
			"- **First time?** Register via the API:\n" +
			"  curl -X POST http://localhost:8080/api/v1/auth/register\n" +
			"    -H \"Content-Type: application/json\"\n" +
			"    -d '{\"email\":\"admin@example.com\",\"password\":\"your-password\",\"display_name\":\"Admin\"}'\n" +
			"- **Security**: Change the default JWT secret in production\n" +
			"- API Keys with no expiration never expire -- revoke them manually when needed",
	},
}

type AiQueryRequest struct {
	Message string `json:"message"`
}
type AiQueryResponse struct {
	Answer string `json:"answer"`
}

func (s *Server) AiQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}
	var req AiQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		writeJSONError(w, "message required", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	answer := findBestAnswer(req.Message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AiQueryResponse{Answer: answer})
}

func findBestAnswer(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	words := tokenize(query)
	if len(words) == 0 {
		return fallbackAnswer()
	}
	type scored struct {
		entry knowledgeEntry
		score int
	}
	var candidates []scored
	for _, entry := range knowledgeBase {
		score := 0
		for _, keyword := range entry.Keywords {
			kw := strings.ToLower(keyword)
			if strings.Contains(query, kw) {
				score += 10
			}
			for _, w := range words {
				if strings.Contains(kw, w) || strings.Contains(w, kw) {
					score++
				}
			}
		}
		if score > 0 {
			candidates = append(candidates, scored{entry, score})
		}
	}
	if len(candidates) == 0 {
		return fallbackAnswer()
	}
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best.entry.Answer
}

func tokenize(s string) []string {
	var words []string
	var cur strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			cur.WriteRune(r)
		} else {
			if cur.Len() > 0 {
				words = append(words, cur.String())
				cur.Reset()
			}
		}
	}
	if cur.Len() > 0 {
		words = append(words, cur.String())
	}
	return words
}

func fallbackAnswer() string {
	return "I couldn't find a specific answer. Try topics like:\n\n" +
		"**Getting Started**\n" +
		"- Creating and managing tunnels\n" +
		"- Running frpc locally\n" +
		"- Proxy types (TCP, HTTP, HTTPS, UDP)\n\n" +
		"**Configuration**\n" +
		"- Subdomain setup\n" +
		"- Port assignment\n" +
		"- Node management\n\n" +
		"**Troubleshooting**\n" +
		"- Connection issues\n" +
		"- Status problems\n\n" +
		"Or check the **Docs** page for the full user guide."
}
