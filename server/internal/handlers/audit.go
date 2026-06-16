package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "strconv"
    "time"

    "github.com/asyou/server/internal/model"
)

// AuditListHandler handles GET /api/v1/audit-logs
func (s *Server) AuditListHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
        return
    }
    q := r.URL.Query()
    limit := 100
    if l := q.Get("limit"); l != "" {
        if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
            limit = v
        }
    }
    rows, err := s.DB.Query(`SELECT id, actor_user_id, action_type, resource_type, resource_id, detail, ip, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
    if err != nil {
        writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    out := make([]model.AuditLog, 0)
    for rows.Next() {
        var al model.AuditLog
        var actorID sql.NullInt64
        var resourceID sql.NullInt64
        var detail sql.NullString
        var ip sql.NullString
        var createdStr string
        if err := rows.Scan(&al.ID, &actorID, &al.ActionType, &al.ResourceType, &resourceID, &detail, &ip, &createdStr); err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        if actorID.Valid {
            al.ActorUserID = &actorID.Int64
        }
        if resourceID.Valid {
            al.ResourceID = &resourceID.Int64
        }
        if detail.Valid {
            al.Detail = &detail.String
        }
        if ip.Valid {
            al.IP = &ip.String
        }
        if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
            al.CreatedAt = t
        }
        out = append(out, al)
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(out)
}

// RecordAudit inserts an audit log entry.
func (s *Server) RecordAudit(actorUserID *int64, actionType, resourceType string, resourceID *int64, detail, ip string) {
    var aID interface{}
    if actorUserID != nil {
        aID = *actorUserID
    }
    var rID interface{}
    if resourceID != nil {
        rID = *resourceID
    }
    det := sql.NullString{String: detail, Valid: detail != ""}
    ipStr := sql.NullString{String: ip, Valid: ip != ""}
    _, _ = s.DB.Exec(`INSERT INTO audit_logs (actor_user_id, action_type, resource_type, resource_id, detail, ip) VALUES (?, ?, ?, ?, ?, ?)`,
        aID, actionType, resourceType, rID, det, ipStr)
}
