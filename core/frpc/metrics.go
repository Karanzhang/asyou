package frpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Metrics holds runtime stats scraped from frpc's admin API.
type Metrics struct {
	ProxyID     int64     `json:"proxy_id"`
	ScrapedAt   time.Time `json:"scraped_at"`
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	ConnCount   int       `json:"conn_count"`
	LastError   string    `json:"last_error,omitempty"`
}

// adminStatusResponse mirrors the JSON from frpc admin /api/status.
type adminStatusResponse struct {
	TCP  []proxyStats `json:"tcp"`
	UDP  []proxyStats `json:"udp"`
	HTTP []proxyStats `json:"http"`
}

type proxyStats struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
	ConnCount  int    `json:"conn_count"`
	Err        string `json:"err"`
}

// Scrape connects to frpc's admin API at the given address and returns metrics.
func Scrape(adminAddr string, adminPort int) ([]Metrics, error) {
	url := fmt.Sprintf("http://%s:%d/api/status", adminAddr, adminPort)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("scrape frpc admin: %w", err)
	}
	defer resp.Body.Close()

	var status adminStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode frpc admin response: %w", err)
	}

	var out []Metrics
	now := time.Now().UTC()
	for _, p := range status.TCP {
		out = append(out, Metrics{ScrapedAt: now, BytesIn: p.BytesIn, BytesOut: p.BytesOut, ConnCount: p.ConnCount, LastError: p.Err})
	}
	for _, p := range status.UDP {
		out = append(out, Metrics{ScrapedAt: now, BytesIn: p.BytesIn, BytesOut: p.BytesOut, ConnCount: p.ConnCount, LastError: p.Err})
	}
	for _, p := range status.HTTP {
		out = append(out, Metrics{ScrapedAt: now, BytesIn: p.BytesIn, BytesOut: p.BytesOut, ConnCount: p.ConnCount, LastError: p.Err})
	}
	return out, nil
}
