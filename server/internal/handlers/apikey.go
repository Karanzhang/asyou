package handlers

import (
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "path"
    "strconv"
    "strings"
    "time"

    "github.com/asyou/server/internal/model"
    "golang.org/x/crypto/bcrypt"
)

// ApiKeysListCreateHandler handles GET/POST /api/v1/api-keys
func (s *Server) ApiKeysListCreateHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := UserIDFromContext(r.Context())
    if !ok {
        writeJSONError(w, "unauthorized", "UNAUTHORIZED", http.StatusUnauthorized)
        return
    }
    switch r.Method {
    case http.MethodGet:
        rows, err := s.DB.Query(`SELECT id, user_id, name, scopes, revoked, created_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`, userID)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        defer rows.Close()
        out := make([]model.ApiKey, 0)
        for rows.Next() {
            var ak model.ApiKey
            var name sql.NullString
            var scopes sql.NullString
            var createdStr string
            if err := rows.Scan(&ak.ID, &ak.UserID, &name, &scopes, &ak.Revoked, &createdStr); err != nil {
                writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
                return
            }
            if name.Valid {
                ak.Name = &name.String
            }
            if scopes.Valid {
                ak.Scopes = &scopes.String
            }
            if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
                ak.CreatedAt = t
            }
            out = append(out, ak)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(out)
    case http.MethodPost:
        var req struct {
            Name   *string `json:"name"`
            Scopes *string `json:"scopes"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
            return
        }
        // generate random token
        buf := make([]byte, 32)
        if _, err := rand.Read(buf); err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        rawToken := "asyou_" + hex.EncodeToString(buf)
        hash, err := bcrypt.GenerateFromPassword([]byte(rawToken), bcrypt.DefaultCost)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        var name interface{}
        if req.Name != nil {
            name = *req.Name
        }
        _, err = s.DB.Exec(`INSERT INTO api_keys (user_id, name, token_hash, scopes) VALUES (?, ?, ?, ?)`, userID, name, string(hash), req.Scopes)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        s.RecordAudit(&userID, "api_key_create", "api_key", nil, "", r.RemoteAddr)
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]string{"token": rawToken})
    default:
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
    }
}

// ApiKeyItemHandler handles DELETE /api/v1/api-keys/{id} (revoke)
func (s *Server) ApiKeyItemHandler(w http.ResponseWriter, r *http.Request) {
    userID, ok := UserIDFromContext(r.Context())
    if !ok {
        writeJSONError(w, "unauthorized", "UNAUTHORIZED", http.StatusUnauthorized)
        return
    }
    idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/api-keys/")
    idStr = path.Clean(idStr)
    id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
    if err != nil {
        writeJSONError(w, "invalid id", "BAD_REQUEST", http.StatusBadRequest)
        return
    }
    switch r.Method {
    case http.MethodDelete:
        res, err := s.DB.Exec(`UPDATE api_keys SET revoked = 1 WHERE id = ? AND user_id = ?`, id, userID)
        if err != nil {
            writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
            return
        }
        affected, _ := res.RowsAffected()
        if affected == 0 {
            writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
            return
        }
        s.RecordAudit(&userID, "api_key_revoke", "api_key", &id, "", r.RemoteAddr)
        w.WriteHeader(http.StatusNoContent)
    default:
        writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
    }
}
