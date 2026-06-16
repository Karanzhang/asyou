package handlers

import (
    "database/sql"
    "net/http"
    "net/http/httptest"
    "os/exec"
    "strconv"
    "strings"
    "testing"

    _ "github.com/mattn/go-sqlite3"
    "github.com/asyou/server/internal/frp"
    
)

func setupTestServer(t *testing.T) *Server {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    if err := db.Ping(); err != nil {
        t.Fatal(err)
    }
    // create minimal proxies table
    sqlStmt := `CREATE TABLE proxies (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, node_id INTEGER, name TEXT NOT NULL, type TEXT NOT NULL, local_ip TEXT, local_port INTEGER NOT NULL, remote_port INTEGER, subdomain TEXT, custom_domains TEXT, enable_tls INTEGER DEFAULT 0, status TEXT DEFAULT 'stopped', annotations TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME)`
    if _, err := db.Exec(sqlStmt); err != nil {
        t.Fatal(err)
    }
    // create minimal nodes table and insert a node
    nodeStmt := `CREATE TABLE nodes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, host TEXT NOT NULL, api_port INTEGER, bind_port INTEGER, tls_enabled INTEGER DEFAULT 1, auth_token TEXT, last_heartbeat DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME)`
    if _, err := db.Exec(nodeStmt); err != nil {
        t.Fatal(err)
    }
    if _, err := db.Exec(`INSERT INTO nodes (name, host, bind_port) VALUES (?, ?, ?)`, "node1", "127.0.0.1", 7000); err != nil {
        t.Fatal(err)
    }
    s := &Server{DB: db, FRP: frp.NewManager(), SSE: NewSSEHub()}
    // override manager to use sleep
    s.FRP.CmdBuilder = func(cfgPath string) *exec.Cmd {
        return exec.Command("/bin/sh", "-c", "sleep 1")
    }
    return s
}

func TestProxyActionStartStop(t *testing.T) {
    s := setupTestServer(t)
    // insert proxy
    res, err := s.DB.Exec(`INSERT INTO proxies (user_id, node_id, name, type, local_ip, local_port) VALUES (?, ?, ?, ?, ?, ?)`, 1, 1, "p1", "tcp", "127.0.0.1", 8080)
    if err != nil {
        t.Fatal(err)
    }
    id, _ := res.LastInsertId()

    if p, err := s.loadProxy(id); err != nil {
        t.Fatalf("loadProxy failed: %v", err)
    } else {
        t.Logf("loaded proxy: %+v", p)
    }

    // direct start attempt for diagnostics
    p, _ := s.loadProxy(id)
    node, err := s.loadNode(p.NodeID)
    if err != nil {
        t.Fatalf("loadNode failed: %v", err)
    }
    if err := s.FRP.Start(&p, node); err != nil {
        t.Fatalf("direct start failed: %v", err)
    }
    if _, err := s.DB.Exec(`UPDATE proxies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, "running", id); err != nil {
        t.Fatalf("direct update failed: %v", err)
    }
    // start via handler
    rr := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/api/v1/proxies/"+strconv.FormatInt(id,10)+"/action", strings.NewReader(`{"action":"start"}`))
    s.ProxyActionHandler(rr, req, strconv.FormatInt(id,10))
    if rr.Code != http.StatusAccepted {
        t.Fatalf("start expected 202 got %d body=%s", rr.Code, rr.Body.String())
    }

    // stop
    rr = httptest.NewRecorder()
    req = httptest.NewRequest("POST", "/api/v1/proxies/"+strconv.FormatInt(id,10)+"/action", strings.NewReader(`{"action":"stop"}`))
    s.ProxyActionHandler(rr, req, strconv.FormatInt(id,10))
    if rr.Code != http.StatusAccepted {
        t.Fatalf("stop expected 202 got %d body=%s", rr.Code, rr.Body.String())
    }
}
