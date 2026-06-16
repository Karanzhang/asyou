package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/asyou/desktop/discovery"
)

// App is the main application struct exposed to the frontend via Wails bindings.
type App struct {
	ctx    context.Context
	config *Config
}

// NewApp creates a new App instance.
func NewApp() *App {
	return &App{
		config: LoadConfig(),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("asyou desktop started")
}

func (a *App) shutdown(ctx context.Context) {
	log.Println("asyou desktop shutting down")
}

// Shutdown cleans up resources.
func (a *App) Shutdown() {}

// AppBindings holds the methods callable from the frontend via wails runtime.
// These are defined on a separate struct so they can be bound cleanly.

// GetServerURL returns the configured server URL.
func (a *App) GetServerURL() string {
	return a.config.ServerURL
}

// SetServerURL updates the server URL.
func (a *App) SetServerURL(url string) error {
	a.config.ServerURL = url
	return a.config.Save()
}

// Login authenticates with the server.
func (a *App) Login(email, password string) error {
	client := NewAPIClient(a.config.ServerURL)
	if err := client.Login(email, password); err != nil {
		return err
	}
	a.config.Email = email
	a.config.Token = client.token
	return a.config.Save()
}

// Register creates an account and logs in.
func (a *App) Register(email, password, displayName string) error {
	client := NewAPIClient(a.config.ServerURL)
	if err := client.Register(email, password, displayName); err != nil {
		return err
	}
	a.config.Email = email
	a.config.Token = client.token
	return a.config.Save()
}

// IsLoggedIn checks if the user has stored credentials.
func (a *App) IsLoggedIn() bool {
	return a.config.Token != ""
}

// Logout clears stored credentials.
func (a *App) Logout() error {
	a.config.Token = ""
	a.config.Email = ""
	return a.config.Save()
}

// GetCurrentUser returns user info from the server.
func (a *App) GetCurrentUser() (map[string]interface{}, error) {
	client := a.authenticatedClient()
	return client.GetUser()
}

// ListNodes returns all nodes from the server.
func (a *App) ListNodes() ([]map[string]interface{}, error) {
	client := a.authenticatedClient()
	return client.ListNodes()
}

// CreateNode creates a new node.
func (a *App) CreateNode(name, host string, bindPort int) error {
	client := a.authenticatedClient()
	return client.CreateNode(name, host, bindPort)
}

// ListProxies returns all proxies from the server.
func (a *App) ListProxies() ([]map[string]interface{}, error) {
	client := a.authenticatedClient()
	return client.ListProxies()
}

// CreateProxy creates a new tunnel via the server.
func (a *App) CreateProxy(name, proxyType string, localPort int, nodeID int) (string, error) {
	client := a.authenticatedClient()
	nid := &nodeID
	if nodeID <= 0 {
		nid = nil
	}
	data, err := client.CreateProxy(name, proxyType, localPort, nid)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// StartProxy starts a proxy tunnel.
func (a *App) StartProxy(proxyID int64) error {
	client := a.authenticatedClient()
	return client.ProxyAction(proxyID, "start")
}

// StopProxy stops a proxy tunnel.
func (a *App) StopProxy(proxyID int64) error {
	client := a.authenticatedClient()
	return client.ProxyAction(proxyID, "stop")
}

// DiscoverPorts scans localhost for listening ports.
func (a *App) DiscoverPorts() ([]discovery.PortInfo, error) {
	ports, err := discovery.DiscoverLocalPorts()
	if err != nil {
		return nil, fmt.Errorf("discover ports: %w", err)
	}
	return ports, nil
}

// QuickTunnel creates and starts a tunnel in one step.
// Returns the proxy ID and any error.
func (a *App) QuickTunnel(name string, localPort int, nodeID int) (int64, error) {
	client := a.authenticatedClient()
	nid := &nodeID
	if nodeID <= 0 {
		nid = nil
	}
	// Determine proxy type based on port conventions
	proxyType := "tcp"
	if localPort == 80 || localPort == 443 || localPort == 8080 {
		proxyType = "http"
	}

	data, err := client.CreateProxy(name, proxyType, localPort, nid)
	if err != nil {
		return 0, err
	}
	// Parse response to get proxy ID
	var proxy map[string]interface{}
	if err := json.Unmarshal(data, &proxy); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	id, ok := proxy["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected response")
	}
	proxyID := int64(id)

	// Start it
	if err := client.ProxyAction(proxyID, "start"); err != nil {
		return proxyID, fmt.Errorf("created but start failed: %w", err)
	}
	return proxyID, nil
}

func (a *App) authenticatedClient() *APIClient {
	client := NewAPIClient(a.config.ServerURL)
	client.token = a.config.Token
	return client
}
