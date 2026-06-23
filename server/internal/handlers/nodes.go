package handlers

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "path"
    "strconv"
    "strings"
    "time"

    "github.com/asyou/core/frps"
    "github.com/asyou/server/internal/model"
)

type nodeCreateRequest struct {
    Name           string  `json:"name"`
    Host           string  `json:"host"`
    ApiPort        *int    `json:"api_port,omitempty"`
    BindPort       *int    `json:"bind_port,omitempty"`
    TlsEnabled     *bool   `json:"tls_enabled,omitempty"`
    AuthToken      *string `json:"auth_token,omitempty"`
    DashboardPort  *int    `json:"dashboard_port,omitempty"`
    DashboardUser  *string `json:"dashboard_user,omitempty"`
    DashboardPwd   *string `json:"dashboard_pwd,omitempty"`
    PortRangeStart *int    `json:"port_range_start,omitempty"`
    PortRangeEnd   *int    `json:"port_range_end,omitempty"`
    Region         *string `json:"region,omitempty"`
    Country        *string `json:"country,omitempty"`
    City           *string `json:"city,omitempty"`
    Latitude       *float64 `json:"latitude,omitempty"`
    Longitude      *float64 `json:"longitude,omitempty"`
    MaxConnections *int    `json:"max_connections,omitempty"`
    Weight         *float64 `json:"weight,omitempty"`
    SubdomainHost  *string `json:"subdomain_host,omitempty"`
}

// NodesListCreateHandler handles GET /nodes and POST /nodes
func (s *Server) NodesListCreateHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case http.MethodGet:
        rows, err := s.DB.Query(`SELECT id, name, host, api_port, bind_port, tls_enabled, auth_token, dashboard_port, dashboard_user, frp_version, region, country, city, latitude, longitude, max_connections, weight, subdomain_host, is_active, last_heartbeat, created_at, updated_at FROM nodes`)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        defer rows.Close()
        list := make([]model.Node, 0)
        for rows.Next() {
            var n model.Node
            var apiPort, bindPort, tlsEnabled, maxConn, isActive, dashPort sql.NullInt64
            var region, country, city, frpVer, lastHeartbeat, createdAt, updatedAt, dashUser, subdomainHost, authToken sql.NullString
            var lat, lng, weight sql.NullFloat64
            if err := rows.Scan(&n.ID, &n.Name, &n.Host, &apiPort, &bindPort, &tlsEnabled, &authToken, &dashPort, &dashUser, &frpVer, &region, &country, &city, &lat, &lng, &maxConn, &weight, &subdomainHost, &isActive, &lastHeartbeat, &createdAt, &updatedAt); err != nil {
                writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
                return
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
            if frpVer.Valid {
                n.FrpVersion = frpVer.String
            }
            if authToken.Valid {
                n.AuthToken = authToken.String
            }
            if region.Valid {
                n.Region = region.String
            }
            if country.Valid {
                n.Country = country.String
            }
            if city.Valid {
                n.City = city.String
            }
            if lat.Valid {
                n.Latitude = lat.Float64
            }
            if lng.Valid {
                n.Longitude = lng.Float64
            }
            if maxConn.Valid {
                n.MaxConnections = int(maxConn.Int64)
            }
            if weight.Valid {
                n.Weight = weight.Float64
            }
            if isActive.Valid {
                n.IsActive = isActive.Int64 != 0
            }
            if subdomainHost.Valid {
                n.SubdomainHost = subdomainHost.String
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
            list = append(list, n)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(list)
    case http.MethodPost:
        if !s.isAdmin(r) {
            writeJSONError(w, "admin access required", "FORBIDDEN", http.StatusForbidden)
            return
        }
        var req nodeCreateRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        if req.Name == "" || req.Host == "" {
            writeJSONError(w, "name and host required", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        ap := sql.NullInt64{}
        if req.ApiPort != nil {
            ap = sql.NullInt64{Int64: int64(*req.ApiPort), Valid: true}
        }
        bp := sql.NullInt64{}
        if req.BindPort != nil {
            bp = sql.NullInt64{Int64: int64(*req.BindPort), Valid: true}
        }
        tls := 1
        if req.TlsEnabled != nil && !*req.TlsEnabled {
            tls = 0
        }
        geoRegion := sql.NullString{String: "", Valid: true}
        if req.Region != nil {
            geoRegion = sql.NullString{String: *req.Region, Valid: true}
        }
        geoCountry := sql.NullString{String: "", Valid: true}
        if req.Country != nil {
            geoCountry = sql.NullString{String: *req.Country, Valid: true}
        }
        geoCity := sql.NullString{String: "", Valid: true}
        if req.City != nil {
            geoCity = sql.NullString{String: *req.City, Valid: true}
        }
        lat := 0.0
        if req.Latitude != nil {
            lat = *req.Latitude
        }
        lng := 0.0
        if req.Longitude != nil {
            lng = *req.Longitude
        }
        maxCon := 100
        if req.MaxConnections != nil {
            maxCon = *req.MaxConnections
        }
        weight := 1.0
        if req.Weight != nil {
            weight = *req.Weight
        }
        dp := sql.NullInt64{}
        if req.DashboardPort != nil {
            dp = sql.NullInt64{Int64: int64(*req.DashboardPort), Valid: true}
        }
        du := sql.NullString{}
        if req.DashboardUser != nil {
            du = sql.NullString{String: *req.DashboardUser, Valid: true}
        }
        dpwd := sql.NullString{}
        if req.DashboardPwd != nil {
            dpwd = sql.NullString{String: *req.DashboardPwd, Valid: true}
        }
        prStart := sql.NullInt64{}
        prEnd := sql.NullInt64{}
        if req.PortRangeStart != nil {
            prStart = sql.NullInt64{Int64: int64(*req.PortRangeStart), Valid: true}
        }
        if req.PortRangeEnd != nil {
            prEnd = sql.NullInt64{Int64: int64(*req.PortRangeEnd), Valid: true}
        }
        if !prStart.Valid && req.BindPort != nil {
            s, e := detectFrpsPortRange(*req.BindPort)
            if e > 0 {
                prStart = sql.NullInt64{Int64: int64(s), Valid: true}
                prEnd = sql.NullInt64{Int64: int64(e), Valid: true}
            }
        }
        sh := sql.NullString{String: "", Valid: true}
        if req.SubdomainHost != nil {
            sh = sql.NullString{String: *req.SubdomainHost, Valid: true}
        }
        _, err := s.DB.Exec(`INSERT INTO nodes (name, host, api_port, bind_port, tls_enabled, auth_token, dashboard_port, dashboard_user, dashboard_pwd, port_range_start, port_range_end, region, country, city, latitude, longitude, max_connections, weight, subdomain_host) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
            req.Name, req.Host, ap, bp, tls, req.AuthToken, dp, du, dpwd, prStart, prEnd, geoRegion, geoCountry, geoCity, lat, lng, maxCon, weight, sh)
        if err != nil {
            writeJSONError(w, "cannot create node", "INTERNAL", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusCreated)
    default:
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
    }
}

// NodeItemHandler handles GET/PUT/DELETE and heartbeat under /nodes/{id}
func (s *Server) NodeItemHandler(w http.ResponseWriter, r *http.Request) {
    // path is /api/v1/nodes/{id} or /api/v1/nodes/{id}/heartbeat
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/nodes/")
    if idStr == "" {
        writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    // node-authenticated proxy stats ingestion: /api/v1/nodes/{id}/proxies/{proxy_id}/stats
    if strings.Contains(idStr, "/proxies/") && strings.HasSuffix(idStr, "/stats") {
        // clean and delegate
        idStr = path.Clean(idStr)
        parts := strings.Split(strings.Trim(idStr, "/"), "/")
        if len(parts) >= 3 && parts[1] == "proxies" {
            nodeID := parts[0]
            proxyID := parts[2]
            s.NodeProxyStatsHandler(w, r, nodeID, proxyID)
            return
        }
    }
    if strings.HasSuffix(idStr, "/heartbeat") {
        idStr = strings.TrimSuffix(idStr, "/heartbeat")
        idStr = strings.TrimSuffix(idStr, "/")
        s.NodeHeartbeatHandler(w, r, idStr)
        return
    }
    if strings.HasSuffix(idStr, "/status") {
        idStr = strings.TrimSuffix(idStr, "/status")
        idStr = strings.TrimSuffix(idStr, "/")
        s.NodeStatusHandler(w, r, idStr)
        return
    }
    if idStr == "best" || strings.HasSuffix(idStr, "/best") {
        s.NodeBestHandler(w, r)
        return
    }
    // id only — JWT required for CRUD operations
    if _, err := s.validateJWT(r); err != nil {
        writeJSONError(w, "unauthorized: "+err.Error(), "UNAUTHORIZED", http.StatusUnauthorized)
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
        var n model.Node
        var apiPort sql.NullInt64
        var bindPort sql.NullInt64
        var tlsEnabled sql.NullInt64
        var region, country, city, frpVer, dashUser sql.NullString
        var lat, lng, weight sql.NullFloat64
        var maxConn, isActive, dashPort sql.NullInt64
        var lastHeartbeat sql.NullString
        var subdomainHost sql.NullString
        err := s.DB.QueryRow(`SELECT id, name, host, api_port, bind_port, tls_enabled, auth_token, dashboard_port, dashboard_user, frp_version, region, country, city, latitude, longitude, max_connections, weight, subdomain_host, is_active, last_heartbeat, created_at, updated_at FROM nodes WHERE id = ?`, id).
            Scan(&n.ID, &n.Name, &n.Host, &apiPort, &bindPort, &tlsEnabled, &n.AuthToken, &dashPort, &dashUser, &frpVer, &region, &country, &city, &lat, &lng, &maxConn, &weight, &subdomainHost, &isActive, &lastHeartbeat, &n.CreatedAt, &n.UpdatedAt)
        if err == sql.ErrNoRows {
            writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
            return
        } else if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
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
        if frpVer.Valid {
            n.FrpVersion = frpVer.String
        }
        if dashPort.Valid {
            n.DashboardPort = int(dashPort.Int64)
        }
        if dashUser.Valid {
            n.DashboardUser = dashUser.String
        }
        if region.Valid { n.Region = region.String }
        if country.Valid { n.Country = country.String }
        if city.Valid { n.City = city.String }
        if lat.Valid { n.Latitude = lat.Float64 }
        if lng.Valid { n.Longitude = lng.Float64 }
        if maxConn.Valid { n.MaxConnections = int(maxConn.Int64) }
        if weight.Valid { n.Weight = weight.Float64 }
        if isActive.Valid { n.IsActive = isActive.Int64 != 0 }
        if subdomainHost.Valid { n.SubdomainHost = subdomainHost.String }
        if lastHeartbeat.Valid {
            if t, err := time.Parse(time.RFC3339, lastHeartbeat.String); err == nil {
                n.LastHeartbeat = t
            }
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(n)
    case http.MethodPut:
        if !s.isAdmin(r) {
            writeJSONError(w, "admin access required", "FORBIDDEN", http.StatusForbidden)
            return
        }
        var upd struct {
            Name           *string `json:"name,omitempty"`
            Host           *string `json:"host,omitempty"`
            ApiPort        *int    `json:"api_port,omitempty"`
            BindPort       *int    `json:"bind_port,omitempty"`
            TlsEnabled     *bool   `json:"tls_enabled,omitempty"`
            AuthToken      *string `json:"auth_token,omitempty"`
            DashboardPort  *int    `json:"dashboard_port,omitempty"`
            DashboardUser  *string `json:"dashboard_user,omitempty"`
            DashboardPwd   *string `json:"dashboard_pwd,omitempty"`
            SubdomainHost  *string `json:"subdomain_host,omitempty"`
        }
        if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
            writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        var set []string
        var args []interface{}
        if upd.Name != nil {
            set = append(set, "name = ?")
            args = append(args, *upd.Name)
        }
        if upd.Host != nil {
            set = append(set, "host = ?")
            args = append(args, *upd.Host)
        }
        if upd.ApiPort != nil {
            set = append(set, "api_port = ?")
            args = append(args, *upd.ApiPort)
        }
        if upd.BindPort != nil {
            set = append(set, "bind_port = ?")
            args = append(args, *upd.BindPort)
        }
        if upd.TlsEnabled != nil {
            v := 0
            if *upd.TlsEnabled {
                v = 1
            }
            set = append(set, "tls_enabled = ?")
            args = append(args, v)
        }
        if upd.AuthToken != nil {
            set = append(set, "auth_token = ?")
            args = append(args, *upd.AuthToken)
        }
        if upd.DashboardPort != nil {
            set = append(set, "dashboard_port = ?")
            args = append(args, *upd.DashboardPort)
        }
        if upd.DashboardUser != nil {
            set = append(set, "dashboard_user = ?")
            args = append(args, *upd.DashboardUser)
        }
        if upd.DashboardPwd != nil {
            set = append(set, "dashboard_pwd = ?")
            args = append(args, *upd.DashboardPwd)
        }
        if upd.SubdomainHost != nil {
            set = append(set, "subdomain_host = ?")
            args = append(args, *upd.SubdomainHost)
        }
        if len(set) == 0 {
            writeJSONError(w, "nothing to update", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        set = append(set, "updated_at = CURRENT_TIMESTAMP")
        query := "UPDATE nodes SET " + strings.Join(set, ", ") + " WHERE id = ?"
        args = append(args, id)
        _, err = s.DB.Exec(query, args...)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        // return updated node
        var n model.Node
        var apiPort, bindPort, tlsEnabled, dashPort sql.NullInt64
        var dashUser, lastHeartbeat, subdomainHost sql.NullString
        err = s.DB.QueryRow(`SELECT id, name, host, api_port, bind_port, tls_enabled, dashboard_port, dashboard_user, subdomain_host, last_heartbeat, created_at, updated_at FROM nodes WHERE id = ?`, id).
            Scan(&n.ID, &n.Name, &n.Host, &apiPort, &bindPort, &tlsEnabled, &dashPort, &dashUser, &subdomainHost, &lastHeartbeat, &n.CreatedAt, &n.UpdatedAt)
        if err == sql.ErrNoRows {
            writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
            return
        } else if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        if apiPort.Valid { n.ApiPort = int(apiPort.Int64) }
        if bindPort.Valid { n.BindPort = int(bindPort.Int64) }
        if tlsEnabled.Valid { n.TlsEnabled = tlsEnabled.Int64 != 0 }
        if subdomainHost.Valid { n.SubdomainHost = subdomainHost.String }
        if dashPort.Valid { n.DashboardPort = int(dashPort.Int64) }
        if dashUser.Valid { n.DashboardUser = dashUser.String }
        if lastHeartbeat.Valid {
            if t, err := time.Parse(time.RFC3339, lastHeartbeat.String); err == nil {
                n.LastHeartbeat = t
            }
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(n)
    case http.MethodDelete:
        if !s.isAdmin(r) {
            writeJSONError(w, "admin access required", "FORBIDDEN", http.StatusForbidden)
            return
        }
        _, err := s.DB.Exec(`DELETE FROM nodes WHERE id = ?`, id)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    default:
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
    }
}

// NodeStatusHandler returns live status from frps admin API for a node.
func (s *Server) NodeStatusHandler(w http.ResponseWriter, r *http.Request, idStr string) {
    id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }

    // Load node
    var n model.Node
    var nApiPort, bindPort, dashPort, tlsEnabled, maxConn, isActive sql.NullInt64
    var dashUser, dashPwd, frpVer, lastHb, region, country, city, subdomainHost sql.NullString
    var lat, lng, weight sql.NullFloat64
    var createdAt, updatedAt sql.NullString
    err = s.DB.QueryRow(`SELECT id, name, host, api_port, bind_port, tls_enabled, dashboard_port, dashboard_user, dashboard_pwd, frp_version, region, country, city, latitude, longitude, max_connections, weight, subdomain_host, is_active, last_heartbeat, created_at, updated_at FROM nodes WHERE id = ?`, id).
        Scan(&n.ID, &n.Name, &n.Host, &nApiPort, &bindPort, &tlsEnabled, &dashPort, &dashUser, &dashPwd, &frpVer, &region, &country, &city, &lat, &lng, &maxConn, &weight, &subdomainHost, &isActive, &lastHb, &createdAt, &updatedAt)
    if err == sql.ErrNoRows {
        writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
        return
    } else if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    if nApiPort.Valid { n.ApiPort = int(nApiPort.Int64) }
    if bindPort.Valid { n.BindPort = int(bindPort.Int64) }
    if tlsEnabled.Valid { n.TlsEnabled = tlsEnabled.Int64 != 0 }
    if dashPort.Valid { n.DashboardPort = int(dashPort.Int64) }
    if dashUser.Valid { n.DashboardUser = dashUser.String }
    if subdomainHost.Valid { n.SubdomainHost = subdomainHost.String }
    if frpVer.Valid { n.FrpVersion = frpVer.String }
    if region.Valid { n.Region = region.String }
    if country.Valid { n.Country = country.String }
    if city.Valid { n.City = city.String }
    if lat.Valid { n.Latitude = lat.Float64 }
    if lng.Valid { n.Longitude = lng.Float64 }
    if maxConn.Valid { n.MaxConnections = int(maxConn.Int64) }
    if weight.Valid { n.Weight = weight.Float64 }
    if subdomainHost.Valid { n.SubdomainHost = subdomainHost.String }
    if isActive.Valid { n.IsActive = isActive.Int64 != 0 }
    if lastHb.Valid {
        if t, err := time.Parse(time.RFC3339, lastHb.String); err == nil {
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

    // Build admin client and query frps
    apiPort := 7500
    if dashPort.Valid {
        apiPort = int(dashPort.Int64)
    } else if n.ApiPort > 0 {
        apiPort = n.ApiPort
    }
    user := "admin"
    if dashUser.Valid && dashUser.String != "" {
        user = dashUser.String
    }
    pwd := ""
    if dashPwd.Valid {
        pwd = dashPwd.String
    }

    client := frps.NewAdminClientWithAuth(n.Host, apiPort, user, pwd)

    serverInfo, err := client.GetServerInfo()
    if err != nil {
        writeJSONError(w, "cannot reach frps admin: "+err.Error(), "UNAVAILABLE", http.StatusServiceUnavailable)
        return
    }

    proxies, err := client.ListAllProxies()
    if err != nil {
        proxies = nil // non-fatal
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "server_info": serverInfo,
        "proxies":     proxies,
        "node_id":     n.ID,
        "node_name":   n.Name,
    })
}

// NodeHeartbeatHandler updates last_heartbeat and records node health metrics.
func (s *Server) NodeHeartbeatHandler(w http.ResponseWriter, r *http.Request, idStr string) {
    if r.Method != http.MethodPost {
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
        return
    }
    id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    // Parse optional health payload
    var health struct {
        Uptime      *int     `json:"uptime"`
        Connections *int     `json:"connections"`
        CPULoad     *float64 `json:"cpu_load"`
        MemoryUsage *float64 `json:"memory_usage"`
        Bandwidth   *float64 `json:"bandwidth"`
        LatencyMs   *int     `json:"latency_ms"`
        FrpVersion  *string `json:"frp_version"`
    }
    json.NewDecoder(r.Body).Decode(&health)

    // Store frp version if provided
    if health.FrpVersion != nil && *health.FrpVersion != "" {
        s.DB.Exec(`UPDATE nodes SET frp_version = ? WHERE id = ?`, *health.FrpVersion, id)
    }

    // Update heartbeat timestamp
    _, err = s.DB.Exec(`UPDATE nodes SET last_heartbeat = ? WHERE id = ?`, time.Now().Format(time.RFC3339), id)
    if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }

    // Record health snapshot
    if health.Connections != nil || health.CPULoad != nil || health.MemoryUsage != nil {
        conns := 0
        if health.Connections != nil { conns = *health.Connections }
        cpu := 0.0
        if health.CPULoad != nil { cpu = *health.CPULoad }
        mem := 0.0
        if health.MemoryUsage != nil { mem = *health.MemoryUsage }
        bw := 0.0
        if health.Bandwidth != nil { bw = *health.Bandwidth }
        lat := 0
        if health.LatencyMs != nil { lat = *health.LatencyMs }
        s.DB.Exec(`INSERT INTO node_health (node_id, latency_ms, current_connections, cpu_load, memory_usage, bandwidth_mbps) VALUES (?, ?, ?, ?, ?, ?)`,
            id, lat, conns, cpu, mem, bw)
    }

    w.WriteHeader(http.StatusNoContent)
}

// NodeProxyStatsHandler handles node-authenticated POST /nodes/{node_id}/proxies/{proxy_id}/stats
func (s *Server) NodeProxyStatsHandler(w http.ResponseWriter, r *http.Request, nodeIDStr, proxyIDStr string) {
    if r.Method != http.MethodPost {
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
        return
    }
    nodeID, err := strconv.ParseInt(strings.Trim(nodeIDStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid node id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    proxyID, err := strconv.ParseInt(strings.Trim(proxyIDStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid proxy id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    // Authenticate node via Authorization: Bearer <token> or X-Node-Token
    auth := r.Header.Get("Authorization")
    token := ""
    if strings.HasPrefix(auth, "Bearer ") {
        token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
    }
    if token == "" {
        token = r.Header.Get("X-Node-Token")
    }
    if token == "" {
        writeJSONError(w, "missing auth token", "UNAUTHORIZED", http.StatusUnauthorized)
        return
    }
    var storedToken sql.NullString
    err = s.DB.QueryRow(`SELECT auth_token FROM nodes WHERE id = ?`, nodeID).Scan(&storedToken)
    if err == sql.ErrNoRows {
        writeJSONError(w, "node not found", "NOT_FOUND", http.StatusNotFound)
        return
    } else if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    if !storedToken.Valid || storedToken.String == "" || storedToken.String != token {
        writeJSONError(w, "unauthorized", "UNAUTHORIZED", http.StatusUnauthorized)
        return
    }
    // verify proxy belongs to node
    var proxyNodeID sql.NullInt64
    err = s.DB.QueryRow(`SELECT node_id FROM proxies WHERE id = ?`, proxyID).Scan(&proxyNodeID)
    if err == sql.ErrNoRows {
        writeJSONError(w, "proxy not found", "NOT_FOUND", http.StatusNotFound)
        return
    } else if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    if !proxyNodeID.Valid || proxyNodeID.Int64 != nodeID {
        writeJSONError(w, "proxy not associated with node", "FORBIDDEN", http.StatusForbidden)
        return
    }
    // decode payload
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
    _, err = s.DB.Exec(`INSERT INTO proxy_stats (proxy_id, timestamp, bytes_in, bytes_out, conn_count) VALUES (?, ?, ?, ?, ?)`, proxyID, ts.Format(time.RFC3339), bi, bo, cc)
    if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
}

// NodeBestHandler returns the best node for scheduling. GET /api/v1/nodes/best
// Query params: region, lat, lng, max_distance
func (s *Server) NodeBestHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
        return
    }
    q := r.URL.Query()

    rows, err := s.DB.Query(`SELECT id, name, host, api_port, bind_port, tls_enabled, frp_version, region, country, city, latitude, longitude, max_connections, weight, is_active, last_heartbeat, created_at, updated_at FROM nodes WHERE is_active = 1`)
    if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    var nodes []model.Node
    for rows.Next() {
        var n model.Node
        var apiPort, bindPort, tlsEnabled, maxConn, isActive sql.NullInt64
        var region, country, city, frpVer, lastHeartbeat, createdAt, updatedAt sql.NullString
        var lat, lng, weight sql.NullFloat64
        rows.Scan(&n.ID, &n.Name, &n.Host, &apiPort, &bindPort, &tlsEnabled, &frpVer,
            &region, &country, &city, &lat, &lng, &maxConn, &weight, &isActive,
            &lastHeartbeat, &createdAt, &updatedAt)
        if apiPort.Valid { n.ApiPort = int(apiPort.Int64) }
        if bindPort.Valid { n.BindPort = int(bindPort.Int64) }
        if tlsEnabled.Valid { n.TlsEnabled = tlsEnabled.Int64 != 0 }
        if frpVer.Valid { n.FrpVersion = frpVer.String }
        if region.Valid { n.Region = region.String }
        if country.Valid { n.Country = country.String }
        if city.Valid { n.City = city.String }
        if lat.Valid { n.Latitude = lat.Float64 }
        if lng.Valid { n.Longitude = lng.Float64 }
        if maxConn.Valid { n.MaxConnections = int(maxConn.Int64) }
        if weight.Valid { n.Weight = weight.Float64 }
        if isActive.Valid { n.IsActive = isActive.Int64 != 0 }
        if lastHeartbeat.Valid {
            if t, err := time.Parse(time.RFC3339, lastHeartbeat.String); err == nil {
                n.LastHeartbeat = t
            }
        }
        nodes = append(nodes, n)
    }
    // subdomain_host not queried in this path (best-node scoring)

    scorer := &NodeScorer{
        PreferRegion: q.Get("region"),
        MaxDistance:  5000,
    }
    if lat := q.Get("lat"); lat != "" {
        fmt.Sscanf(lat, "%f", &scorer.PreferLat)
    }
    if lng := q.Get("lng"); lng != "" {
        fmt.Sscanf(lng, "%f", &scorer.PreferLng)
    }
    if d := q.Get("max_distance"); d != "" {
        fmt.Sscanf(d, "%f", &scorer.MaxDistance)
    }

    healthMap := s.LoadLatestHealth()
    best := scorer.SelectBest(nodes, healthMap)
    if best == nil {
        writeJSONError(w, "no available nodes", "NOT_FOUND", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(best)
}

// detectFrpsPortRange scans local frps config files to find allow_ports matching the bind_port.
func detectFrpsPortRange(bindPort int) (start, end int) {
    candidates := []string{
        fmt.Sprintf("/etc/frps.toml"),
        fmt.Sprintf("/etc/frps-%d.toml", bindPort),
        fmt.Sprintf("/etc/frps/frps.toml"),
        fmt.Sprintf("/opt/asyou/frps.toml"),
    }
    for _, path := range candidates {
        data, err := os.ReadFile(path)
        if err != nil {
            continue
        }
        content := string(data)
        // Match bind_port in this config
        if !strings.Contains(content, fmt.Sprintf("bind_port = %d", bindPort)) &&
            !strings.Contains(content, fmt.Sprintf("bind_port=%d", bindPort)) {
            continue
        }
        // Extract allow_ports value
        for _, line := range strings.Split(content, "\n") {
            line = strings.TrimSpace(line)
            if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
                continue
            }
            if strings.Contains(line, "allow_ports") {
                parts := strings.SplitN(line, "=", 2)
                if len(parts) < 2 {
                    continue
                }
                val := strings.TrimSpace(parts[1])
                val = strings.Trim(val, "\"'")
                // Parse "31000-31999" or "31000-31999,32000-32099"
                rangeStr := strings.Split(val, ",")[0]
                rangeStr = strings.TrimSpace(rangeStr)
                if n, _ := fmt.Sscanf(rangeStr, "%d-%d", &start, &end); n == 2 {
                    return start, end
                }
            }
        }
    }
    return 0, 0
}
