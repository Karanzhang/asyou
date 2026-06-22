package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sdk "github.com/asyou/sdk-go"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		cmdVersion()
	case "check":
		cmdCheck()
	case "login":
		cmdLogin()
	case "expose":
		cmdExpose()
	case "list":
		cmdList()
	case "delete":
		cmdDelete()
	case "logout":
		cmdLogout()
	case "start":
		cmdStart()
	case "nodes":
		cmdNodes()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`asyou CLI — Tunnel management

Usage:
  asyou login <email> <password>          Login to server
  asyou logout                            Clear saved credentials
  asyou expose [--n <name>] [--node <id>] [--remote-port <port>] <local_port>   Create & start a tunnel (--node optional; auto-selects if omitted)
  asyou list                              List your tunnels
  asyou delete <id>                       Delete a tunnel
  asyou start <id>                        Start frpc for an existing tunnel
  asyou nodes                             List available nodes`)
}

func configPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "asyou", "cli-config.json")
}

func loadConfig() *sdk.Client {
	cfg := sdk.NewClient("http://localhost:8080")
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	var cred struct {
		ServerURL string `json:"server_url"`
		Token     string `json:"token"`
	}
	json.Unmarshal(data, &cred)
	if cred.ServerURL != "" {
		cfg = sdk.NewClient(cred.ServerURL)
	}
	if cred.Token != "" {
		cfg.SetToken(cred.Token)
	}
	return cfg
}

func saveConfig(c *sdk.Client) {
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.Marshal(map[string]string{
		"server_url": c.BaseURL,
		"token":      c.Token(),
	})
	os.WriteFile(path, data, 0600)
}

func cmdLogin() {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	server := fs.String("s", "http://localhost:8080", "Server URL")
	fs.Parse(os.Args[2:])
	args := fs.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: asyou login [--s <url>] <email> <password>")
		os.Exit(1)
	}
	client := sdk.NewClient(*server)
	if err := client.Login(args[0], args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "login failed: %v\n", err)
		os.Exit(1)
	}
	saveConfig(client)
	fmt.Println("Logged in as", args[0])
}

func cmdLogout() {
	os.Remove(configPath())
	fmt.Println("Logged out")
}

func cmdExpose() {
	// Manual flag parsing — Go's flag package stops at first non-flag arg,
	// so we parse ourselves to allow flags anywhere in the argument list.
	rawArgs := os.Args[2:]
	var tunnelName string
	nodeID := 0
	remotePort := 0
	var positional []string
	for i := 0; i < len(rawArgs); i++ {
		switch rawArgs[i] {
		case "-n":
			if i+1 < len(rawArgs) {
				tunnelName = rawArgs[i+1]
				i++
			} else {
				fmt.Fprintln(os.Stderr, "error: -n requires a value")
				os.Exit(1)
			}
		case "--node":
			if i+1 < len(rawArgs) {
				fmt.Sscanf(rawArgs[i+1], "%d", &nodeID)
				i++
			} else {
				fmt.Fprintln(os.Stderr, "error: --node requires a value")
				os.Exit(1)
			}
		case "--remote-port":
			if i+1 < len(rawArgs) {
				fmt.Sscanf(rawArgs[i+1], "%d", &remotePort)
				i++
			} else {
				fmt.Fprintln(os.Stderr, "error: --remote-port requires a value")
				os.Exit(1)
			}
		default:
			positional = append(positional, rawArgs[i])
		}
	}
	if len(positional) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: asyou expose [--n <name>] [--node <id>] [--remote-port <port>] <local_port>")
		os.Exit(1)
	}

	fmt.Printf("[config] tunnel_name=%q node_id=%d remote_port=%d positional=%v\n", tunnelName, nodeID, remotePort, positional)

	client := loadConfig()
	if client.Token() == "" {
		fmt.Fprintln(os.Stderr, "Not logged in. Run 'asyou login' first.")
		os.Exit(1)
	}

	if tunnelName == "" {
		tunnelName = fmt.Sprintf("cli-tunnel-%s", positional[0])
	}

	port := 0
	fmt.Sscanf(positional[0], "%d", &port)
	if port == 0 {
		fmt.Fprintf(os.Stderr, "invalid port: %s\n", positional[0])
		os.Exit(1)
	}

	// Auto-select node if not specified
	if nodeID == 0 {
		nodes, err := client.ListNodes()
		if err == nil {
			// Filter active nodes
			activeNodes := make([]sdk.Node, 0)
			for _, n := range nodes {
				activeNodes = append(activeNodes, n)
			}
			if len(activeNodes) == 1 {
				nodeID = int(activeNodes[0].ID)
				fmt.Printf("[asyou] auto-selected node #%d %q (only available node)\n", nodeID, activeNodes[0].Name)
			} else if len(activeNodes) > 1 {
				fmt.Fprintf(os.Stderr, "Multiple nodes available. Please specify one with --node:\n")
				for _, n := range activeNodes {
					fmt.Fprintf(os.Stderr, "  %d: %s (%s:%d)\n", n.ID, n.Name, n.Host, n.BindPort)
				}
				fmt.Fprintf(os.Stderr, "  Run 'asyou nodes' to see all nodes.\n")
				os.Exit(1)
			}
		} else {
			fmt.Printf("[asyou] warning: cannot list nodes: %v (will use server fallback)\n", err)
		}
	}

	fmt.Printf("[asyou] creating tunnel %q (local port %d, node %d)...\n", tunnelName, port, nodeID)
	proxy, err := client.CreateProxy(tunnelName, "tcp", port, nodeID, remotePort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[asyou] created proxy #%d, server returned remote_port=%v\n", proxy.ID, proxy.RemotePort)

	// If server didn't assign a remote port in the create response, fetch it
	if proxy.RemotePort == nil || *proxy.RemotePort == 0 {
		fmt.Printf("[asyou] remote port not assigned yet, re-fetching proxy #%d...\n", proxy.ID)
		if fetched, err := client.GetProxy(proxy.ID); err == nil && fetched.RemotePort != nil && *fetched.RemotePort > 0 {
			proxy = fetched
			fmt.Printf("[asyou] fetched proxy #%d, remote_port=%d\n", proxy.ID, *proxy.RemotePort)
		} else if err != nil {
			fmt.Printf("[asyou] warning: failed to re-fetch proxy: %v\n", err)
		} else {
			fmt.Printf("[asyou] warning: remote port still unassigned after re-fetch\n")
		}
	}

	// Get node info for frpc config
	var frpsHost string
	var frpsPort int
	if nodeID > 0 {
		fmt.Printf("[asyou] fetching node #%d info...\n", nodeID)
		node, err := client.GetNode(int64(nodeID))
		if err == nil {
			frpsHost = node.Host
			frpsPort = node.BindPort
			fmt.Printf("[asyou] node #%d: host=%s bind_port=%d\n", nodeID, frpsHost, frpsPort)
		} else {
			fmt.Fprintf(os.Stderr, "warning: cannot get node info: %v\n", err)
		}
	}
	if frpsHost == "" {
		// Fall back to the server URL host (API and frps are on same machine)
		frpsHost = client.BaseURL
		// Strip protocol prefix
		for _, p := range []string{"https://", "http://"} {
			frpsHost = strings.TrimPrefix(frpsHost, p)
		}
		// Strip port and path
		if idx := strings.Index(frpsHost, ":"); idx > 0 {
			frpsHost = frpsHost[:idx]
		}
		if idx := strings.Index(frpsHost, "/"); idx > 0 {
			frpsHost = frpsHost[:idx]
		}
		frpsPort = 7000
		fmt.Printf("[asyou] using fallback frps: %s:%d\n", frpsHost, frpsPort)
	}
	if frpsPort == 0 {
		frpsPort = 7000
	}

	// Generate frpc config and run locally
	cfgDir, _ := os.UserConfigDir()
	cfgPath := filepath.Join(cfgDir, "asyou", fmt.Sprintf("proxy-%d.ini", proxy.ID))
	os.MkdirAll(filepath.Dir(cfgPath), 0755)

	iniContent := fmt.Sprintf(`[common]
server_addr = %s
server_port = %d

[%s]
type = tcp
local_ip = 127.0.0.1
local_port = %d
`, frpsHost, frpsPort, tunnelName, port)
	// Add remote_port if specified by user or assigned by server
	if remotePort > 0 {
		iniContent += fmt.Sprintf("remote_port = %d\n", remotePort)
	} else if proxy.RemotePort != nil && *proxy.RemotePort > 0 {
		iniContent += fmt.Sprintf("remote_port = %d\n", *proxy.RemotePort)
	}
	fmt.Printf("[asyou] generated frpc config:\n%s\n", iniContent)
	if err := os.WriteFile(cfgPath, []byte(iniContent), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "write config failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tunnel #%d '%s' created.\n", proxy.ID, proxy.Name)
	runFrpc(cfgPath, "", proxy, frpsHost)
}

func cmdStart() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: asyou start <id>")
		os.Exit(1)
	}
	client := loadConfig()
	if client.Token() == "" {
		fmt.Fprintln(os.Stderr, "Not logged in. Run 'asyou login' first.")
		os.Exit(1)
	}
	var id int64
	fmt.Sscanf(os.Args[2], "%d", &id)
	if id == 0 {
		fmt.Fprintf(os.Stderr, "invalid id: %s\n", os.Args[2])
		os.Exit(1)
	}

	proxy, err := client.GetProxy(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get proxy failed: %v\n", err)
		os.Exit(1)
	}

	// Resolve frps host:port from node
	var frpsHost string
	var frpsPort int
	if proxy.NodeID != nil && *proxy.NodeID > 0 {
		node, err := client.GetNode(*proxy.NodeID)
		if err == nil {
			frpsHost = node.Host
			frpsPort = node.BindPort
			fmt.Printf("[asyou] node #%d: host=%s bind_port=%d\n", *proxy.NodeID, frpsHost, frpsPort)
		} else {
			fmt.Fprintf(os.Stderr, "warning: cannot get node info: %v\n", err)
		}
	}
	if frpsHost == "" {
		frpsHost = client.BaseURL
		for _, p := range []string{"https://", "http://"} {
			frpsHost = strings.TrimPrefix(frpsHost, p)
		}
		if idx := strings.Index(frpsHost, ":"); idx > 0 {
			frpsHost = frpsHost[:idx]
		}
		if idx := strings.Index(frpsHost, "/"); idx > 0 {
			frpsHost = frpsHost[:idx]
		}
		frpsPort = 7000
		fmt.Printf("[asyou] using fallback frps: %s:%d\n", frpsHost, frpsPort)
	}
	if frpsPort == 0 {
		frpsPort = 7000
	}

	// Generate frpc config
	cfgDir, _ := os.UserConfigDir()
	cfgPath := filepath.Join(cfgDir, "asyou", fmt.Sprintf("proxy-%d.ini", proxy.ID))
	os.MkdirAll(filepath.Dir(cfgPath), 0755)

	iniContent := fmt.Sprintf(`[common]
server_addr = %s
server_port = %d

[%s]
type = %s
local_ip = 127.0.0.1
local_port = %d
`, frpsHost, frpsPort, proxy.Name, proxy.Type, proxy.LocalPort)
	if proxy.RemotePort != nil && *proxy.RemotePort > 0 {
		iniContent += fmt.Sprintf("remote_port = %d\n", *proxy.RemotePort)
	}
	if err := os.WriteFile(cfgPath, []byte(iniContent), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "write config failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[asyou] generated config: %s\n", cfgPath)

	runFrpc(cfgPath, "", proxy, frpsHost)
}

func runFrpc(cfgPath, frpcPath string, proxy *sdk.Proxy, frpsHost string) {
	if frpcPath == "" {
		candidates := []string{
			"frpc",
			filepath.Join(filepath.Dir(os.Args[0]), "frpc"),
			filepath.Join(filepath.Dir(os.Args[0]), "frpc.exe"),
			"C:\\Windows\\System32\\frpc.exe",
			"/usr/local/bin/frpc",
			"/usr/bin/frpc",
		}
		for _, c := range candidates {
			if path, err := exec.LookPath(c); err == nil {
				frpcPath = path
				break
			}
			if _, err := os.Stat(c); err == nil {
				frpcPath = c
				break
			}
		}
	}
	if frpcPath == "" {
		fmt.Fprintln(os.Stderr, "Error: frpc not found.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Install frpc manually:")
		fmt.Fprintln(os.Stderr, "  Windows: Download from https://github.com/fatedier/frp/releases")
		fmt.Fprintln(os.Stderr, "           and place frpc.exe in C:\\Windows\\System32\\")
		fmt.Fprintln(os.Stderr, "  Linux:   sudo cp frpc /usr/local/bin/frpc")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "Or run frpc directly with the config at:\n  %s -c %s\n", "frpc", cfgPath)
		os.Exit(1)
	}
	cmd := exec.Command(frpcPath, "-c", cfgPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("Config: %s\n", cfgPath)
	fmt.Printf("frpc:  %s\n", frpcPath)
	if proxy.RemotePort != nil && *proxy.RemotePort > 0 {
		fmt.Printf("\n🚀 Access your service at:\n")
		fmt.Printf("   http://%s:%d\n", frpsHost, *proxy.RemotePort)
		fmt.Printf("   (Make sure firewall allows port %d)\n", *proxy.RemotePort)
	}
	fmt.Println("")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "frpc exited: %v\n", err)
		os.Exit(1)
	}
}

func cmdList() {
	client := loadConfig()
	if client.Token() == "" {
		fmt.Fprintln(os.Stderr, "Not logged in. Run 'asyou login' first.")
		os.Exit(1)
	}
	proxies, err := client.ListProxies()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list failed: %v\n", err)
		os.Exit(1)
	}
	if len(proxies) == 0 {
		fmt.Println("No tunnels.")
		return
	}
	fmt.Printf("%-4s %-20s %-8s %-8s %-8s %s\n", "ID", "Name", "Type", "LPort", "RPort", "Status")
	for _, p := range proxies {
		rp := "-"
		if p.RemotePort != nil && *p.RemotePort > 0 {
			rp = fmt.Sprintf("%d", *p.RemotePort)
		}
		fmt.Printf("%-4d %-20s %-8s %-8d %-8s %s\n", p.ID, p.Name, p.Type, p.LocalPort, rp, p.Status)
	}
}

func cmdDelete() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: asyou delete <id>")
		os.Exit(1)
	}
	client := loadConfig()
	if client.Token() == "" {
		fmt.Fprintln(os.Stderr, "Not logged in. Run 'asyou login' first.")
		os.Exit(1)
	}
	var id int64
	fmt.Sscanf(os.Args[2], "%d", &id)
	if id == 0 {
		fmt.Fprintf(os.Stderr, "invalid id: %s\n", os.Args[2])
		os.Exit(1)
	}

	// Fetch proxy info before deleting (for logging)
	if proxy, err := client.GetProxy(id); err == nil {
		fmt.Printf("[asyou] deleting tunnel #%d %q (status=%s)\n", id, proxy.Name, proxy.Status)
	} else {
		fmt.Printf("[asyou] deleting tunnel #%d\n", id)
	}

	if err := client.DeleteProxy(id); err != nil {
		fmt.Fprintf(os.Stderr, "delete failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Tunnel #%d deleted\n", id)

	// Clean up local config file if present
	cfgDir, _ := os.UserConfigDir()
	cfgPath := filepath.Join(cfgDir, "asyou", fmt.Sprintf("proxy-%d.ini", id))
	if _, err := os.Stat(cfgPath); err == nil {
		os.Remove(cfgPath)
		fmt.Printf("[asyou] cleaned up local config: %s\n", cfgPath)
	}
}

func cmdNodes() {
	client := loadConfig()
	if client.Token() == "" {
		fmt.Fprintln(os.Stderr, "Not logged in. Run 'asyou login' first.")
		os.Exit(1)
	}
	nodes, err := client.ListNodes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "list failed: %v\n", err)
		os.Exit(1)
	}
	if len(nodes) == 0 {
		fmt.Println("No nodes configured.")
		return
	}
	fmt.Printf("%-4s %-20s %-16s %s\n", "ID", "Name", "Host", "Port")
	for _, n := range nodes {
		fmt.Printf("%-4d %-20s %-16s %d\n", n.ID, n.Name, n.Host, n.BindPort)
	}
}

func cmdVersion() {
	client := loadConfig()
	// Try unauthenticated first
	resp, err := sdk.NewClient(client.BaseURL).GetVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get version: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("asyou CLI version:     0.1.0\n")
	fmt.Printf("Server version:         %s\n", resp["server_version"])
	fmt.Printf("Recommended frpc:       %s\n", resp["recommended_frpc_version"])
	fmt.Printf("Download URL:           %s\n", resp["frpc_download_url"])
	if nodes, ok := resp["nodes_by_version"].([]interface{}); ok {
		for _, nv := range nodes {
			if m, ok := nv.(map[string]interface{}); ok {
				fmt.Printf("  Node frp %-8s: %d node(s)\n", m["version"], m["count"])
			}
		}
	}
}

func cmdCheck() {
	client := loadConfig()
	resp, err := sdk.NewClient(client.BaseURL).GetVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "version check failed: %v\n", err)
		os.Exit(1)
	}
	expected, _ := resp["recommended_frpc_version"].(string)
	actual := getFrpcVersion()
	fmt.Printf("Expected frpc: %s\n", expected)
	fmt.Printf("Actual frpc:   %s\n", actual)
	if actual == "" {
		fmt.Println("⚠ frpc not found. Install it:")
		fmt.Println("  ", resp["frpc_download_url"])
		os.Exit(1)
	}
	if expected != "" && !strings.HasPrefix(actual, expected[:3]) {
		fmt.Println("✗ Version mismatch! Update frpc to match the server.")
		os.Exit(1)
	}
	fmt.Println("✓ Version OK")
}

func getFrpcVersion() string {
	// Check common paths
	for _, p := range []string{"frpc", "/usr/local/bin/frpc", "/usr/bin/frpc", "/tmp/frpc", "./frpc"} {
		cmd := exec.Command(p, "--version")
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return ""
}

