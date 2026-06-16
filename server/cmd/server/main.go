package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/asyou/server/internal/db"
	"github.com/asyou/server/internal/frp"
	"github.com/asyou/server/internal/handlers"
)

func main() {
	cwd, _ := os.Getwd()
	migrationsDir := filepath.Join(cwd, "..", "migrations")
	dbPath := filepath.Join(cwd, "..", "asyou.db")
	log.Printf("migrations dir: %s", migrationsDir)
	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer dbConn.Close()
	if err := db.RunMigrations(dbConn, migrationsDir); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	frpManager := frp.NewManager()
	sseHub := handlers.NewSSEHub()
	acmeCfg := handlers.DefaultACMEConfig()
	s := &handlers.Server{DB: dbConn, FRP: frpManager, SSE: sseHub, ACME: acmeCfg}

	// Start periodic stats broadcaster (every 10s)
	s.StartStatsBroadcaster(10 * time.Second)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("asyou server is running. Use /api/v1/auth/register or /api/v1/auth/login."))
	})
	http.HandleFunc("/api/v1/auth/register", s.RegisterHandler)
	http.HandleFunc("/api/v1/auth/login", s.LoginHandler)
	// version info (no auth)
	http.HandleFunc("/api/v1/version", s.VersionHandler)
	// users (protected)
	http.HandleFunc("/api/v1/users/me", s.AuthMiddleware(s.UsersMeHandler))
	// nodes — list/create protected; item handler does JWT check internally
	http.HandleFunc("/api/v1/nodes", s.AuthMiddleware(s.NodesListCreateHandler))
	http.HandleFunc("/api/v1/nodes/", s.NodeItemHandler)
	// proxies (all protected)
	http.HandleFunc("/api/v1/proxies", s.AuthMiddleware(s.ProxiesListCreateHandler))
	http.HandleFunc("/api/v1/proxies/", s.AuthMiddleware(s.ProxyItemHandler))
	// audit logs (protected)
	http.HandleFunc("/api/v1/audit-logs", s.AuthMiddleware(s.AuditListHandler))
	// api keys (protected)
	http.HandleFunc("/api/v1/api-keys", s.AuthMiddleware(s.ApiKeysListCreateHandler))
	http.HandleFunc("/api/v1/api-keys/", s.AuthMiddleware(s.ApiKeyItemHandler))
	// metrics (no auth — used by Prometheus/node monitoring)
	http.HandleFunc("/api/v1/metrics", s.MetricsHandler)
	// SSE real-time events (auth via query param)
	http.HandleFunc("/api/v1/events", s.SSEHandler)
	// certificates (protected)
	http.HandleFunc("/api/v1/certs", s.AuthMiddleware(s.CertsListHandler))
	http.HandleFunc("/api/v1/certs/", s.AuthMiddleware(s.CertsItemHandler))

	log.Println("starting server :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
