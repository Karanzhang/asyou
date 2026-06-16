package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/asyou/server/internal/model"
)

// CertsListHandler handles GET /api/v1/certs
func (s *Server) CertsListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}
	userID := mustGetUserID(r)
	var rows *sql.Rows
	var err error
	if s.isAdmin(r) {
		rows, err = s.DB.Query(`SELECT id, user_id, proxy_id, domain, issuer, expires_at, auto_renew, created_at, updated_at FROM certificates ORDER BY expires_at`)
	} else {
		rows, err = s.DB.Query(`SELECT id, user_id, proxy_id, domain, issuer, expires_at, auto_renew, created_at, updated_at FROM certificates WHERE user_id = ? ORDER BY expires_at`, userID)
	}
	if err != nil {
		writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	out := make([]model.Certificate, 0)
	for rows.Next() {
		var c model.Certificate
		var expiresStr, createdAtStr, updatedAtStr string
		if err := rows.Scan(&c.ID, &c.UserID, &c.ProxyID, &c.Domain, &c.Issuer, &expiresStr, &c.AutoRenew, &createdAtStr, &updatedAtStr); err != nil {
			writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
			return
		}
		if t, err := time.Parse(time.RFC3339, expiresStr); err == nil {
			c.ExpiresAt = t
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			c.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			c.UpdatedAt = t
		}
		out = append(out, c)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// CertsItemHandler handles GET/DELETE /api/v1/certs/{id}
func (s *Server) CertsItemHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v1/certs/")
	idStr = strings.TrimSuffix(idStr, "/")

	// Check for provision action
	if idStr == "provision" || strings.HasSuffix(idStr, "/provision") {
		s.CertsProvisionHandler(w, r)
		return
	}

	idStr = path.Clean(idStr)
	id, err := strconv.ParseInt(strings.Trim(idStr, "/"), 10, 64)
	if err != nil {
		writeJSONError(w, "invalid id", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	userID := mustGetUserID(r)

	switch r.Method {
	case http.MethodGet:
		var c model.Certificate
		var expiresStr, createdAtStr, updatedAtStr string

		if s.isAdmin(r) {
			err = s.DB.QueryRow(`SELECT id, user_id, proxy_id, domain, issuer, expires_at, auto_renew, created_at, updated_at FROM certificates WHERE id = ?`, id).
				Scan(&c.ID, &c.UserID, &c.ProxyID, &c.Domain, &c.Issuer, &expiresStr, &c.AutoRenew, &createdAtStr, &updatedAtStr)
		} else {
			err = s.DB.QueryRow(`SELECT id, user_id, proxy_id, domain, issuer, expires_at, auto_renew, created_at, updated_at FROM certificates WHERE id = ? AND user_id = ?`, id, userID).
				Scan(&c.ID, &c.UserID, &c.ProxyID, &c.Domain, &c.Issuer, &expiresStr, &c.AutoRenew, &createdAtStr, &updatedAtStr)
		}
		if err == sql.ErrNoRows {
			writeJSONError(w, "not found", "NOT_FOUND", http.StatusNotFound)
			return
		} else if err != nil {
			writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
			return
		}
		if t, err := time.Parse(time.RFC3339, expiresStr); err == nil {
			c.ExpiresAt = t
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			c.CreatedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			c.UpdatedAt = t
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c)
	case http.MethodDelete:
		var res sql.Result
		if s.isAdmin(r) {
			res, err = s.DB.Exec(`DELETE FROM certificates WHERE id = ?`, id)
		} else {
			res, err = s.DB.Exec(`DELETE FROM certificates WHERE id = ? AND user_id = ?`, id, userID)
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

// CertsProvisionHandler provisions a certificate for a proxy's custom domain.
// POST /api/v1/certs/provision with body: { "proxy_id": 1, "domain": "example.com" }
func (s *Server) CertsProvisionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ProxyID int64  `json:"proxy_id"`
		Domain  string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	if req.Domain == "" {
		writeJSONError(w, "domain required", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	userID := mustGetUserID(r)

	// Verify proxy ownership
	var ownerID int64
	if s.isAdmin(r) {
		err := s.DB.QueryRow(`SELECT user_id FROM proxies WHERE id = ?`, req.ProxyID).Scan(&ownerID)
		if err == sql.ErrNoRows {
			writeJSONError(w, "proxy not found", "NOT_FOUND", http.StatusNotFound)
			return
		} else if err != nil {
			writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
			return
		}
	} else {
		err := s.DB.QueryRow(`SELECT user_id FROM proxies WHERE id = ? AND user_id = ?`, req.ProxyID, userID).Scan(&ownerID)
		if err == sql.ErrNoRows {
			writeJSONError(w, "proxy not found", "NOT_FOUND", http.StatusNotFound)
			return
		} else if err != nil {
			writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
			return
		}
	}

	// Provision certificate via ACME
	acmeClient := NewACMEClient(s.ACME)
	certPEM, keyPEM, expiresAt, err := acmeClient.ProvisionCert(req.Domain)
	if err != nil {
		writeJSONError(w, "provision failed: "+err.Error(), "ACME_ERROR", http.StatusInternalServerError)
		return
	}

	// Store certificate
	_, err = s.DB.Exec(`INSERT INTO certificates (user_id, proxy_id, domain, cert_pem, key_pem, issuer, expires_at, auto_renew) VALUES (?, ?, ?, ?, ?, ?, ?, 1)`,
		ownerID, req.ProxyID, req.Domain, certPEM, keyPEM, "letsencrypt", expiresAt.Format(time.RFC3339))
	if err != nil {
		writeJSONError(w, "store cert failed", "INTERNAL", http.StatusInternalServerError)
		return
	}

	// Enable TLS on the proxy
	s.DB.Exec(`UPDATE proxies SET enable_tls = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, req.ProxyID)

	s.RecordAudit(&userID, "cert_provision", "certificate", &req.ProxyID, "domain="+req.Domain, r.RemoteAddr)
	s.broadcastProxyUpdate(req.ProxyID, "", "")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"domain":     req.Domain,
		"expires_at": expiresAt.Format(time.RFC3339),
		"issuer":     "letsencrypt",
	})
}
