package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handlePasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST", http.StatusMethodNotAllowed)
		return
	}
	var p struct {
		RestaurantId int    `json:"restaurantId"`
		Email        string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.Email == "" || p.RestaurantId == 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	var exists bool
	row := s.store.DB.QueryRowContext(r.Context(), "SELECT EXISTS (SELECT 1 FROM admins WHERE restaurant_id=$1 AND email=$2)", p.RestaurantId, p.Email)
	_ = row.Scan(&exists)

	if exists {
		token, err := generateToken(32)
		if err != nil {
			fmt.Println("generate token failed:", err)
			writeJSON(w, map[string]any{"ok": true})
			return
		}
		expires := time.Now().Add(1 * time.Hour)
		_, err = s.store.DB.ExecContext(r.Context(), "INSERT INTO password_resets (admin_email, token, created_at, expires_at) VALUES ($1,$2,$3,$4) ON CONFLICT (admin_email) DO UPDATE SET token=EXCLUDED.token, created_at=EXCLUDED.created_at, expires_at=EXCLUDED.expires_at, used_at=NULL",
			p.Email, token, time.Now(), expires)
		if err != nil {
			fmt.Println("failed inserting password reset:", err)
			writeJSON(w, map[string]any{"ok": true})
			return
		}

		frontend := os.Getenv("FRONTEND_URL")
		if frontend == "" {
			frontend = "http://localhost:3000"
		}
		resetURL := fmt.Sprintf("%s/password-reset/confirm?token=%s&restaurantId=%d", frontend, token, p.RestaurantId)
		subject := "Password reset for admin account"
		body := fmt.Sprintf("A request to reset the admin password was made. If you requested this, click the link to set a new password (expires in 1 hour):\n\n%s\n\nIf you did not request this, ignore this email.", resetURL)
		go func() {
			if err := sendEmail(p.Email, subject, body); err != nil {
				fmt.Println("password reset email failed:", err)
			}
		}()

		_ = insertAuditLog(r.Context(), s.store.DB, p.RestaurantId, p.Email, "password_reset_requested", map[string]any{"email": p.Email}, r.RemoteAddr)
	}

	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handlePasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
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
	var pr struct {
		ID        int
		Email     string
		ExpiresAt sql.NullTime
		UsedAt    sql.NullTime
	}
	row := s.store.DB.QueryRowContext(r.Context(), "SELECT id, admin_email, expires_at, used_at FROM password_resets WHERE token=$1", p.Token)
	if err := row.Scan(&pr.ID, &pr.Email, &pr.ExpiresAt, &pr.UsedAt); err != nil {
		http.Error(w, "invalid or expired token", http.StatusBadRequest)
		return
	}
	if pr.UsedAt.Valid {
		http.Error(w, "token already used", http.StatusBadRequest)
		return
	}
	if pr.ExpiresAt.Valid && time.Now().After(pr.ExpiresAt.Time) {
		http.Error(w, "token expired", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}
	res, err := s.store.DB.ExecContext(r.Context(), "UPDATE admins SET password_hash=$1 WHERE email=$2", string(hash), pr.Email)
	if err != nil {
		http.Error(w, "failed to update password", http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "account not found", http.StatusBadRequest)
		return
	}

	_, _ = s.store.DB.ExecContext(r.Context(), "UPDATE password_resets SET used_at=$1 WHERE id=$2", time.Now(), pr.ID)
	var restaurantID int
	row2 := s.store.DB.QueryRowContext(r.Context(), "SELECT restaurant_id FROM admins WHERE email=$1 LIMIT 1", pr.Email)
	_ = row2.Scan(&restaurantID)

	_ = insertAuditLog(r.Context(), s.store.DB, restaurantID, pr.Email, "password_reset_confirmed", map[string]any{"token_id": pr.ID}, r.RemoteAddr)
	go func() {
		subject := "Password successfully reset"
		body := "Your admin password has been successfully reset. If you did not perform this action, please contact support."
		_ = sendEmail(pr.Email, subject, body)
	}()

	writeJSON(w, map[string]any{"ok": true})
}
