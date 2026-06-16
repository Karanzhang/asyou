package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEHub manages Server-Sent Event connections and broadcasts.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewSSEHub creates a new SSE hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan string]struct{}),
	}
}

// Subscribe adds a client channel and returns it.
func (h *SSEHub) Subscribe() chan string {
	ch := make(chan string, 64)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a client channel.
func (h *SSEHub) Unsubscribe(ch chan string) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

// Broadcast sends a message to all connected clients.
func (h *SSEHub) Broadcast(data interface{}) {
	msg, err := json.Marshal(data)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- string(msg):
		default:
			// drop if client is slow
		}
	}
}

// SSEMessage is the envelope for all SSE events.
type SSEMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SSEHandler handles GET /api/v1/events (SSE endpoint).
// Auth is done via ?token= query param.
func (s *Server) SSEHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate via query token
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// try Authorization header
		auth := r.Header.Get("Authorization")
		if len(auth) > 7 && auth[:7] == "Bearer " {
			tokenStr = auth[7:]
		}
	}
	if tokenStr == "" {
		writeJSONError(w, "missing token", "UNAUTHORIZED", http.StatusUnauthorized)
		return
	}
	// Validate by re-parsing JWT
	userID, err := s.validateJWT(r)
	if err != nil {
		// For SSE, we accept token via query param too
		// Clone the request to set Authorization header
		r2 := *r
		r2.Header = r.Header.Clone()
		if r2.Header.Get("Authorization") == "" {
			r2.Header.Set("Authorization", "Bearer "+tokenStr)
		}
		userID, err = s.validateJWT(&r2)
		if err != nil {
			writeJSONError(w, "unauthorized", "UNAUTHORIZED", http.StatusUnauthorized)
			return
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Notify client about user context
	fmt.Fprintf(w, "event: connected\ndata: {\"user_id\":%d}\n\n", userID)
	flusher.Flush()

	ch := s.SSE.Subscribe()
	defer s.SSE.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			// keepalive
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// broadcastProxyUpdate sends a proxy status change to all SSE clients.
func (s *Server) broadcastProxyUpdate(proxyID int64, status string, annotations string) {
	s.SSE.Broadcast(SSEMessage{
		Type: "proxy_update",
		Data: map[string]interface{}{
			"id":          proxyID,
			"status":      status,
			"annotations": annotations,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		},
	})
}

// StartStatsBroadcaster periodically pushes traffic stats to SSE clients.
func (s *Server) StartStatsBroadcaster(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			// Fetch recent stats for all proxies and broadcast
			rows, err := s.DB.Query(`SELECT proxy_id, SUM(bytes_in), SUM(bytes_out), SUM(conn_count) FROM proxy_stats WHERE timestamp > datetime('now', '-1 minute') GROUP BY proxy_id`)
			if err != nil {
				continue
			}
			type statSummary struct {
				ProxyID   int64 `json:"proxy_id"`
				BytesIn   int64 `json:"bytes_in"`
				BytesOut  int64 `json:"bytes_out"`
				ConnCount int   `json:"conn_count"`
			}
			var stats []statSummary
			for rows.Next() {
				var s statSummary
				if err := rows.Scan(&s.ProxyID, &s.BytesIn, &s.BytesOut, &s.ConnCount); err == nil {
					stats = append(stats, s)
				}
			}
			rows.Close()
			if len(stats) > 0 {
				s.SSE.Broadcast(SSEMessage{
					Type: "stats_update",
					Data: stats,
				})
			}
		}
	}()
}
