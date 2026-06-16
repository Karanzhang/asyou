package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os/exec"
    "strconv"
    "strings"
    "testing"

    _ "github.com/mattn/go-sqlite3"
    "github.com/asyou/server/internal/frp"
)

type annotationRecord struct {
    Error string `json:"error"`
    When  string `json:"when"`
}

func setupAnnotationTestServer(t *testing.T) *Server {
    db, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    sqlStmt := `CREATE TABLE proxies (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL, node_id INTEGER, name TEXT NOT NULL, type TEXT NOT NULL, local_ip TEXT, local_port INTEGER NOT NULL, remote_port INTEGER, subdomain TEXT, custom_domains TEXT, enable_tls INTEGER DEFAULT 0, status TEXT DEFAULT 'stopped', annotations TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME)`
    if _, err := db.Exec(sqlStmt); err != nil {
        t.Fatal(err)
    }
    nodeStmt := `CREATE TABLE nodes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, host TEXT NOT NULL, api_port INTEGER, bind_port INTEGER, tls_enabled INTEGER DEFAULT 1, auth_token TEXT, last_heartbeat DATETIME, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME)`
    if _, err := db.Exec(nodeStmt); err != nil {
        t.Fatal(err)
    }
    if _, err := db.Exec(`INSERT INTO nodes (name, host, bind_port) VALUES (?, ?, ?)`, "node1", "127.0.0.1", 7000); err != nil {
        t.Fatal(err)
    }
    s := &Server{DB: db, FRP: frp.NewManager(), SSE: NewSSEHub()}
    s.FRP.CmdBuilder = func(cfgPath string) *exec.Cmd {
        return exec.Command("nonexistent-binary")
    }
    return s
}

func TestProxyActionStartFailureAnnotationJSON(t *testing.T) {
    s := setupAnnotationTestServer(t)
    res, err := s.DB.Exec(`INSERT INTO proxies (user_id, node_id, name, type, local_ip, local_port) VALUES (?, ?, ?, ?, ?, ?)`, 1, 1, "p1", "tcp", "127.0.0.1", 8080)
    if err != nil {
        t.Fatal(err)
    }
    id, _ := res.LastInsertId()

    rr := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/api/v1/proxies/"+strconv.FormatInt(id, 10)+"/action", strings.NewReader(`{"action":"start"}`))
    s.ProxyActionHandler(rr, req, strconv.FormatInt(id, 10))
    if rr.Code != http.StatusInternalServerError {
        t.Fatalf("expected 500 got %d", rr.Code)
    }

    var annotation string
    if err := s.DB.QueryRow(`SELECT annotations FROM proxies WHERE id = ?`, id).Scan(&annotation); err != nil {
        t.Fatal(err)
    }
    var rec annotationRecord
    if err := json.Unmarshal([]byte(annotation), &rec); err != nil {
        t.Fatalf("annotation is not JSON: %v", err)
    }
    if rec.Error == "" || rec.When != "start" {
        t.Fatalf("unexpected annotation content: %#v", rec)
    }
}
