package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
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
	SMTPHost  string
	SMTPPort  int
	SMTPUser  string
	SMTPPass  string
	SMTPFrom  string
	PublicURL string // public-facing URL for reset links
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
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			writeJSONError(w, "email already registered", "CONFLICT", http.StatusConflict)
		} else {
			writeJSONError(w, "cannot create user", "INTERNAL", http.StatusInternalServerError)
		}
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

// generateResetToken creates a cryptographically random token.
func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ForgotPasswordHandler sends a password reset email.
func (s *Server) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	// Always return 200 regardless of whether email exists (prevent enumeration)
	defer func() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "If the email exists, a reset link has been sent."})
	}()

	var userID int64
	err := s.DB.QueryRow(`SELECT id FROM users WHERE email = ?`, req.Email).Scan(&userID)
	if err != nil {
		return // email not found — silently return
	}

	token, err := generateResetToken()
	if err != nil {
		return
	}

	expiresAt := time.Now().Add(1 * time.Hour)
	if _, err := s.DB.Exec(`INSERT INTO password_resets (user_id, token, expires_at) VALUES (?, ?, ?)`, userID, token, expiresAt); err != nil {
		return
	}

	// Build reset link
	resetURL := s.PublicURL + "/reset-password?token=" + token
	body := fmt.Sprintf(`Hello,

You requested a password reset for your asyou account.

Click the link below to reset your password (valid for 1 hour):
%s

If you did not request this, please ignore this email.

— asyou`, resetURL)

	s.sendEmail(req.Email, "asyou Password Reset", body)
}

// ResetPasswordHandler resets the password using a valid token.
func (s *Server) ResetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method not allowed", "METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "bad request", "BAD_REQUEST", http.StatusBadRequest)
		return
	}
	if req.Token == "" || req.Password == "" {
		writeJSONError(w, "token and password required", "BAD_REQUEST", http.StatusBadRequest)
		return
	}

	var userID int64
	var expiresAt time.Time
	err := s.DB.QueryRow(`SELECT user_id, expires_at FROM password_resets WHERE token = ? AND used = 0`, req.Token).Scan(&userID, &expiresAt)
	if err != nil {
		writeJSONError(w, "invalid or expired token", "INVALID_TOKEN", http.StatusBadRequest)
		return
	}
	if time.Now().After(expiresAt) {
		writeJSONError(w, "token has expired", "EXPIRED_TOKEN", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
		return
	}

	if _, err := s.DB.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, string(hash), userID); err != nil {
		writeJSONError(w, "internal error", "INTERNAL", http.StatusInternalServerError)
		return
	}
	// Mark token as used
	s.DB.Exec(`UPDATE password_resets SET used = 1 WHERE token = ?`, req.Token)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Password has been reset successfully."})
}

// sendEmail sends an email via SMTP.
func (s *Server) sendEmail(to, subject, body string) {
	if s.SMTPHost == "" {
		return // SMTP not configured
	}
	port := s.SMTPPort
	if port == 0 {
		port = 587
	}
	auth := smtp.PlainAuth("", s.SMTPUser, s.SMTPPass, s.SMTPHost)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", s.SMTPFrom, to, subject, body))
	addr := fmt.Sprintf("%s:%d", s.SMTPHost, port)
	if err := smtp.SendMail(addr, auth, s.SMTPFrom, []string{to}, msg); err != nil {
		// Log but don't expose to client
		fmt.Printf("[smtp] send failed: %v\n", err)
	}
}

// writeJSONError sends a structured error response.
func writeJSONError(w http.ResponseWriter, message, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message, "code": code})
}
