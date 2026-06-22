package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"github.com/asyou/server/internal/frp"
	"github.com/asyou/server/internal/model"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

var jwtKey = []byte("change-this-secret")

type Server struct {
	DB        *sql.DB
	FRP       *frp.Manager
	SSE       *SSEHub
	ACME      *ACMEConfig
	// ProxyStartPort is the first port in the range auto-assigned to proxies.
	// Must match the allow_ports range in frps config (default: 31000).
	ProxyStartPort int
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSONError(w, "email and password required", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
		return
	}
	res, err := s.DB.Exec(`INSERT INTO users (email, password_hash, display_name) VALUES (?, ?, ?)`, req.Email, string(hash), req.DisplayName)
	if err != nil {
		writeJSONError(w, "cannot create user", "INTERNAL", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	u := model.User{ID: id, Email: req.Email, DisplayName: req.DisplayName}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	var id int64
	var passwordHash string
	err := s.DB.QueryRow(`SELECT id, password_hash FROM users WHERE email = ?`, req.Email).Scan(&id, &passwordHash)
	if err == sql.ErrNoRows {
		writeJSONError(w, "invalid credentials", "UNAUTHORIZED", http.StatusUnauthorized)
		return
	} else if err != nil {
		writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		writeJSONError(w, "invalid credentials", "UNAUTHORIZED", http.StatusUnauthorized)
		return
	}
	// create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": id,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString(jwtKey)
	if err != nil {
		writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"access_token": signed, "expires_in": 86400})
}

// validateJWT extracts and validates the Bearer token or X-Api-Key, returning the user ID.
func (s *Server) validateJWT(r *http.Request) (int64, error) {
	// X-Api-Key support
	apiKey := r.Header.Get("X-Api-Key")
	if apiKey != "" {
		return s.authenticateAPIKey(apiKey)
	}
	// Bearer JWT
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return 0, fmt.Errorf("missing or invalid authorization header")
	}
	tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if tokenStr == "" {
		return 0, fmt.Errorf("empty token")
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid or expired token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}
	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid token subject")
	}
	return int64(subFloat), nil
}

// authenticateAPIKey looks up an API key by iterating all non-revoked keys and checking bcrypt.
func (s *Server) authenticateAPIKey(raw string) (int64, error) {
	rows, err := s.DB.Query(`SELECT id, user_id, token_hash FROM api_keys WHERE revoked = 0`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var userID int64
		var hash string
		if err := rows.Scan(&id, &userID, &hash); err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil {
			return userID, nil
		}
	}
	return 0, fmt.Errorf("invalid api key")
}

// AuthMiddleware validates JWT Bearer token or X-Api-Key and injects user_id into request context.
func (s *Server) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := s.validateJWT(r)
		if err != nil {
			writeJSONError(w, "unauthorized: "+err.Error(), "UNAUTHORIZED", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// UserIDFromContext extracts the authenticated user ID from request context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

// UsersMeHandler returns the currently authenticated user.
func (s *Server) UsersMeHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "unauthorized", "UNAUTHORIZED", http.StatusUnauthorized)
		return
	}
	var u model.User
	var displayName sql.NullString
	err := s.DB.QueryRow(`SELECT id, email, display_name, role FROM users WHERE id = ?`, userID).
		Scan(&u.ID, &u.Email, &displayName, &u.Role)
	if err == sql.ErrNoRows {
		writeJSONError(w, "user not found", "NOT_FOUND", http.StatusNotFound)
		return
	} else if err != nil {
		writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
		return
	}
	if displayName.Valid {
		u.DisplayName = displayName.String
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

// writeJSONError sends a structured error response.
func writeJSONError(w http.ResponseWriter, message, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message, "code": code})
}
