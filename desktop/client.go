package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient communicates with the asyou management server.
type APIClient struct {
	BaseURL string
	token   string
	client  *http.Client
}

// NewAPIClient creates a new client for the asyou server.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *APIClient) request(method, path string, body interface{}) ([]byte, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		json.Unmarshal(data, &errResp)
		if errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return data, nil
}

// Login authenticates with the server.
func (c *APIClient) Login(email, password string) error {
	data, err := c.request("POST", "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return err
	}
	var res struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return err
	}
	c.token = res.AccessToken
	return nil
}

// Register creates a new account and logs in.
func (c *APIClient) Register(email, password, displayName string) error {
	_, err := c.request("POST", "/api/v1/auth/register", map[string]string{
		"email":        email,
		"password":     password,
		"display_name": displayName,
	})
	if err != nil {
		return err
	}
	return c.Login(email, password)
}

// GetUser returns the current user.
func (c *APIClient) GetUser() (map[string]interface{}, error) {
	data, err := c.request("GET", "/api/v1/users/me", nil)
	if err != nil {
		return nil, err
	}
	var user map[string]interface{}
	json.Unmarshal(data, &user)
	return user, nil
}

// ListNode returns all nodes.
func (c *APIClient) ListNodes() ([]map[string]interface{}, error) {
	data, err := c.request("GET", "/api/v1/nodes", nil)
	if err != nil {
		return nil, err
	}
	var nodes []map[string]interface{}
	json.Unmarshal(data, &nodes)
	return nodes, nil
}

// CreateNode creates a new node.
func (c *APIClient) CreateNode(name, host string, bindPort int) error {
	_, err := c.request("POST", "/api/v1/nodes", map[string]interface{}{
		"name":       name,
		"host":       host,
		"bind_port":  bindPort,
	})
	return err
}

// ListProxies returns all proxies.
func (c *APIClient) ListProxies() ([]map[string]interface{}, error) {
	data, err := c.request("GET", "/api/v1/proxies", nil)
	if err != nil {
		return nil, err
	}
	var proxies []map[string]interface{}
	json.Unmarshal(data, &proxies)
	return proxies, nil
}

// CreateProxy creates a new proxy.
func (c *APIClient) CreateProxy(name, proxyType string, localPort int, nodeID *int) ([]byte, error) {
	body := map[string]interface{}{
		"name":       name,
		"type":       proxyType,
		"local_port": localPort,
	}
	if nodeID != nil {
		body["node_id"] = *nodeID
	}
	return c.request("POST", "/api/v1/proxies", body)
}

// ProxyAction sends a lifecycle action to a proxy.
func (c *APIClient) ProxyAction(id int64, action string) error {
	_, err := c.request("POST", fmt.Sprintf("/api/v1/proxies/%d/action", id), map[string]string{
		"action": action,
	})
	return err
}

// GetProxyStats returns stats for a proxy.
func (c *APIClient) GetProxyStats(id int64, limit int) ([]map[string]interface{}, error) {
	data, err := c.request("GET", fmt.Sprintf("/api/v1/proxies/%d/stats?limit=%d", id, limit), nil)
	if err != nil {
		return nil, err
	}
	var stats []map[string]interface{}
	json.Unmarshal(data, &stats)
	return stats, nil
}

// IsAuthenticated returns whether the client has a token.
func (c *APIClient) IsAuthenticated() bool {
	return c.token != ""
}
