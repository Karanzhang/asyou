package frps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ServerInfo holds basic frps runtime information.
type ServerInfo struct {
	Version         string `json:"version"`
	BindPort        int    `json:"bindPort"`
	TotalTrafficIn  int64  `json:"totalTrafficIn"`
	TotalTrafficOut int64  `json:"totalTrafficOut"`
	CurConns        int    `json:"curConns"`
	ClientCounts    int    `json:"clientCounts"`
}

// ProxyConf holds the configuration of a proxy from frps admin API.
type ProxyConf struct {
	RemotePort int    `json:"remotePort"`
	LocalIP    string `json:"localIP"`
	LocalPort  int    `json:"localPort"`
	Type       string `json:"type"`
}

// ProxyEntry is a single proxy as returned by the frps admin API.
type ProxyEntry struct {
	Name            string     `json:"name"`
	Conf            *ProxyConf `json:"conf"`
	Status          string     `json:"status"`
	ClientID        string     `json:"clientID"`
	TodayTrafficIn  int64      `json:"todayTrafficIn"`
	TodayTrafficOut int64      `json:"todayTrafficOut"`
	CurConns        int        `json:"curConns"`
	LastStartTime   string     `json:"lastStartTime"`
	LastCloseTime   string     `json:"lastCloseTime"`
}

// ProxyListResponse wraps the proxy list returned by frps admin API.
type ProxyListResponse struct {
	Proxies []ProxyEntry `json:"proxies"`
}

// AdminClient communicates with frps's built-in admin API.
type AdminClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Username   string
	Password   string
}

// NewAdminClient creates a client for frps admin at address:port.
func NewAdminClient(addr string, port int) *AdminClient {
	return &AdminClient{
		BaseURL: fmt.Sprintf("http://%s:%d", addr, port),
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// NewAdminClientWithAuth creates a client with dashboard credentials.
func NewAdminClientWithAuth(addr string, port int, user, pwd string) *AdminClient {
	return &AdminClient{
		BaseURL:    fmt.Sprintf("http://%s:%d", addr, port),
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		Username:   user,
		Password:   pwd,
	}
}

// GetServerInfo fetches /api/serverinfo from frps.
func (c *AdminClient) GetServerInfo() (*ServerInfo, error) {
	var info ServerInfo
	if err := c.get("/api/serverinfo", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ListProxies fetches /api/proxy/:type from frps and returns the proxy list.
func (c *AdminClient) ListProxies(proxyType string) ([]ProxyEntry, error) {
	var resp ProxyListResponse
	if err := c.get("/api/proxy/"+proxyType, &resp); err != nil {
		return nil, err
	}
	return resp.Proxies, nil
}

// ListAllProxies fetches all proxy types from frps.
func (c *AdminClient) ListAllProxies() ([]ProxyEntry, error) {
	all := make([]ProxyEntry, 0)
	for _, t := range []string{"tcp", "http", "https", "udp", "stcp", "xtcp"} {
		list, err := c.ListProxies(t)
		if err != nil {
			continue
		}
		all = append(all, list...)
	}
	return all, nil
}

// GetProxyStats returns traffic stats for a specific proxy by name.
func (c *AdminClient) GetProxyStats(proxyType, name string) (*ProxyEntry, error) {
	list, err := c.ListProxies(proxyType)
	if err != nil {
		return nil, err
	}
	for _, p := range list {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("proxy %s not found", name)
}

// HealthCheck returns true if frps admin responds.
func (c *AdminClient) HealthCheck() bool {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/healthz")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (c *AdminClient) get(path string, dest interface{}) error {
	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("frps admin create request: %w", err)
	}
	if c.Username != "" || c.Password != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("frps admin GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("frps admin GET %s: status %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("frps admin decode: %w", err)
	}
	return nil
}
