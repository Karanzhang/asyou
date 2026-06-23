# asyou CLI

Command-line tool for tunnel management.

## Install

```bash
cd cli && go build -o /usr/local/bin/asyou .
```

## Usage

```bash
# Login to server (default: http://localhost:8080)
asyou login admin@example.com mypassword
# → Logged in as admin@example.com

# List tunnels
asyou list
# → ID   Name       Type   Port   Status
# → 1    my-tunnel  tcp    3000   running

# Expose a local port (creates + starts)
asyou expose 3000 --n my-app
# → Tunnel #2 'my-app' created and started on port 3000

# Expose on a specific node
asyou expose 3000 --n my-app --node 1

# Expose an HTTP tunnel with subdomain (requires frps subdomain_host)
asyou expose 8080 --type http --subdomain myapp -n my-web

# Expose a tunnel on a specific node with custom remote port
asyou expose 3000 -n my-app --node 1 --remote-port 31000

# Delete a tunnel
asyou delete 2
# → Tunnel #2 deleted

# List nodes
asyou nodes
# → ID   Name   Host             Port
# → 1    demo   127.0.0.1        7000

# Logout
asyou logout
```

## Commands

| Command | Description |
|---|---|
| `login [--s <url>] <email> <password>` | Login and save credentials |
| `logout` | Clear saved session |
| `expose [--n <name>] [--type <tcp\|http\|https\|udp>] [--subdomain <name>] [--node <id>] [--remote-port <port>] <port>` | Create + start a tunnel |
| `list` | List all tunnels |
| `delete <id>` | Delete a tunnel |
| `nodes` | List frps nodes |
