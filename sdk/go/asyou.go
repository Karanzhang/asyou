// Package asyou provides a Go SDK for the asyou management API.
package asyou

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the asyou API client.
type Client struct {
	BaseURL    string
	httpClient *http.Client
	token      string
	useAPIKey  bool
}

// SetToken sets the auth token (JWT Bearer).
func (c *Client) SetToken(t string) { c.token = t; c.useAPIKey = false }

// Token returns the current auth token.
func (c *Client) Token() string { return c.token }

// SetAPIKey sets an API key for X-Api-Key header auth.
func (c *Client) SetAPIKey(k string) { c.token = k; c.useAPIKey = true }

// IsAPIKey returns true if using API key auth.
func (c *Client) IsAPIKey() bool { return c.useAPIKey }

// User represents an asyou platform user.
type User struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// Proxy represents a tunnel.
type Proxy struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	NodeID     *int64 `json:"node_id,omitempty"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalIP    string `json:"local_ip"`
	LocalPort  int    `json:"local_port"`
	RemotePort *int   `json:"remote_port,omitempty"`
	Subdomain  *string `json:"subdomain,omitempty"`
	Status     string `json:"status"`
}

// Node represents a frps server.
type Node struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	BindPort int    `json:"bind_port"`
}

// NewClient creates a new SDK client pointing at the given server URL.
func NewClient(serverURL string) *Client {
	return &Client{
		BaseURL:    serverURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) do(method, path string, body, result interface{}) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		if c.useAPIKey {
			req.Header.Set("X-Api-Key", c.token)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		var e struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		if json.Unmarshal(data, &e) == nil && e.Error != "" {
			return fmt.Errorf("%s", e.Error)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if result != nil && len(data) > 0 {
		return json.Unmarshal(data, result)
	}
	return nil
}

// Login authenticates with email and password.
func (c *Client) Login(email, password string) error {
	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.do("POST", "/api/v1/auth/login", map[string]string{
		"email": email, "password": password,
	}, &res); err != nil {
		return err
	}
	c.token = res.AccessToken
	c.useAPIKey = false
	return nil
}

// Register creates a new account and logs in.
func (c *Client) Register(email, password, displayName string) error {
	if err := c.do("POST", "/api/v1/auth/register", map[string]string{
		"email": email, "password": password, "display_name": displayName,
	}, nil); err != nil {
		return err
	}
	return c.Login(email, password)
}

// ForgotPassword sends a password reset email for the given address.
func (c *Client) ForgotPassword(email string) error {
	var res struct {
		Message string `json:"message"`
	}
	return c.do("POST", "/api/v1/auth/forgot-password", map[string]string{
		"email": email,
	}, &res)
}

// ResetPassword resets the user's password using a reset token.
func (c *Client) ResetPassword(token, newPassword string) error {
	var res struct {
		Message string `json:"message"`
	}
	return c.do("POST", "/api/v1/auth/reset-password", map[string]string{
		"token": token, "password": newPassword,
	}, &res)
}

// GetMe returns the currently authenticated user.
func (c *Client) GetMe() (*User, error) {
	var u User
	if err := c.do("GET", "/api/v1/users/me", nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// ListProxies returns all tunnels for the authenticated user.
func (c *Client) ListProxies() ([]Proxy, error) {
	var list []Proxy
	if err := c.do("GET", "/api/v1/proxies", nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// GetProxy returns a single tunnel by ID.
func (c *Client) GetProxy(id int64) (*Proxy, error) {
	var p Proxy
	if err := c.do("GET", fmt.Sprintf("/api/v1/proxies/%d", id), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// CreateProxy creates a new tunnel.
func (c *Client) CreateProxy(name, proxyType string, localPort int, nodeID int, remotePort int, subdomain string) (*Proxy, error) {
	body := map[string]interface{}{
		"name": name, "type": proxyType, "local_port": localPort,
	}
	if nodeID > 0 {
		body["node_id"] = nodeID
	}
	if remotePort > 0 {
		body["remote_port"] = remotePort
	}
	if subdomain != "" {
		body["subdomain"] = subdomain
	}
	var p Proxy
	if err := c.do("POST", "/api/v1/proxies", body, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// DeleteProxy deletes a tunnel by ID.
func (c *Client) DeleteProxy(id int64) error {
	return c.do("DELETE", fmt.Sprintf("/api/v1/proxies/%d", id), nil, nil)
}

// ProxyAction sends a lifecycle action (start/stop/reload) to a tunnel.
func (c *Client) ProxyAction(id int64, action string) error {
	return c.do("POST", fmt.Sprintf("/api/v1/proxies/%d/action", id),
		map[string]string{"action": action}, nil)
}

// ListNodes returns all frps nodes.
func (c *Client) ListNodes() ([]Node, error) {
	var list []Node
	if err := c.do("GET", "/api/v1/nodes", nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// GetNode returns a single node by ID.
func (c *Client) GetNode(id int64) (*Node, error) {
	var n Node
	if err := c.do("GET", fmt.Sprintf("/api/v1/nodes/%d", id), nil, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// GetVersion returns server and frp version information.
func (c *Client) GetVersion() (map[string]interface{}, error) {
	var info map[string]interface{}
	if err := c.do("GET", "/api/v1/version", nil, &info); err != nil {
		return nil, err
	}
	return info, nil
}
