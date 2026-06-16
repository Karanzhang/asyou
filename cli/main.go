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
  asyou expose <local_port> [name]        Create & start a tunnel
  asyou list                              List your tunnels
  asyou delete <id>                       Delete a tunnel
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
	fs := flag.NewFlagSet("expose", flag.ExitOnError)
	name := fs.String("n", "", "Tunnel name (default: auto)")
	nodeID := fs.Int("node", 0, "Node ID")
	fs.Parse(os.Args[2:])
	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: asyou expose [--n <name>] [--node <id>] <local_port>")
		os.Exit(1)
	}
	client := loadConfig()
	if client.Token() == "" {
		fmt.Fprintln(os.Stderr, "Not logged in. Run 'asyou login' first.")
		os.Exit(1)
	}

	tunnelName := *name
	if tunnelName == "" {
		tunnelName = fmt.Sprintf("cli-tunnel-%s", args[0])
	}

	port := 0
	fmt.Sscanf(args[0], "%d", &port)
	if port == 0 {
		fmt.Fprintf(os.Stderr, "invalid port: %s\n", args[0])
		os.Exit(1)
	}

	proxy, err := client.CreateProxy(tunnelName, "tcp", port, *nodeID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create failed: %v\n", err)
		os.Exit(1)
	}

	if err := client.ProxyAction(proxy.ID, "start"); err != nil {
		fmt.Fprintf(os.Stderr, "start failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tunnel #%d '%s' created and started on port %d\n", proxy.ID, proxy.Name, port)
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
	fmt.Printf("%-4s %-20s %-8s %-6s %s\n", "ID", "Name", "Type", "Port", "Status")
	for _, p := range proxies {
		fmt.Printf("%-4d %-20s %-8s %-6d %s\n", p.ID, p.Name, p.Type, p.LocalPort, p.Status)
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
	if err := client.DeleteProxy(id); err != nil {
		fmt.Fprintf(os.Stderr, "delete failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Tunnel #%d deleted\n", id)
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

