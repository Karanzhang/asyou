package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "path"
    "strconv"
    "strings"
    "time"

    "github.com/asyou/core/frps"
    "github.com/asyou/server/internal/model"
)

type proxyCreateRequest struct {
    Name         string   `json:"name"`
    Type         string   `json:"type"`
    LocalIP      *string  `json:"local_ip,omitempty"`
    LocalPort    int      `json:"local_port"`
    RemotePort   *int     `json:"remote_port,omitempty"`
    Subdomain    *string  `json:"subdomain,omitempty"`
    CustomDomains []string `json:"custom_domains,omitempty"`
    NodeID       *int64   `json:"node_id,omitempty"`
}

// ProxiesListCreateHandler handles GET /proxies and POST /proxies
func (s *Server) ProxiesListCreateHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        userID := mustGetUserID(r)
        var rows *sql.Rows
        var err error
        if s.isAdmin(r) {
            rows, err = s.DB.Query(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies ORDER BY id`)
        } else {
            rows, err = s.DB.Query(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies WHERE user_id = ? ORDER BY id`, userID)
        }
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        defer rows.Close()
        list := make([]model.Proxy, 0)
        for rows.Next() {
            var p model.Proxy
            var nodeID sql.NullInt64
            var remotePort sql.NullInt64
            var customDomains sql.NullString
            var annotations sql.NullString
            var enableTls sql.NullInt64
            var createdAt sql.NullString
            var updatedAt sql.NullString
            if err := rows.Scan(&p.ID, &p.UserID, &nodeID, &p.Name, &p.Type, &p.LocalIP, &p.LocalPort, &remotePort, &p.Subdomain, &customDomains, &enableTls, &p.Status, &annotations, &createdAt, &updatedAt); err != nil {
                writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
                return
            }
            if nodeID.Valid {
                n := int64(nodeID.Int64)
                p.NodeID = &n
            }
            if remotePort.Valid {
                rp := int(remotePort.Int64)
                p.RemotePort = &rp
            }
            if customDomains.Valid {
                p.CustomDomains = &customDomains.String
            }
            if annotations.Valid {
                p.Annotations = &annotations.String
            }
            if enableTls.Valid {
                p.EnableTls = enableTls.Int64 != 0
            }
            if createdAt.Valid {
                if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
                    p.CreatedAt = t
                }
            }
            if updatedAt.Valid {
                if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
                    p.UpdatedAt = t
                }
            }
            list = append(list, p)
        }

        // Live status check: group stopped proxies by node, query frps once per node
        type nodeProxies struct {
            ids     map[int]struct{}
            names   map[string]int // proxy name → index in list
        }
        nodeMap := make(map[int64]*nodeProxies)
        for i := range list {
            if list[i].Status != "stopped" || list[i].NodeID == nil {
                continue
            }
            nid := *list[i].NodeID
            if nodeMap[nid] == nil {
                nodeMap[nid] = &nodeProxies{ids: map[int]struct{}{}, names: make(map[string]int)}
            }
            nodeMap[nid].ids[i] = struct{}{}
            nodeMap[nid].names[list[i].Name] = i
        }
        for nid, np := range nodeMap {
            proxies, err := s.fetchNodeProxies(nid)
            if err != nil {
                continue
            }
            for _, p := range proxies {
                if idx, ok := np.names[p.Name]; ok && p.Status == "online" {
                    list[idx].Status = "running (local)"
                }
            }
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(list)
    case http.MethodPost:
        var req proxyCreateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        if req.Name == "" || req.LocalPort == 0 {
            writeJSONError(w, "name and local_port required", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        localIP := "127.0.0.1"
        if req.LocalIP != nil {
            localIP = *req.LocalIP
        }
        cdoms := ""
        if len(req.CustomDomains) > 0 {
            // simple JSON encode inline
            b, _ := json.Marshal(req.CustomDomains)
            cdoms = string(b)
        }
        var nodeID interface{}
        if req.NodeID != nil {
            nodeID = *req.NodeID
        }
        // Auto-assign remote port if not specified
        var remotePortVal interface{}
        if req.RemotePort != nil {
            remotePortVal = *req.RemotePort
        } else if req.NodeID != nil {
            startPort := s.ProxyStartPort
            if startPort == 0 {
                startPort = 31000
            }
            remotePortVal = startPort
        }
        res, err := s.DB.Exec(`INSERT INTO proxies (user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, mustGetUserID(r), nodeID, req.Name, req.Type, localIP, req.LocalPort, remotePortVal, req.Subdomain, cdoms, 0, "stopped", "")
        if err != nil {
            writeJSONError(w, "cannot create proxy: "+err.Error(), "INTERNAL", http.StatusInternalServerError)
            return
        }
        newID, _ := res.LastInsertId()
        // Return the created proxy
        var p model.Proxy
        var pNodeID sql.NullInt64
        var pRemotePort sql.NullInt64
        var pCustomDomains sql.NullString
        var pAnnotations sql.NullString
        var pEnableTls sql.NullInt64
        var pCreatedAt sql.NullString
        var pUpdatedAt sql.NullString
        s.DB.QueryRow(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies WHERE id = ?`, newID).
            Scan(&p.ID, &p.UserID, &pNodeID, &p.Name, &p.Type, &p.LocalIP, &p.LocalPort, &pRemotePort, &p.Subdomain, &pCustomDomains, &pEnableTls, &p.Status, &pAnnotations, &pCreatedAt, &pUpdatedAt)
        if pNodeID.Valid {
            n := int64(pNodeID.Int64)
            p.NodeID = &n
        }
        if pRemotePort.Valid {
            rp := int(pRemotePort.Int64)
            p.RemotePort = &rp
        }
        if pCustomDomains.Valid {
            p.CustomDomains = &pCustomDomains.String
        }
        if pAnnotations.Valid {
            p.Annotations = &pAnnotations.String
        }
        if pEnableTls.Valid {
            p.EnableTls = pEnableTls.Int64 != 0
        }
        if pCreatedAt.Valid {
            if t, err := time.Parse(time.RFC3339, pCreatedAt.String); err == nil {
                p.CreatedAt = t
            }
        }
        if pUpdatedAt.Valid {
            if t, err := time.Parse(time.RFC3339, pUpdatedAt.String); err == nil {
                p.UpdatedAt = t
            }
        }
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(p)
    default:
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
    }
}

// ProxyItemHandler handles GET/PUT/DELETE and /action under /proxies/{id}
func (s *Server) ProxyItemHandler(w http.ResponseWriter, r *http.Request) {
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/proxies/")
    if idStr == "" {
        writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    if strings.HasSuffix(idStr, "/action") {
        idStr = strings.TrimSuffix(idStr, "/action")
        idStr = strings.TrimSuffix(idStr, "/")
        s.ProxyActionHandler(w, r, idStr)
        return
    }
    if strings.HasSuffix(idStr, "/stats") {
        idStr = strings.TrimSuffix(idStr, "/stats")
        idStr = strings.TrimSuffix(idStr, "/")
        s.ProxyStatsHandler(w, r, idStr)
        return
    }
    idStr = path.Clean(idStr)
    id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    switch r.Method {
    case http.MethodGet:
        userID := mustGetUserID(r)
        var p model.Proxy
        var nodeID sql.NullInt64
        var remotePort sql.NullInt64
        var customDomains sql.NullString
        var annotations sql.NullString
        var enableTls sql.NullInt64
        var createdAt sql.NullString
        var updatedAt sql.NullString
        var err error
        if s.isAdmin(r) {
            err = s.DB.QueryRow(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies WHERE id = ?`, id).
                Scan(&p.ID, &p.UserID, &nodeID, &p.Name, &p.Type, &p.LocalIP, &p.LocalPort, &remotePort, &p.Subdomain, &customDomains, &enableTls, &p.Status, &annotations, &createdAt, &updatedAt)
        } else {
            err = s.DB.QueryRow(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies WHERE id = ? AND user_id = ?`, id, userID).
                Scan(&p.ID, &p.UserID, &nodeID, &p.Name, &p.Type, &p.LocalIP, &p.LocalPort, &remotePort, &p.Subdomain, &customDomains, &enableTls, &p.Status, &annotations, &createdAt, &updatedAt)
        }
        if err == sql.ErrNoRows {
            writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
            return
        } else if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        if nodeID.Valid {
            n := int64(nodeID.Int64)
            p.NodeID = &n
        }
        if remotePort.Valid {
            rp := int(remotePort.Int64)
            p.RemotePort = &rp
        }
        if customDomains.Valid {
            p.CustomDomains = &customDomains.String
        }
        if annotations.Valid {
            p.Annotations = &annotations.String
        }
        if enableTls.Valid {
            p.EnableTls = enableTls.Int64 != 0
        }
        if createdAt.Valid {
            if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
                p.CreatedAt = t
            }
        }
        if updatedAt.Valid {
            if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
                p.UpdatedAt = t
            }
        }

        // Live status check: if DB says stopped but proxy has a node, check frps
        if p.Status == "stopped" && p.NodeID != nil {
            liveStatus := s.checkProxyLiveStatus(*p.NodeID, p.Name)
            if liveStatus != "" {
                p.Status = liveStatus
            }
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(p)
    case http.MethodPut:
        var upd struct {
            LocalPort    *int     `json:"local_port,omitempty"`
            RemotePort   *int     `json:"remote_port,omitempty"`
            Subdomain    *string  `json:"subdomain,omitempty"`
            CustomDomains *[]string `json:"custom_domains,omitempty"`
            EnableTls    *bool    `json:"enable_tls,omitempty"`
        }
        if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
            writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        var set []string
        var args []interface{}
        if upd.LocalPort != nil {
            set = append(set, "local_port = ?")
            args = append(args, *upd.LocalPort)
        }
        if upd.RemotePort != nil {
            set = append(set, "remote_port = ?")
            args = append(args, *upd.RemotePort)
        }
        if upd.Subdomain != nil {
            set = append(set, "subdomain = ?")
            args = append(args, *upd.Subdomain)
        }
        if upd.CustomDomains != nil {
            b, _ := json.Marshal(upd.CustomDomains)
            set = append(set, "custom_domains = ?")
            args = append(args, string(b))
        }
        if upd.EnableTls != nil {
            val := 0
            if *upd.EnableTls {
                val = 1
            }
            set = append(set, "enable_tls = ?")
            args = append(args, val)
        }
        if len(set) == 0 {
            writeJSONError(w, "nothing to update", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        set = append(set, "updated_at = CURRENT_TIMESTAMP")
        userID := mustGetUserID(r)
        query := "UPDATE proxies SET " + strings.Join(set, ", ") + " WHERE id = ?"
        args = append(args, id)
        if !s.isAdmin(r) {
            query += " AND user_id = ?"
            args = append(args, userID)
        }
        _, err := s.DB.Exec(query, args...)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        // return updated resource
        var p model.Proxy
        var nodeID sql.NullInt64
        var remotePort sql.NullInt64
        var customDomains sql.NullString
        var annotations sql.NullString
        var enableTls sql.NullInt64
        var createdAt sql.NullString
        var updatedAt sql.NullString
        if s.isAdmin(r) {
            err = s.DB.QueryRow(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies WHERE id = ?`, id).
                Scan(&p.ID, &p.UserID, &nodeID, &p.Name, &p.Type, &p.LocalIP, &p.LocalPort, &remotePort, &p.Subdomain, &customDomains, &enableTls, &p.Status, &annotations, &createdAt, &updatedAt)
        } else {
            err = s.DB.QueryRow(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies WHERE id = ? AND user_id = ?`, id, userID).
                Scan(&p.ID, &p.UserID, &nodeID, &p.Name, &p.Type, &p.LocalIP, &p.LocalPort, &remotePort, &p.Subdomain, &customDomains, &enableTls, &p.Status, &annotations, &createdAt, &updatedAt)
        }
        if err == sql.ErrNoRows {
            writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
            return
        } else if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        if nodeID.Valid {
            n := int64(nodeID.Int64)
            p.NodeID = &n
        }
        if remotePort.Valid {
            rp := int(remotePort.Int64)
            p.RemotePort = &rp
        }
        if customDomains.Valid {
            p.CustomDomains = &customDomains.String
        }
        if annotations.Valid {
            p.Annotations = &annotations.String
        }
        if enableTls.Valid {
            p.EnableTls = enableTls.Int64 != 0
        }
        if createdAt.Valid {
            if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
                p.CreatedAt = t
            }
        }
        if updatedAt.Valid {
            if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
                p.UpdatedAt = t
            }
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(p)
    case http.MethodDelete:
        userID := mustGetUserID(r)
        var res sql.Result
        if s.isAdmin(r) {
            res, err = s.DB.Exec(`DELETE FROM proxies WHERE id = ?`, id)
        } else {
            res, err = s.DB.Exec(`DELETE FROM proxies WHERE id = ? AND user_id = ?`, id, userID)
        }
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        affected, _ := res.RowsAffected()
        if affected == 0 {
            writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    default:
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
    }
}

// ProxyActionHandler handles POST /proxies/{id}/action
func (s *Server) ProxyActionHandler(w http.ResponseWriter, r *http.Request, idStr string) {
    if r.Method != http.MethodPost {
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
        return
    }
    id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    var body struct{ Action string `json:"action"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    proxy, err := s.loadProxy(id)
    if err == sql.ErrNoRows {
        writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
        return
    } else if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    node, err := s.loadNode(proxy.NodeID)
    if err != nil && err != sql.ErrNoRows {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    switch body.Action {
    case "start":
        if s.FRP == nil {
            writeJSONError(w, "frp manager unavailable", "INTERNAL", http.StatusInternalServerError)
            return
        }
        if err := s.FRP.Start(&proxy, node); err != nil {
            s.recordProxyError(id, "start", err)
            writeJSONError(w, "failed to start proxy: "+err.Error(), "INTERNAL", http.StatusInternalServerError)
            return
        }
        _, err := s.DB.Exec(`UPDATE proxies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, "running", id)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        s.broadcastProxyUpdate(id, "running", "")
        w.WriteHeader(http.StatusAccepted)
    case "stop":
        if s.FRP == nil {
            writeJSONError(w, "frp manager unavailable", "INTERNAL", http.StatusInternalServerError)
            return
        }
        if err := s.FRP.Stop(id); err != nil {
            s.recordProxyError(id, "stop", err)
            writeJSONError(w, "failed to stop proxy: "+err.Error(), "INTERNAL", http.StatusInternalServerError)
            return
        }
        _, err := s.DB.Exec(`UPDATE proxies SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, "stopped", id)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        s.broadcastProxyUpdate(id, "stopped", "")
        w.WriteHeader(http.StatusAccepted)
    case "reload":
        // implement reload as stop then start, recording errors to annotations
        if s.FRP == nil {
            writeJSONError(w, "frp manager unavailable", "INTERNAL", http.StatusInternalServerError)
            return
        }
        if err := s.FRP.Stop(id); err != nil {
            s.recordProxyError(id, "reload_stop", err)
            writeJSONError(w, "failed to stop: "+err.Error(), "INTERNAL", http.StatusInternalServerError)
            return
        }
        if err := s.FRP.Start(&proxy, node); err != nil {
            s.recordProxyError(id, "reload_start", err)
            writeJSONError(w, "failed to start: "+err.Error(), "INTERNAL", http.StatusInternalServerError)
            return
        }
        _, err := s.DB.Exec(`UPDATE proxies SET status = ?, annotations = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, "running", "", id)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusAccepted)
    default:
        writeJSONError(w, "unknown action", "BAD_REQUEST", http.StatusBadRequest)
    }
}

func (s *Server) loadProxy(id int64) (model.Proxy, error) {
    var p model.Proxy
    var nodeID sql.NullInt64
    var remotePort sql.NullInt64
    var customDomains sql.NullString
    var annotations sql.NullString
    var enableTls sql.NullInt64
    var createdAt sql.NullString
    var updatedAt sql.NullString
    err := s.DB.QueryRow(`SELECT id, user_id, node_id, name, type, local_ip, local_port, remote_port, subdomain, custom_domains, enable_tls, status, annotations, created_at, updated_at FROM proxies WHERE id = ?`, id).
        Scan(&p.ID, &p.UserID, &nodeID, &p.Name, &p.Type, &p.LocalIP, &p.LocalPort, &remotePort, &p.Subdomain, &customDomains, &enableTls, &p.Status, &annotations, &createdAt, &updatedAt)
    if err != nil {
        return p, err
    }
    if nodeID.Valid {
        n := int64(nodeID.Int64)
        p.NodeID = &n
    }
    if remotePort.Valid {
        rp := int(remotePort.Int64)
        p.RemotePort = &rp
    }
    if customDomains.Valid {
        p.CustomDomains = &customDomains.String
    }
    if annotations.Valid {
        p.Annotations = &annotations.String
    }
    if enableTls.Valid {
        p.EnableTls = enableTls.Int64 != 0
    }
    if createdAt.Valid {
        if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
            p.CreatedAt = t
        }
    }
    if updatedAt.Valid {
        if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
            p.UpdatedAt = t
        }
    }
    return p, nil
}

func (s *Server) recordProxyError(id int64, when string, err error) {
    msg := err.Error()
    if s.FRP != nil {
        if last := s.FRP.LastError(id); last != "" {
            msg = last
        }
    }
    annotation := struct {
        Error string `json:"error"`
        When  string `json:"when"`
    }{Error: msg, When: when}
    data, _ := json.Marshal(annotation)
    _, _ = s.DB.Exec(`UPDATE proxies SET annotations = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, string(data), id)
}

func (s *Server) loadNode(nodeID *int64) (*model.Node, error) {
    if nodeID == nil {
        return nil, sql.ErrNoRows
    }
    var n model.Node
    var apiPort sql.NullInt64
    var bindPort sql.NullInt64
    var tlsEnabled sql.NullInt64
    var authToken sql.NullString
    var lastHeartbeat sql.NullString
    var createdAt sql.NullString
    var updatedAt sql.NullString
    err := s.DB.QueryRow(`SELECT id, name, host, api_port, bind_port, tls_enabled, auth_token, last_heartbeat, created_at, updated_at FROM nodes WHERE id = ?`, *nodeID).
        Scan(&n.ID, &n.Name, &n.Host, &apiPort, &bindPort, &tlsEnabled, &authToken, &lastHeartbeat, &createdAt, &updatedAt)
    if err != nil {
        return nil, err
    }
    if apiPort.Valid {
        n.ApiPort = int(apiPort.Int64)
    }
    if bindPort.Valid {
        n.BindPort = int(bindPort.Int64)
    }
    if tlsEnabled.Valid {
        n.TlsEnabled = tlsEnabled.Int64 != 0
    }
    if authToken.Valid {
        n.AuthToken = authToken.String
    }
    if lastHeartbeat.Valid {
        if t, err := time.Parse(time.RFC3339, lastHeartbeat.String); err == nil {
            n.LastHeartbeat = t
        }
    }
    if createdAt.Valid {
        if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
            n.CreatedAt = t
        }
    }
    if updatedAt.Valid {
        if t, err := time.Parse(time.RFC3339, updatedAt.String); err == nil {
            n.UpdatedAt = t
        }
    }
    return &n, nil
}

// fetchNodeProxies queries frps admin API for all active proxies on a node.
func (s *Server) fetchNodeProxies(nodeID int64) ([]frps.ProxyEntry, error) {
    var dashPort sql.NullInt64
    var host, dashUser, dashPwd sql.NullString
    err := s.DB.QueryRow(`SELECT host, dashboard_port, dashboard_user, dashboard_pwd FROM nodes WHERE id = ?`, nodeID).
        Scan(&host, &dashPort, &dashUser, &dashPwd)
    if err != nil || !host.Valid || host.String == "" {
        return nil, fmt.Errorf("node not found")
    }
    apiPort := 7500
    if dashPort.Valid {
        apiPort = int(dashPort.Int64)
    }
    user := "admin"
    if dashUser.Valid && dashUser.String != "" {
        user = dashUser.String
    }
    pwd := ""
    if dashPwd.Valid {
        pwd = dashPwd.String
    }
    client := frps.NewAdminClientWithAuth(host.String, apiPort, user, pwd)
    return client.ListAllProxies()
}

// checkProxyLiveStatus queries frps admin API to check if a proxy is actually running.
func (s *Server) checkProxyLiveStatus(nodeID int64, proxyName string) string {
    list, err := s.fetchNodeProxies(nodeID)
    if err != nil {
        return ""
    }
    for _, p := range list {
        if p.Name == proxyName && p.Status == "online" {
            return "running (local)"
        }
    }
    return ""
}

// ProxyStatsHandler handles POST/GET /proxies/{id}/stats
func (s *Server) ProxyStatsHandler(w http.ResponseWriter, r *http.Request, idStr string) {
    id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    switch r.Method {
    case http.MethodPost:
        var req struct {
            Timestamp *time.Time `json:"timestamp,omitempty"`
            BytesIn   *int64     `json:"bytes_in,omitempty"`
            BytesOut  *int64     `json:"bytes_out,omitempty"`
            ConnCount *int       `json:"conn_count,omitempty"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        ts := time.Now().UTC()
        if req.Timestamp != nil {
            ts = req.Timestamp.UTC()
        }
        bi := int64(0)
        bo := int64(0)
        cc := 0
        if req.BytesIn != nil {
            bi = *req.BytesIn
        }
        if req.BytesOut != nil {
            bo = *req.BytesOut
        }
        if req.ConnCount != nil {
            cc = *req.ConnCount
        }
        _, err := s.DB.Exec(`INSERT INTO proxy_stats (proxy_id, timestamp, bytes_in, bytes_out, conn_count) VALUES (?, ?, ?, ?, ?)`, id, ts.Format(time.RFC3339), bi, bo, cc)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusCreated)
    case http.MethodGet:
        q := r.URL.Query()
        limit := 100
        if l := q.Get("limit"); l != "" {
            if v, err := strconv.Atoi(l); err == nil && v > 0 {
                limit = v
            }
        }
        rows, err := s.DB.Query(`SELECT id, proxy_id, timestamp, bytes_in, bytes_out, conn_count FROM proxy_stats WHERE proxy_id = ? ORDER BY timestamp DESC LIMIT ?`, id, limit)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        defer rows.Close()
        out := make([]model.ProxyStats, 0)
        for rows.Next() {
            var ps model.ProxyStats
            var ts string
            if err := rows.Scan(&ps.ID, &ps.ProxyID, &ts, &ps.BytesIn, &ps.BytesOut, &ps.ConnCount); err != nil {
                writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
                return
            }
            if t, err := time.Parse(time.RFC3339, ts); err == nil {
                ps.Timestamp = t
            }
            out = append(out, ps)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(out)
    default:
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
    }
}
