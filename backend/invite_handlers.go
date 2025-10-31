package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v4"
)

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Server) handleInviteAdmin(w http.ResponseWriter, r *http.Request, claims jwt.MapClaims) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST", http.StatusMethodNotAllowed)
		return
	}
	role, _ := claims["role"].(string)
	if role != "owner" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	restIdFromToken := 0
	if v, ok := claims["restaurantId"].(float64); ok {
		restIdFromToken = int(v)
	}
	restaurantId, err := getIDFromPath("/api/admin/invite/", r.URL.Path)
	if err != nil || restaurantId == 0 {
		http.Error(w, "invalid restaurant id in path", http.StatusBadRequest)
		return
	}
	if restIdFromToken != 0 && restIdFromToken != restaurantId {
		http.Error(w, "token restaurant mismatch", http.StatusForbidden)
		return
	}

	var payload struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Email == "" || payload.Role == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if payload.Role != "chef" && payload.Role != "owner" {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	token, err := generateToken(32)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(72 * time.Hour)

	_, err = s.store.DB.ExecContext(r.Context(), `INSERT INTO admin_invitations (restaurant_id, email, role, token, created_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (restaurant_id, email) DO UPDATE SET token=EXCLUDED.token, created_at=EXCLUDED.created_at, expires_at=EXCLUDED.expires_at, role=EXCLUDED.role`,
		restaurantId, payload.Email, payload.Role, token, time.Now(), expires)
	if err != nil {
		http.Error(w, "failed to create invite", http.StatusInternalServerError)
		return
	}

	frontend := os.Getenv("FRONTEND_URL")
	if frontend == "" {
		frontend = "http://localhost:3000"
	}
	inviteURL := fmt.Sprintf("%s/invite/accept?token=%s&restaurantId=%d", frontend, token, restaurantId)
	subject := "You're invited to manage the restaurant"
	body := fmt.Sprintf("You have been invited as '%s' for restaurant ID %d.\n\nClick the link to accept and create your password (expires in 72 hours):\n\n%s", payload.Role, restaurantId, inviteURL)
	go func() {
		if err := sendEmail(payload.Email, subject, body); err != nil {
			fmt.Println("failed sending invite email:", err)
		}
	}()

	pl := map[string]any{"action": "invite_created", "invited_email": payload.Email, "role": payload.Role}
	_ = insertAuditLog(r.Context(), s.store.DB, restaurantId, claims["email"].(string), "invite_created", pl, r.RemoteAddr)

	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST", http.StatusMethodNotAllowed)
		return
	}
	var p struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Token == "" || p.Password == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	var inv struct {
		ID           int
		RestaurantID int
		Email        string
		Role         string
		Token        string
		ExpiresAt    sql.NullTime
		AcceptedAt   sql.NullTime
	}
	row := s.store.DB.QueryRowContext(r.Context(), "SELECT id, restaurant_id, email, role, token, expires_at, accepted_at FROM admin_invitations WHERE token=$1", p.Token)
	if err := row.Scan(&inv.ID, &inv.RestaurantID, &inv.Email, &inv.Role, &inv.Token, &inv.ExpiresAt, &inv.AcceptedAt); err != nil {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}
	if inv.AcceptedAt.Valid {
		http.Error(w, "invite already accepted", http.StatusBadRequest)
		return
	}
	if inv.ExpiresAt.Valid && time.Now().After(inv.ExpiresAt.Time) {
		http.Error(w, "invite expired", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}
	perms := []string{}
	permB, _ := json.Marshal(perms)
	_, err = s.store.DB.ExecContext(r.Context(), "INSERT INTO admins (restaurant_id, email, password_hash, role, permissions) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (restaurant_id, email) DO UPDATE SET password_hash=EXCLUDED.password_hash, role=EXCLUDED.role, permissions=EXCLUDED.permissions",
		inv.RestaurantID, inv.Email, string(hash), inv.Role, permB)
	if err != nil {
		http.Error(w, "failed to create admin", http.StatusInternalServerError)
		return
	}

	_, _ = s.store.DB.ExecContext(r.Context(), "UPDATE admin_invitations SET accepted_at=$1 WHERE id=$2", time.Now(), inv.ID)
	_ = insertAuditLog(r.Context(), s.store.DB, inv.RestaurantID, inv.Email, "invite_accepted", map[string]any{"invitation_id": inv.ID}, r.RemoteAddr)

	go func() {
		subject := "Your admin account is ready"
		body := fmt.Sprintf("Hello %s. Your admin account for restaurant %d is now active. You can login at %s/restaurant/%d/admin", inv.Email, inv.RestaurantID, os.Getenv("FRONTEND_URL"), inv.RestaurantID)
		_ = sendEmail(inv.Email, subject, body)
	}()

	writeJSON(w, map[string]any{"ok": true})
}

func insertAuditLog(ctx context.Context, db *sql.DB, restaurantID int, adminEmail string, action string, payload any, ip string) error {
	plb, _ := json.Marshal(payload)
	_, err := db.ExecContext(ctx, "INSERT INTO audit_log (restaurant_id, admin_email, action, payload, ip, created_at) VALUES ($1,$2,$3,$4,$5,$6)", restaurantID, adminEmail, action, plb, ip, time.Now())
	return err
}
