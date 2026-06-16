package handlers

import (
    "fmt"
    "net/http"
)

// MetricsHandler exposes Prometheus-style server metrics at GET /api/v1/metrics
func (s *Server) MetricsHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
        return
    }

    var userCount, nodeCount, proxyCount, runningCount int

    _ = s.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount)
    _ = s.DB.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&nodeCount)
    _ = s.DB.QueryRow(`SELECT COUNT(*) FROM proxies`).Scan(&proxyCount)
    _ = s.DB.QueryRow(`SELECT COUNT(*) FROM proxies WHERE status = 'running'`).Scan(&runningCount)

    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "# HELP asyou_users_total Total number of registered users\n")
    fmt.Fprintf(w, "# TYPE asyou_users_total gauge\n")
    fmt.Fprintf(w, "asyou_users_total %d\n", userCount)
    fmt.Fprintf(w, "# HELP asyou_nodes_total Total number of nodes\n")
    fmt.Fprintf(w, "# TYPE asyou_nodes_total gauge\n")
    fmt.Fprintf(w, "asyou_nodes_total %d\n", nodeCount)
    fmt.Fprintf(w, "# HELP asyou_proxies_total Total number of proxies\n")
    fmt.Fprintf(w, "# TYPE asyou_proxies_total gauge\n")
    fmt.Fprintf(w, "asyou_proxies_total %d\n", proxyCount)
    fmt.Fprintf(w, "# HELP asyou_proxies_running Number of running proxies\n")
    fmt.Fprintf(w, "# TYPE asyou_proxies_running gauge\n")
    fmt.Fprintf(w, "asyou_proxies_running %d\n", runningCount)
}
