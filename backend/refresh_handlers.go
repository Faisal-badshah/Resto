package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const RefreshCookieName = "refresh_token"

var (
	RefreshTokenDuration = 30 * 24 * time.Hour
	AccessTokenTTL       = 15 * time.Minute
)

func cookieIsSecure() bool {
	if os.Getenv("ALLOW_INSECURE_COOKIES") == "1" {
		return false
	}
	return os.Getenv("ENV") == "production"
}

func setRefreshCookie(w http.ResponseWriter, token string, expires time.Time) {
	secure := cookieIsSecure()
	c := &http.Cookie{
		Name:     RefreshCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Expires:  expires,
	}
	http.SetCookie(w, c)
}

func clearRefreshCookie(w http.ResponseWriter) {
	c := &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieIsSecure(),
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
	http.SetCookie(w, c)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "only GET/POST allowed", http.StatusMethodNotAllowed)
		return
	}
	c, err := r.Cookie(RefreshCookieName)
	if err != nil {
		http.Error(w, "no refresh token", http.StatusUnauthorized)
		return
	}
	raw := c.Value
	found, err := s.store.FindRefreshToken(r.Context(), raw)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	if found.Revoked {
		http.Error(w, "token revoked", http.StatusUnauthorized)
		return
	}
	if found.ExpiresAt.Valid && time.Now().After(found.ExpiresAt.Time) {
		http.Error(w, "token expired", http.StatusUnauthorized)
		return
	}
	admin, err := s.store.GetAdminByEmail(r.Context(), found.RestaurantID, found.Email)
	if err != nil {
		http.Error(w, "admin not found", http.StatusUnauthorized)
		return
	}
	accessToken, err := createTokenWithTTL(admin, AccessTokenTTL)
	if err != nil {
		http.Error(w, "failed to create access token", http.StatusInternalServerError)
		return
	}
	newRaw, newID, err := s.store.RotateRefreshToken(r.Context(), raw, r.RemoteAddr, r.UserAgent(), RefreshTokenDuration)
	if err != nil {
		fmt.Println("refresh rotation failed:", err)
		writeJSON(w, map[string]any{"accessToken": accessToken, "role": admin.Role, "expiresIn": int(AccessTokenTTL.Seconds()), "currentSessionId": found.ID})
		return
	}
	setRefreshCookie(w, newRaw, time.Now().Add(RefreshTokenDuration))
	writeJSON(w, map[string]any{"accessToken": accessToken, "role": admin.Role, "expiresIn": int(AccessTokenTTL.Seconds()), "currentSessionId": newID})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	c, err := r.Cookie(RefreshCookieName)
	if err == nil {
		found, errFind := s.store.FindRefreshToken(r.Context(), c.Value)
		if errFind == nil {
			_ = s.store.RevokeRefreshTokenByID(r.Context(), found.ID)
			_ = insertAuditLog(r.Context(), s.store.DB, found.RestaurantID, found.Email, "session_revoked_by_user", map[string]any{"session_id": found.ID}, r.RemoteAddr)
		} else {
			_ = s.store.RevokeRefreshTokenByRaw(r.Context(), c.Value)
		}
	}
	clearRefreshCookie(w)
	writeJSON(w, map[string]any{"ok": true})
}

func createTokenWithTTL(admin AdminUser, ttl time.Duration) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}
	claims := jwt.MapClaims{
		"restaurantId": admin.RestaurantID,
		"email":        admin.Email,
		"role":         admin.Role,
		"exp":          time.Now().Add(ttl).Unix(),
		"iat":          time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
