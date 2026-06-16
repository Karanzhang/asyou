package frpc

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Manager manages frpc process lifecycles for multiple proxies.
type Manager struct {
	mu         sync.Mutex
	processes  map[int64]*exec.Cmd
	lastErr    map[int64]string
	configPath map[int64]string
	// CmdBuilder builds the command for a given config path. Override in tests.
	CmdBuilder func(cfgPath string) *exec.Cmd
}

// NewManager creates a new Manager with default CmdBuilder.
// It searches for frpc in common locations.
func NewManager() *Manager {
	frpcPath := findFrpc()
	return &Manager{
		processes:  make(map[int64]*exec.Cmd),
		lastErr:    make(map[int64]string),
		configPath: make(map[int64]string),
		CmdBuilder: func(cfgPath string) *exec.Cmd {
			return exec.Command(frpcPath, "-c", cfgPath)
		},
	}
}

// findFrpc searches for frpc in common locations.
func findFrpc() string {
	candidates := []string{
		"frpc",
		"/usr/local/bin/frpc",
		"/usr/bin/frpc",
		"/tmp/frpc",
		"./frpc",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// fallback — let exec.Cmd report the error
	return "frpc"
}

// Start launches an frpc process for the given proxy and server config.
func (m *Manager) Start(proxy *ProxyConfig, server *ServerConfig) error {
	m.mu.Lock()
	if _, running := m.processes[proxy.ID]; running {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	if server == nil || server.Host == "" {
		return fmt.Errorf("missing or invalid server config")
	}
	if proxy.LocalPort == 0 {
		return fmt.Errorf("proxy local_port required")
	}

	cfg := BuildINI(proxy, server)
	cfgPath, err := writeTempConfig(proxy.ID, cfg)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	cmd := m.CmdBuilder(cfgPath)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		os.Remove(cfgPath)
		return fmt.Errorf("start frpc: %w", err)
	}

	m.mu.Lock()
	m.processes[proxy.ID] = cmd
	m.configPath[proxy.ID] = cfgPath
	m.mu.Unlock()

	go func() {
		err := cmd.Wait()
		if err != nil {
			m.mu.Lock()
			m.lastErr[proxy.ID] = stderr.String()
			m.mu.Unlock()
		}
		m.mu.Lock()
		delete(m.processes, proxy.ID)
		delete(m.configPath, proxy.ID)
		m.mu.Unlock()
		os.Remove(cfgPath)
	}()

	return nil
}

// Stop terminates an frpc process by proxy ID.
func (m *Manager) Stop(proxyID int64) error {
	m.mu.Lock()
	cmd, ok := m.processes[proxyID]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	delete(m.processes, proxyID)
	delete(m.configPath, proxyID)
	m.mu.Unlock()

	if cmd.Process != nil {
		if err := cmd.Process.Kill(); err != nil {
			m.mu.Lock()
			m.lastErr[proxyID] = err.Error()
			m.mu.Unlock()
			return err
		}
	}
	return nil
}

// IsRunning checks if a proxy's frpc process is active.
func (m *Manager) IsRunning(proxyID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.processes[proxyID]
	return ok
}

// LastError returns the last captured error for a proxy.
func (m *Manager) LastError(proxyID int64) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastErr[proxyID]
}

// RunningProxies returns the set of proxy IDs currently managed.
func (m *Manager) RunningProxies() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]int64, 0, len(m.processes))
	for id := range m.processes {
		ids = append(ids, id)
	}
	return ids
}

func writeTempConfig(proxyID int64, content string) (string, error) {
	dir := os.TempDir()
	file, err := os.CreateTemp(dir, fmt.Sprintf("asyou-proxy-%d-*.ini", proxyID))
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return "", err
	}
	return file.Name(), nil
}

// FrpcVersion returns the version of the frpc binary, or empty string if unknown.
func FrpcVersion() string {
	path := findFrpc()
	cmd := exec.Command(path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CheckVersionCompatibility checks if frpc is compatible with a required version.
// Returns true if frpc version matches the required major.minor.
func CheckVersionCompatibility(required string) bool {
	v := FrpcVersion()
	if v == "" || required == "" {
		return true // can't check, assume compatible
	}
	// Simple prefix check: "0.69" matches "0.69.1"
	return strings.HasPrefix(v, required) || strings.HasPrefix(required, v)
}
