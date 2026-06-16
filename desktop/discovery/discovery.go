// Package discovery provides local port discovery for the desktop client.
package discovery

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

// PortInfo describes a listening port discovered on the local machine.
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp, udp
	Process  string `json:"process,omitempty"`
	PID      int    `json:"pid,omitempty"`
}

// DiscoverLocalPorts scans for listening TCP ports on localhost.
func DiscoverLocalPorts() ([]PortInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return discoverWindows()
	case "linux", "darwin":
		return discoverUnix()
	default:
		return discoverFallback()
	}
}

func discoverWindows() ([]PortInfo, error) {
	cmd := exec.Command("netstat", "-an")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseNetstat(string(out))
}

func discoverUnix() ([]PortInfo, error) {
	cmd := exec.Command("ss", "-tlnp")
	out, err := cmd.Output()
	if err != nil {
		// fallback to netstat
		cmd = exec.Command("netstat", "-tlnp")
		out, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}
	return parseSS(string(out))
}

func discoverFallback() ([]PortInfo, error) {
	// Simple port scan of common ports on localhost
	var ports []PortInfo
	for _, p := range []int{3000, 4000, 5000, 8000, 8080, 8443, 9000} {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			conn.Close()
			ports = append(ports, PortInfo{Port: p, Protocol: "tcp"})
		}
	}
	return ports, nil
}

func parseNetstat(output string) ([]PortInfo, error) {
	var ports []PortInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Look for LISTENING state
		state := ""
		for _, f := range fields {
			if strings.EqualFold(f, "listening") {
				state = "LISTEN"
				break
			}
		}
		if state != "LISTEN" {
			continue
		}
		// Extract address and port
		for _, f := range fields {
			if strings.Contains(f, ":") && strings.Count(f, ":") >= 1 {
				parts := strings.Split(f, ":")
				portStr := parts[len(parts)-1]
				port := 0
				fmt.Sscanf(portStr, "%d", &port)
				if port > 0 && port < 65536 {
					ports = append(ports, PortInfo{
						Port:     port,
						Protocol: "tcp",
						Process:  extractProcessName(fields),
					})
				}
				break
			}
		}
	}
	return ports, nil
}

func parseSS(output string) ([]PortInfo, error) {
	var ports []PortInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Netid") || strings.HasPrefix(line, "State") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Format: Netid State Recv-Q Send-Q Local Address:Port Peer Address:Port Process
		localAddr := fields[3]
		if !strings.Contains(localAddr, ":") {
			continue
		}
		parts := strings.Split(localAddr, ":")
		portStr := parts[len(parts)-1]
		port := 0
		fmt.Sscanf(portStr, "%d", &port)
		if port > 0 && port < 65536 {
			p := PortInfo{
				Port:     port,
				Protocol: fields[0],
			}
			if len(fields) > 5 {
				p.Process = strings.Trim(strings.Join(fields[5:], " "), "()")
			}
			ports = append(ports, p)
		}
	}
	return ports, nil
}

func extractProcessName(fields []string) string {
	for _, f := range fields {
		if strings.Contains(f, ".exe") {
			return f
		}
	}
	return ""
}
