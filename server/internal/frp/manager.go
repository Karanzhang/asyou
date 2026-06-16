// Package frp provides a thin adapter around core/frpc for backward compatibility.
// New code should use github.com/asyou/core/frpc directly.
package frp

import (
	"fmt"
	"os/exec"

	corefrpc "github.com/asyou/core/frpc"
	"github.com/asyou/server/internal/model"
)

// Manager wraps core/frpc.Manager with the server's model types.
type Manager struct {
	inner *corefrpc.Manager
	// CmdBuilder overrides the frpc command. Used in tests.
	CmdBuilder func(cfgPath string) *exec.Cmd
}

// NewManager creates a new Manager wrapping core/frpc.
func NewManager() *Manager {
	m := &Manager{inner: corefrpc.NewManager()}
	// Use the default CmdBuilder from core (which finds frpc via findFrpc).
	// Tests may override m.CmdBuilder.
	return m
}

// Start launches frpc for the given proxy and node.
func (m *Manager) Start(proxy *model.Proxy, node *model.Node) error {
	if node == nil {
		return fmt.Errorf("missing node")
	}
	serverPort := node.BindPort
	if serverPort == 0 {
		serverPort = 7000
	}
	server := &corefrpc.ServerConfig{
		Host:  node.Host,
		Port:  serverPort,
		Token: node.AuthToken,
	}
	pc := corefrpc.MarshalProxyConfig(proxy.ID, proxy.Name, proxy.Type, proxy.LocalIP,
		proxy.LocalPort, proxy.RemotePort, proxy.Subdomain, proxy.CustomDomains)

	// propagate CmdBuilder
	if m.CmdBuilder != nil {
		m.inner.CmdBuilder = m.CmdBuilder
	}
	return m.inner.Start(pc, server)
}

// Stop terminates a proxy's frpc process.
func (m *Manager) Stop(proxyID int64) error {
	return m.inner.Stop(proxyID)
}

// IsRunning checks if a proxy's frpc process is active.
func (m *Manager) IsRunning(proxyID int64) bool {
	return m.inner.IsRunning(proxyID)
}

// LastError returns the last captured error for a proxy.
func (m *Manager) LastError(proxyID int64) string {
	return m.inner.LastError(proxyID)
}

// RunningProxies returns the set of proxy IDs currently managed.
func (m *Manager) RunningProxies() []int64 {
	return m.inner.RunningProxies()
}

