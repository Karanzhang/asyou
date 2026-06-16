package handlers

import (
	"net/http"
)

// mustGetUserID extracts the authenticated user ID from the request context.
// Returns 0 if not authenticated (callers should check).
func mustGetUserID(r *http.Request) int64 {
	id, _ := UserIDFromContext(r.Context())
	return id
}

// isAdmin checks whether the authenticated user has the admin role.
func (s *Server) isAdmin(r *http.Request) bool {
	id := mustGetUserID(r)
	if id == 0 {
		return false
	}
	var role string
	err := s.DB.QueryRow(`SELECT role FROM users WHERE id = ?`, id).Scan(&role)
	if err != nil {
		return false
	}
	return role == "admin"
}
