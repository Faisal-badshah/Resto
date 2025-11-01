package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var creds struct {
		RestaurantID int    `json:"restaurantId"`
		Email        string `json:"email"`
		Password     string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	
	admin, err := s.store.GetAdminByEmail(r.Context(), creds.RestaurantID, creds.Email)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(creds.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	
	// Create access token
	token, err := createTokenWithTTL(admin, AccessTokenTTL)
	if err != nil {
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}
	
	// Create refresh token
	refreshToken, sessionID, err := s.store.CreateRefreshToken(r.Context(), admin.RestaurantID, admin.Email, r.RemoteAddr, r.UserAgent(), RefreshTokenDuration)
	if err != nil {
		http.Error(w, "failed to create refresh token", http.StatusInternalServerError)
		return
	}
	
	setRefreshCookie(w, refreshToken, time.Now().Add(RefreshTokenDuration))
	
	writeJSON(w, map[string]interface{}{
		"token":            token,
		"role":             admin.Role,
		"expiresIn":        int(AccessTokenTTL.Seconds()),
		"currentSessionId": sessionID,
	})
}

func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "no token", http.StatusUnauthorized)
		return
	}
	
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}
	
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	
	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "invalid claims", http.StatusUnauthorized)
		return
	}
	
	writeJSON(w, map[string]interface{}{
		"valid": true,
		"role":  claims["role"],
		"email": claims["email"],
	})
}

func (s *Server) requireAuth(handler func(http.ResponseWriter, *http.Request, jwt.MapClaims)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "no token", http.StatusUnauthorized)
			return
		}
		
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		
		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "dev-secret"
		}
		
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "invalid claims", http.StatusUnauthorized)
			return
		}
		
		handler(w, r, claims)
	}
}
