package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func (s *Server) handleGetSessions(w http.ResponseWriter, r *http.Request, claims jwt.MapClaims) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	restaurantID, err := getIDFromPath("/api/admin/sessions/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid restaurant id", http.StatusBadRequest)
		return
	}

	role, _ := claims["role"].(string)
	requesterEmail, _ := claims["email"].(string)

	var rows *sql.Rows
	if role == "owner" {
		rows, err = s.store.DB.QueryContext(r.Context(), "SELECT id, admin_email, created_at, expires_at, revoked, ip, user_agent FROM refresh_tokens WHERE restaurant_id=$1 ORDER BY created_at DESC LIMIT 200", restaurantID)
	} else {
		rows, err = s.store.DB.QueryContext(r.Context(), "SELECT id, admin_email, created_at, expires_at, revoked, ip, user_agent FROM refresh_tokens WHERE restaurant_id=$1 AND admin_email=$2 ORDER BY created_at DESC LIMIT 200", restaurantID, requesterEmail)
	}
	if err != nil {
		http.Error(w, "failed to query sessions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Sess struct {
		ID        int        `json:"id"`
		Email     string     `json:"adminEmail"`
		CreatedAt time.Time  `json:"createdAt"`
		ExpiresAt *time.Time `json:"expiresAt,omitempty"`
		Revoked   bool       `json:"revoked"`
		IP        string     `json:"ip"`
		UserAgent string     `json:"userAgent"`
	}

	var out []Sess
	for rows.Next() {
		var srec Sess
		var expires sql.NullTime
		if err := rows.Scan(&srec.ID, &srec.Email, &srec.CreatedAt, &expires, &srec.Revoked, &srec.IP, &srec.UserAgent); err != nil {
			http.Error(w, "failed to read row", http.StatusInternalServerError)
			return
		}
		if expires.Valid {
			srec.ExpiresAt = &expires.Time
		}
		out = append(out, srec)
	}
	writeJSON(w, out)
}

func (s *Server) handleRevokeSession(w http.ResponseWriter, r *http.Request, claims jwt.MapClaims) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var p struct {
		SessionId int `json:"sessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.SessionId == 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	var rec struct {
		ID           int
		RestaurantID int
		Email        string
		Revoked      bool
	}
	row := s.store.DB.QueryRowContext(r.Context(), "SELECT id, restaurant_id, admin_email, revoked FROM refresh_tokens WHERE id=$1", p.SessionId)
	if err := row.Scan(&rec.ID, &rec.RestaurantID, &rec.Email, &rec.Revoked); err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	role, _ := claims["role"].(string)
	requesterEmail, _ := claims["email"].(string)

	if role != "owner" && requesterEmail != rec.Email {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := s.store.RevokeRefreshTokenByID(r.Context(), rec.ID); err != nil {
		http.Error(w, "failed to revoke", http.StatusInternalServerError)
		return
	}

	_ = insertAuditLog(r.Context(), s.store.DB, rec.RestaurantID, requesterEmail, "session_revoked", map[string]any{"revoked_session_id": rec.ID, "revoked_for": rec.Email}, r.RemoteAddr)

	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleRevokeAllOtherSessions(w http.ResponseWriter, r *http.Request, claims jwt.MapClaims) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	restaurantID := 0
	if v, ok := claims["restaurantId"].(float64); ok {
		restaurantID = int(v)
	}
	email, _ := claims["email"].(string)
	if restaurantID == 0 || email == "" {
		http.Error(w, "invalid token claims", http.StatusUnauthorized)
		return
	}

	cookie, err := r.Cookie(RefreshCookieName)
	var currentRaw string
	if err == nil {
		currentRaw = cookie.Value
	}

	var currentID int
	if currentRaw != "" {
		found, err := s.store.FindRefreshToken(r.Context(), currentRaw)
		if err == nil {
			currentID = found.ID
		}
	}

	var res sql.Result
	if currentID > 0 {
		res, err = s.store.DB.ExecContext(r.Context(), "UPDATE refresh_tokens SET revoked=true WHERE restaurant_id=$1 AND admin_email=$2 AND id<>$3 AND revoked=false", restaurantID, email, currentID)
	} else {
		res, err = s.store.DB.ExecContext(r.Context(), "UPDATE refresh_tokens SET revoked=true WHERE restaurant_id=$1 AND admin_email=$2 AND revoked=false", restaurantID, email)
	}
	if err != nil {
		http.Error(w, "failed to revoke sessions", http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()

	_ = insertAuditLog(r.Context(), s.store.DB, restaurantID, email, "session_revoke_other", map[string]any{"revoked_count": affected}, r.RemoteAddr)

	writeJSON(w, map[string]any{"ok": true, "revoked": affected})
}
