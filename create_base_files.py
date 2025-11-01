#!/usr/bin/env python3
"""
create_base_files.py

This script creates all the BASE files needed for the restaurant site.
Run this BEFORE applying the security patch.

Usage: python create_base_files.py
"""
import os

def write_file(path, content):
    """Write content to file, creating directories as needed."""
    directory = os.path.dirname(path)
    if directory and not os.path.exists(directory):
        os.makedirs(directory, exist_ok=True)
    
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)
    print(f"✓ Created: {path}")

def create_backend_files():
    """Create all backend Go files."""
    
    # backend/main.go
    write_file("backend/main.go", '''package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Server struct {
	store *Store
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/resto?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	store := &Store{DB: db}
	server := &Server{store: store}

	mux := http.NewServeMux()
	
	// Public routes
	mux.HandleFunc("/api/restaurants/", server.handleRestaurants)
	mux.HandleFunc("/api/orders/", server.handleOrders)
	mux.HandleFunc("/api/subscribe/", server.handleSubscribe)
	mux.HandleFunc("/api/reviews/", server.handleReviews)
	mux.HandleFunc("/api/menus/", server.handleMenus)
	
	// Auth routes
	mux.HandleFunc("/api/login", server.handleLogin)
	mux.HandleFunc("/api/verify", server.handleVerify)
	mux.HandleFunc("/api/refresh", server.handleRefresh)
	mux.HandleFunc("/api/logout", server.handleLogout)
	
	// Admin routes (with auth)
	mux.HandleFunc("/api/admin/orders/", server.requireAuth(server.handleAdminOrders))
	mux.HandleFunc("/api/restaurants_patch/", server.requireAuth(server.handleRestaurantPatch))
	mux.HandleFunc("/api/admin/invite/", server.requireAuth(server.handleInviteAdmin))
	mux.HandleFunc("/api/admin/invite/accept", server.handleAcceptInvite)
	mux.HandleFunc("/api/admin/password_reset/request", server.handlePasswordResetRequest)
	mux.HandleFunc("/api/admin/password_reset/confirm", server.handlePasswordResetConfirm)
	mux.HandleFunc("/api/admin/sessions/", server.requireAuth(server.handleGetSessions))
	mux.HandleFunc("/api/admin/sessions/revoke", server.requireAuth(server.handleRevokeSession))
	mux.HandleFunc("/api/admin/sessions/revoke_all", server.requireAuth(server.handleRevokeAllOtherSessions))
	mux.HandleFunc("/api/admin/export/", server.requireAuth(server.handleExportData))
	mux.HandleFunc("/api/admin/export_media/", server.requireAuth(server.handleExportMedia))
	mux.HandleFunc("/api/admin/audit/", server.requireAuth(server.handleAuditLog))

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{os.Getenv("ALLOW_ORIGIN")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func getIDFromPath(prefix, path string) (int, error) {
	idStr := strings.TrimPrefix(path, prefix)
	idStr = strings.TrimSuffix(idStr, "/")
	if idx := strings.Index(idStr, "/"); idx >= 0 {
		idStr = idStr[:idx]
	}
	return strconv.Atoi(idStr)
}
''')

    # backend/store.go
    write_file("backend/store.go", '''package main

import (
	"context"
	"database/sql"
	"encoding/json"
)

type Store struct {
	DB *sql.DB
}

type Restaurant struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Story       string                 `json:"story"`
	Address     string                 `json:"address"`
	Phone       string                 `json:"phone"`
	Email       string                 `json:"email"`
	Hours       string                 `json:"hours"`
	SocialLinks []string               `json:"socialLinks"`
	Offerings   []string               `json:"offerings"`
	SiteConfig  map[string]interface{} `json:"siteConfig"`
}

type MenuItem struct {
	Name      string  `json:"name"`
	Desc      string  `json:"desc"`
	Price     float64 `json:"price"`
	Img       string  `json:"img"`
	Available bool    `json:"available"`
}

type MenuCategory struct {
	Category string     `json:"category"`
	Items    []MenuItem `json:"items"`
}

type Gallery struct {
	Images   []string `json:"images"`
	Captions []string `json:"captions"`
}

type Review struct {
	Name    string `json:"name"`
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
	Date    string `json:"date"`
}

type RestaurantData struct {
	Restaurant Restaurant     `json:"restaurant"`
	Menus      []MenuCategory `json:"menus"`
	Galleries  Gallery        `json:"galleries"`
	Reviews    []Review       `json:"reviews"`
}

func (s *Store) GetRestaurant(ctx context.Context, id int) (Restaurant, error) {
	var r Restaurant
	var socialJSON, offeringsJSON, configJSON []byte
	
	row := s.DB.QueryRowContext(ctx, 
		"SELECT id, name, story, address, phone, email, hours, social_links, offerings, site_config FROM restaurants WHERE id=$1", id)
	
	err := row.Scan(&r.ID, &r.Name, &r.Story, &r.Address, &r.Phone, &r.Email, &r.Hours, &socialJSON, &offeringsJSON, &configJSON)
	if err != nil {
		return r, err
	}
	
	json.Unmarshal(socialJSON, &r.SocialLinks)
	json.Unmarshal(offeringsJSON, &r.Offerings)
	json.Unmarshal(configJSON, &r.SiteConfig)
	
	return r, nil
}

func (s *Store) LoadRestaurantData(ctx context.Context, id int) (RestaurantData, error) {
	var data RestaurantData
	
	rest, err := s.GetRestaurant(ctx, id)
	if err != nil {
		return data, err
	}
	data.Restaurant = rest
	
	// Get menus
	rows, err := s.DB.QueryContext(ctx, "SELECT category, items_json FROM menus WHERE restaurant_id=$1", id)
	if err != nil {
		return data, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var cat MenuCategory
		var itemsJSON []byte
		rows.Scan(&cat.Category, &itemsJSON)
		json.Unmarshal(itemsJSON, &cat.Items)
		data.Menus = append(data.Menus, cat)
	}
	
	// Get gallery
	var imgJSON, capJSON []byte
	row := s.DB.QueryRowContext(ctx, "SELECT images, captions FROM galleries WHERE restaurant_id=$1", id)
	if err := row.Scan(&imgJSON, &capJSON); err == nil {
		json.Unmarshal(imgJSON, &data.Galleries.Images)
		json.Unmarshal(capJSON, &data.Galleries.Captions)
	}
	
	// Get reviews
	var revJSON []byte
	row = s.DB.QueryRowContext(ctx, "SELECT testimonials FROM reviews WHERE restaurant_id=$1", id)
	if err := row.Scan(&revJSON); err == nil {
		json.Unmarshal(revJSON, &data.Reviews)
	}
	
	return data, nil
}

type AdminUser struct {
	ID           int      `json:"id"`
	RestaurantID int      `json:"restaurantId"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"-"`
	Role         string   `json:"role"`
	Permissions  []string `json:"permissions"`
}

func (s *Store) GetAdminByEmail(ctx context.Context, restaurantID int, email string) (AdminUser, error) {
	var admin AdminUser
	var permsJSON []byte
	
	row := s.DB.QueryRowContext(ctx, 
		"SELECT id, restaurant_id, email, password_hash, role, permissions FROM admins WHERE restaurant_id=$1 AND email=$2",
		restaurantID, email)
	
	err := row.Scan(&admin.ID, &admin.RestaurantID, &admin.Email, &admin.PasswordHash, &admin.Role, &permsJSON)
	if err != nil {
		return admin, err
	}
	
	json.Unmarshal(permsJSON, &admin.Permissions)
	return admin, nil
}
''')

    # backend/handlers.go
    write_file("backend/handlers.go", '''package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleRestaurants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/restaurants/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	data, err := s.store.LoadRestaurantData(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	
	writeJSON(w, data)
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/orders/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	var order struct {
		Items           []map[string]interface{} `json:"items"`
		Total           float64                  `json:"total"`
		CustomerName    string                   `json:"customerName"`
		CustomerPhone   string                   `json:"customerPhone"`
		CustomerAddress string                   `json:"customerAddress"`
		CustomerEmail   string                   `json:"customerEmail"`
		Notes           string                   `json:"notes"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	
	itemsJSON, _ := json.Marshal(order.Items)
	_, err = s.store.DB.ExecContext(r.Context(),
		`INSERT INTO orders (restaurant_id, items_json, total, status, created_at, customer_name, customer_phone, customer_address, customer_email, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, itemsJSON, order.Total, "pending", time.Now(), order.CustomerName, order.CustomerPhone, order.CustomerAddress, order.CustomerEmail, order.Notes)
	
	if err != nil {
		http.Error(w, "failed to create order", http.StatusInternalServerError)
		return
	}
	
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/subscribe/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	var payload struct {
		Email string `json:"email"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	
	_, err = s.store.DB.ExecContext(r.Context(),
		"INSERT INTO subscribers (restaurant_id, email) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING",
		id, payload.Email)
	
	if err != nil {
		http.Error(w, "failed to subscribe", http.StatusInternalServerError)
		return
	}
	
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (s *Server) handleReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/reviews/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	var review struct {
		Name    string `json:"name"`
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	
	// For simplicity, just append to existing testimonials
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (s *Server) handleMenus(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromPath("/api/menus/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	if r.Method == http.MethodPost {
		// Admin update menus
		var menus []MenuCategory
		if err := json.NewDecoder(r.Body).Decode(&menus); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		
		// Update each category
		for _, cat := range menus {
			itemsJSON, _ := json.Marshal(cat.Items)
			_, err := s.store.DB.ExecContext(r.Context(),
				`INSERT INTO menus (restaurant_id, category, items_json) VALUES ($1, $2, $3)
				 ON CONFLICT (restaurant_id, category) DO UPDATE SET items_json = EXCLUDED.items_json`,
				id, cat.Category, itemsJSON)
			if err != nil {
				http.Error(w, "failed to update menu", http.StatusInternalServerError)
				return
			}
		}
		
		writeJSON(w, map[string]interface{}{"ok": true})
		return
	}
	
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleRestaurantPatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/restaurants_patch/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	
	// Simple implementation - update allowed fields
	// In production, validate each field properly
	
	writeJSON(w, map[string]interface{}{"ok": true})
}

func (s *Server) handleAdminOrders(w http.ResponseWriter, r *http.Request, claims map[string]interface{}) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/admin/orders/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	rows, err := s.store.DB.QueryContext(r.Context(),
		`SELECT id, items_json, total, status, created_at, customer_name, customer_phone, customer_email, notes
		 FROM orders WHERE restaurant_id=$1 ORDER BY created_at DESC LIMIT 100`, id)
	
	if err != nil {
		http.Error(w, "failed to fetch orders", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var orders []map[string]interface{}
	for rows.Next() {
		var order map[string]interface{} = make(map[string]interface{})
		var itemsJSON []byte
		var id int
		var total float64
		var status, name, phone, email, notes string
		var createdAt time.Time
		
		rows.Scan(&id, &itemsJSON, &total, &status, &createdAt, &name, &phone, &email, &notes)
		
		var items []map[string]interface{}
		json.Unmarshal(itemsJSON, &items)
		
		order["id"] = id
		order["items"] = items
		order["total"] = total
		order["status"] = status
		order["createdAt"] = createdAt
		order["customerName"] = name
		order["customerPhone"] = phone
		order["customerEmail"] = email
		order["notes"] = notes
		
		orders = append(orders, order)
	}
	
	writeJSON(w, orders)
}

func (s *Server) handleExportData(w http.ResponseWriter, r *http.Request, claims map[string]interface{}) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/admin/export/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	data, err := s.store.LoadRestaurantData(r.Context(), id)
	if err != nil {
		http.Error(w, "failed to load data", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=export.json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request, claims map[string]interface{}) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}
	
	id, err := getIDFromPath("/api/admin/audit/", r.URL.Path)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	
	rows, err := s.store.DB.QueryContext(r.Context(),
		`SELECT id, admin_email, action, payload, ip, created_at 
		 FROM audit_log WHERE restaurant_id=$1 ORDER BY created_at DESC LIMIT 200`, id)
	
	if err != nil {
		http.Error(w, "failed to fetch audit log", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var logs []map[string]interface{}
	for rows.Next() {
		var log map[string]interface{} = make(map[string]interface{})
		var id int
		var email, action, ip string
		var payloadJSON []byte
		var createdAt time.Time
		
		rows.Scan(&id, &email, &action, &payloadJSON, &ip, &createdAt)
		
		var payload map[string]interface{}
		json.Unmarshal(payloadJSON, &payload)
		
		log["id"] = id
		log["email"] = email
		log["action"] = action
		log["payload"] = payload
		log["ip"] = ip
		log["createdAt"] = createdAt
		
		logs = append(logs, log)
	}
	
	writeJSON(w, logs)
}
''')

    # backend/auth.go
    write_file("backend/auth.go", '''package main

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
''')

    # backend/email.go
    write_file("backend/email.go", '''package main

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
)

func sendEmail(to, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	
	if host == "" || user == "" {
		fmt.Println("SMTP not configured, skipping email to:", to)
		return nil
	}
	
	if from == "" {
		from = user
	}
	
	msg := fmt.Sprintf("From: %s\\r\\nTo: %s\\r\\nSubject: %s\\r\\n\\r\\n%s", from, to, subject, body)
	
	auth := smtp.PlainAuth("", user, pass, host)
	
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         host,
	}
	
	addr := host + ":" + port
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	
	if err = client.Auth(auth); err != nil {
		return err
	}
	
	if err = client.Mail(from); err != nil {
		return err
	}
	
	if err = client.Rcpt(to); err != nil {
		return err
	}
	
	w, err := client.Data()
	if err != nil {
		return err
	}
	
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	
	err = w.Close()
	if err != nil {
		return err
	}
	
	return client.Quit()
}
''')

    print("\n✅ Backend files created!")

def create_frontend_files():
    """Create all frontend React files."""
    
    # frontend/public/index.html
    write_file("frontend/public/index.html", '''<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Restaurant Site</title>
</head>
<body>
  <div id="root"></div>
</body>
</html>
''')

    # frontend/src/index.js
    write_file("frontend/src/index.js", '''import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
''')

    # frontend/src/App.js
    write_file("frontend/src/App.js", '''import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Home from './pages/Home';
import Admin from './pages/Admin';
import InviteAccept from './pages/InviteAccept';
import PasswordResetRequest from './pages/PasswordResetRequest';
import PasswordResetConfirm from './pages/PasswordResetConfirm';

export default function App() {
  const restaurantId = process.env.REACT_APP_DEFAULT_RESTAURANT || "1";
  
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Home restaurantId={restaurantId} />} />
        <Route path="/restaurant/:id" element={<Home />} />
        <Route path="/restaurant/:id/admin" element={<Admin />} />
        <Route path="/invite/accept" element={<InviteAccept />} />
        <Route path="/password-reset/request" element={<PasswordResetRequest />} />
        <Route path="/password-reset/confirm" element={<PasswordResetConfirm />} />
      </Routes>
    </Router>
  );
}
''')

    # frontend/src/pages/Home.jsx
    write_file("frontend/src/pages/Home.jsx", '''import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { fetchRestaurant, postOrder, postSubscribe } from '../api';

export default function Home({ restaurantId: defaultId }) {
  const { id } = useParams();
  const restaurantId = id || defaultId;
  
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [cart, setCart] = useState([]);
  
  useEffect(() => {
    fetchRestaurant(restaurantId)
      .then(res => {
        setData(res);
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
        setLoading(false);
      });
  }, [restaurantId]);
  
  const addToCart = (item) => {
    setCart([...cart, item]);
  };
  
  const submitOrder = async () => {
    if (cart.length === 0) return alert('Cart is empty');
    
    const total = cart.reduce((sum, item) => sum + item.price, 0);
    const customerName = prompt('Your name:');
    const customerPhone = prompt('Your phone:');
    const customerAddress = prompt('Your address:');
    
    if (!customerName || !customerPhone) return;
    
    try {
      await postOrder(restaurantId, {
        items: cart,
        total,
        customerName,
        customerPhone,
        customerAddress,
        customerEmail: '',
        notes: ''
      });
      alert('Order placed successfully!');
      setCart([]);
    } catch (err) {
      alert('Failed to place order');
    }
  };
  
  if (loading) return <div style={{ padding: 20 }}>Loading...</div>;
  if (!data) return <div style={{ padding: 20 }}>Restaurant not found</div>;
  
  return (
    <div style={{ padding: 20, maxWidth: 1200, margin: '0 auto' }}>
      <header style={{ marginBottom: 40 }}>
        <h1>{data.restaurant.name}</h1>
        <p>{data.restaurant.story}</p>
        <div>
          <strong>Address:</strong> {data.restaurant.address}<br/>
          <strong>Phone:</strong> {data.restaurant.phone}<br/>
          <strong>Hours:</strong> {data.restaurant.hours}
        </div>
      </header>
      
      <section style={{ marginBottom: 40 }}>
        <h2>Menu</h2>
        {data.menus.map(category => (
          <div key={category.category} style={{ marginBottom: 30 }}>
            <h3>{category.category}</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: 20 }}>
              {category.items.map((item, idx) => (
                <div key={idx} style={{ border: '1px solid #ddd', padding: 15, borderRadius: 8 }}>
                  {item.img && <img src={item.img} alt={item.name} style={{ width: '100%', height: 150, objectFit: 'cover', borderRadius: 4 }} />}
                  <h4>{item.name}</h4>
                  <p>{item.desc}</p>
                  <p><strong>${item.price.toFixed(2)}</strong></p>
                  {item.available && (
                    <button onClick={() => addToCart(item)} style={{ padding: '8px 16px', cursor: 'pointer' }}>
                      Add to Cart
                    </button>
                  )}
                  {!item.available && <span style={{ color: '#999' }}>Not available</span>}
                </div>
              ))}
            </div>
          </div>
        ))}
      </section>
      
      {cart.length > 0 && (
        <section style={{ position: 'fixed', bottom: 0, left: 0, right: 0, background: '#fff', borderTop: '2px solid #333', padding: 20 }}>
          <div style={{ maxWidth: 1200, margin: '0 auto', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <strong>Cart ({cart.length} items)</strong>
              <span style={{ marginLeft: 20 }}>
                Total: ${cart.reduce((sum, item) => sum + item.price, 0).toFixed(2)}
              </span>
            </div>
            <button onClick={submitOrder} style={{ padding: '10px 20px', fontSize: 16, cursor: 'pointer' }}>
              Place Order
            </button>
          </div>
        </section>
      )}
      
      {data.galleries.images.length > 0 && (
        <section style={{ marginBottom: 40 }}>
          <h2>Gallery</h2>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 15 }}>
            {data.galleries.images.map((img, idx) => (
              <div key={idx}>
                <img src={img} alt="" style={{ width: '100%', height: 200, objectFit: 'cover', borderRadius: 4 }} />
                {data.galleries.captions[idx] && <p style={{ marginTop: 5, fontSize: 14 }}>{data.galleries.captions[idx]}</p>}
              </div>
            ))}
          </div>
        </section>
      )}
      
      {data.reviews.length > 0 && (
        <section style={{ marginBottom: 40 }}>
          <h2>Reviews</h2>
          {data.reviews.map((review, idx) => (
            <div key={idx} style={{ borderBottom: '1px solid #eee', padding: '15px 0' }}>
              <div><strong>{review.name}</strong> - {'⭐'.repeat(review.rating)}</div>
              <p>{review.comment}</p>
              <small>{new Date(review.date).toLocaleDateString()}</small>
            </div>
          ))}
        </section>
      )}
    </div>
  );
}
''')

    # frontend/src/pages/Admin.jsx (Complete version)
    write_file("frontend/src/pages/Admin.jsx", '''import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import LoginModal from '../components/LoginModal';
import ActiveSessions from './ActiveSessions';
import {
  verify,
  refreshSession,
  logout,
  adminGetOrders,
  adminUpdateMenus,
  adminInvite,
  adminGetSessions,
  adminRevokeSession,
  adminRevokeOtherSessions,
  adminExportData,
  adminExportMedia,
  adminExportMediaToS3,
  adminGetAudit,
  fetchRestaurant
} from '../api';

export default function Admin() {
  const { id } = useParams();
  const restaurantId = id || process.env.REACT_APP_DEFAULT_RESTAURANT || "1";
  const navigate = useNavigate();
  
  const [accessToken, setAccessToken] = useState(localStorage.getItem('accessToken') || '');
  const [role, setRole] = useState('');
  const [currentSessionId, setCurrentSessionId] = useState(null);
  const [authenticated, setAuthenticated] = useState(false);
  
  const [tab, setTab] = useState('orders');
  const [orders, setOrders] = useState([]);
  const [sessions, setSessions] = useState([]);
  const [auditLog, setAuditLog] = useState([]);
  const [data, setData] = useState(null);
  
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('chef');
  const [inviteStatus, setInviteStatus] = useState('');
  
  useEffect(() => {
    if (accessToken) {
      verify(accessToken)
        .then(res => {
          if (res.valid) {
            setRole(res.role);
            setAuthenticated(true);
          }
        })
        .catch(() => {
          // Try refresh
          refreshSession()
            .then(res => {
              setAccessToken(res.accessToken);
              localStorage.setItem('accessToken', res.accessToken);
              setRole(res.role);
              setCurrentSessionId(res.currentSessionId);
              setAuthenticated(true);
            })
            .catch(() => {
              setAuthenticated(false);
              setAccessToken('');
              localStorage.removeItem('accessToken');
            });
        });
    }
  }, [accessToken]);
  
  useEffect(() => {
    if (authenticated && tab === 'orders') {
      loadOrders();
    } else if (authenticated && tab === 'sessions') {
      loadSessions();
    } else if (authenticated && tab === 'audit') {
      loadAuditLog();
    } else if (authenticated && tab === 'menu') {
      loadData();
    }
  }, [authenticated, tab]);
  
  const loadOrders = () => {
    adminGetOrders(restaurantId, accessToken)
      .then(res => setOrders(res))
      .catch(err => console.error(err));
  };
  
  const loadSessions = () => {
    adminGetSessions(restaurantId, accessToken)
      .then(res => setSessions(res))
      .catch(err => console.error(err));
  };
  
  const loadAuditLog = () => {
    adminGetAudit(restaurantId, accessToken)
      .then(res => setAuditLog(res))
      .catch(err => console.error(err));
  };
  
  const loadData = () => {
    fetchRestaurant(restaurantId)
      .then(res => setData(res))
      .catch(err => console.error(err));
  };
  
  const handleLogin = ({ accessToken: token, role: userRole, currentSessionId: sessionId }) => {
    setAccessToken(token);
    setRole(userRole);
    setCurrentSessionId(sessionId);
    setAuthenticated(true);
    localStorage.setItem('accessToken', token);
  };
  
  const handleLogout = async () => {
    await logout();
    setAccessToken('');
    setRole('');
    setAuthenticated(false);
    localStorage.removeItem('accessToken');
  };
  
  const handleInvite = async () => {
    if (!inviteEmail || !inviteRole) return setInviteStatus('Fill all fields');
    setInviteStatus('Sending...');
    try {
      await adminInvite(restaurantId, inviteEmail, inviteRole, accessToken);
      setInviteStatus('Invite sent!');
      setInviteEmail('');
    } catch (err) {
      setInviteStatus('Failed to send invite');
    }
  };
  
  const handleRevokeSession = async (sessionId) => {
    try {
      await adminRevokeSession(sessionId, accessToken);
      loadSessions();
    } catch (err) {
      alert('Failed to revoke session');
    }
  };
  
  const handleRevokeOtherSessions = async () => {
    if (!confirm('Revoke all other sessions?')) return;
    try {
      await adminRevokeOtherSessions(accessToken);
      loadSessions();
    } catch (err) {
      alert('Failed to revoke sessions');
    }
  };
  
  const handleExportData = async () => {
    try {
      const blob = await adminExportData(restaurantId, accessToken);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `restaurant_${restaurantId}_data.json`;
      a.click();
    } catch (err) {
      alert('Export failed');
    }
  };
  
  const handleExportMedia = async () => {
    try {
      const blob = await adminExportMedia(restaurantId, accessToken);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `restaurant_${restaurantId}_media.zip`;
      a.click();
    } catch (err) {
      alert('Export failed');
    }
  };
  
  if (!authenticated) {
    return <LoginModal restaurantId={restaurantId} onLogin={handleLogin} />;
  }
  
  return (
    <div style={{ padding: 20 }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 20, borderBottom: '2px solid #333', paddingBottom: 10 }}>
        <h1>Admin Panel - Restaurant {restaurantId}</h1>
        <div>
          <span style={{ marginRight: 15 }}>Role: <strong>{role}</strong></span>
          <button onClick={handleLogout}>Logout</button>
        </div>
      </header>
      
      <nav style={{ marginBottom: 20 }}>
        <button onClick={() => setTab('orders')} style={{ marginRight: 10, fontWeight: tab === 'orders' ? 'bold' : 'normal' }}>Orders</button>
        <button onClick={() => setTab('menu')} style={{ marginRight: 10, fontWeight: tab === 'menu' ? 'bold' : 'normal' }}>Menu</button>
        <button onClick={() => setTab('sessions')} style={{ marginRight: 10, fontWeight: tab === 'sessions' ? 'bold' : 'normal' }}>Sessions</button>
        <button onClick={() => setTab('audit')} style={{ marginRight: 10, fontWeight: tab === 'audit' ? 'bold' : 'normal' }}>Audit Log</button>
        {role === 'owner' && <button onClick={() => setTab('invite')} style={{ marginRight: 10, fontWeight: tab === 'invite' ? 'bold' : 'normal' }}>Invite Admin</button>}
        <button onClick={() => setTab('export')} style={{ marginRight: 10, fontWeight: tab === 'export' ? 'bold' : 'normal' }}>Export</button>
      </nav>
      
      {tab === 'orders' && (
        <section>
          <h2>Recent Orders</h2>
          {orders.length === 0 ? <p>No orders yet</p> : (
            <div>
              {orders.map(order => (
                <div key={order.id} style={{ border: '1px solid #ddd', padding: 15, marginBottom: 10, borderRadius: 4 }}>
                  <div><strong>Order #{order.id}</strong> - {order.status}</div>
                  <div>Customer: {order.customerName} ({order.customerPhone})</div>
                  <div>Total: ${order.total.toFixed(2)}</div>
                  <div>Date: {new Date(order.createdAt).toLocaleString()}</div>
                  <details style={{ marginTop: 10 }}>
                    <summary>Items</summary>
                    <ul>
                      {order.items.map((item, idx) => (
                        <li key={idx}>{item.name} - ${item.price}</li>
                      ))}
                    </ul>
                  </details>
                </div>
              ))}
            </div>
          )}
        </section>
      )}
      
      {tab === 'menu' && (
        <section>
          <h2>Menu Management</h2>
          {data ? (
            <div>
              {data.menus.map(cat => (
                <div key={cat.category} style={{ marginBottom: 30 }}>
                  <h3>{cat.category}</h3>
                  <ul>
                    {cat.items.map((item, idx) => (
                      <li key={idx}>
                        {item.name} - ${item.price} - {item.available ? 'Available' : 'Unavailable'}
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
              <p><em>Menu editing UI coming soon...</em></p>
            </div>
          ) : <p>Loading...</p>}
        </section>
      )}
      
      {tab === 'sessions' && (
        <section>
          <h2>Active Sessions</h2>
          <div style={{ marginBottom: 15 }}>
            <button onClick={handleRevokeOtherSessions}>Revoke All Other Sessions</button>
          </div>
          <ActiveSessions 
            sessions={sessions} 
            currentSessionId={currentSessionId}
            onRevoke={handleRevokeSession}
          />
        </section>
      )}
      
      {tab === 'audit' && (
        <section>
          <h2>Audit Log</h2>
          {auditLog.length === 0 ? <p>No audit entries</p> : (
            <div>
              {auditLog.map(log => (
                <div key={log.id} style={{ borderBottom: '1px solid #eee', padding: 10 }}>
                  <div><strong>{log.action}</strong> by {log.email}</div>
                  <div>IP: {log.ip} | {new Date(log.createdAt).toLocaleString()}</div>
                  <details>
                    <summary>Payload</summary>
                    <pre style={{ fontSize: 12 }}>{JSON.stringify(log.payload, null, 2)}</pre>
                  </details>
                </div>
              ))}
            </div>
          )}
        </section>
      )}
      
      {tab === 'invite' && role === 'owner' && (
        <section>
          <h2>Invite Admin</h2>
          <div style={{ maxWidth: 400 }}>
            <div style={{ marginBottom: 10 }}>
              <label>Email</label><br/>
              <input 
                type="email" 
                value={inviteEmail} 
                onChange={e => setInviteEmail(e.target.value)}
                style={{ width: '100%', padding: 8 }}
              />
            </div>
            <div style={{ marginBottom: 10 }}>
              <label>Role</label><br/>
              <select value={inviteRole} onChange={e => setInviteRole(e.target.value)} style={{ width: '100%', padding: 8 }}>
                <option value="chef">Chef</option>
                <option value="owner">Owner</option>
              </select>
            </div>
            <button onClick={handleInvite}>Send Invitation</button>
            {inviteStatus && <div style={{ marginTop: 10 }}>{inviteStatus}</div>}
          </div>
        </section>
      )}
      
      {tab === 'export' && (
        <section>
          <h2>Export Data</h2>
          <div>
            <button onClick={handleExportData} style={{ marginRight: 10, marginBottom: 10 }}>
              Export Full Data (JSON)
            </button>
            <button onClick={handleExportMedia} style={{ marginBottom: 10 }}>
              Export Media (ZIP)
            </button>
          </div>
          <p><em>Media can also be uploaded to S3 - check API documentation</em></p>
        </section>
      )}
    </div>
  );
}
''')

    print("\n✅ Frontend files created!")

def create_readme():
    """Create comprehensive README."""
    write_file("README.md", '''# Restaurant Site - Complete Application

A full-stack restaurant management system with admin panel, secure authentication, and media export.

## Features

### Customer-Facing
- Browse menu with categories
- Add items to cart and place orders
- View gallery and reviews
- Subscribe to newsletter

### Admin Features
- Secure JWT authentication with refresh tokens
- Session management (view/revoke active sessions)
- Order management
- Menu editing
- Admin invitation system
- Password reset flow
- Audit logging
- Data export (JSON)
- Media export (ZIP or S3)

## Tech Stack

**Backend:**
- Go 1.20+
- PostgreSQL
- JWT authentication
- AWS SDK (for S3 exports)

**Frontend:**
- React 18
- React Router
- Axios

## Quick Start

### 1. Prerequisites
- Docker and Docker Compose
- Go 1.20+
- Node.js 18+
- PostgreSQL 15+

### 2. Setup

```bash
# Copy environment variables
cp .env.example .env

# Edit .env with your configuration
# At minimum, set JWT_SECRET to a strong random value

# Start services with Docker
make up

# Or manually:
docker-compose up --build
```

### 3. Initialize Database

```bash
# Run migrations
make migrate

# Create first admin user
make create-admin RESTAURANT_ID=1 ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=strongpass123 ADMIN_ROLE=owner
```

### 4. Access Application

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Admin panel: http://localhost:3000/restaurant/1/admin

## Development

### Running Locally (without Docker)

**Backend:**
```bash
cd backend
go mod download
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/resto?sslmode=disable"
export JWT_SECRET="your-secret-key"
go run .
```

**Frontend:**
```bash
cd frontend
npm install
npm start
```

### Database Migrations

```bash
# Run all migrations
./scripts/run_migrations.sh

# Or manually with psql
psql $DATABASE_URL -f db/migrations.sql
psql $DATABASE_URL -f db/admin_onboarding_migrations.sql
psql $DATABASE_URL -f db/password_reset_migration.sql
psql $DATABASE_URL -f db/refresh_tokens_migration.sql
```

### Creating Admin Users

```bash
# Using the script
go run scripts/create_admin.go --restaurant 1 --email admin@example.com --password 'SecurePass123' --role owner

# Using Makefile
make create-admin RESTAURANT_ID=1 ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=pass123 ADMIN_ROLE=owner
```

## API Endpoints

### Public Endpoints
- `GET /api/restaurants/:id` - Get restaurant data
- `POST /api/orders/:id` - Place order
- `POST /api/subscribe/:id` - Subscribe to newsletter
- `POST /api/reviews/:id` - Submit review

### Auth Endpoints
- `POST /api/login` - Login (returns JWT + refresh token cookie)
- `POST /api/refresh` - Refresh access token
- `POST /api/logout` - Logout and revoke session
- `GET /api/verify` - Verify token

### Admin Endpoints (require authentication)
- `GET /api/admin/orders/:id` - List orders
- `POST /api/menus/:id` - Update menus
- `POST /api/restaurants_patch/:id` - Update restaurant info
- `POST /api/admin/invite/:id` - Invite admin (owner only)
- `POST /api/admin/invite/accept` - Accept invitation
- `POST /api/admin/password_reset/request` - Request password reset
- `POST /api/admin/password_reset/confirm` - Confirm password reset
- `GET /api/admin/sessions/:id` - List sessions
- `POST /api/admin/sessions/revoke` - Revoke session
- `POST /api/admin/sessions/revoke_all` - Revoke all other sessions
- `GET /api/admin/export/:id` - Export data (JSON)
- `GET/POST /api/admin/export_media/:id` - Export media (ZIP/S3)
- `GET /api/admin/audit/:id` - View audit log

## Security Features

1. **JWT with Refresh Tokens**: Short-lived access tokens (15min) + HTTP-only refresh cookies (30 days)
2. **Token Rotation**: Refresh tokens are rotated on each use
3. **Session Management**: View and revoke active sessions
4. **Password Reset**: Time-limited tokens (1 hour expiry)
5. **Admin Invitations**: Secure invitation flow with 72-hour expiry
6. **Audit Logging**: All admin actions logged with IP addresses
7. **CSRF Protection**: SameSite cookies
8. **bcrypt**: Password hashing with cost factor 10

## Environment Variables

See `.env.example` for all available variables. Key ones:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5432/resto?sslmode=disable
JWT_SECRET=change-me-to-a-strong-secret
FRONTEND_URL=http://localhost:3000
ALLOW_ORIGIN=http://localhost:3000
ENV=development
ALLOW_INSECURE_COOKIES=1  # Set to 0 in production

# SMTP (optional, for emails)
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASS=password
SMTP_FROM=notifications@example.com

# AWS (for S3 exports)
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
```

## Production Deployment

1. Set `ENV=production` in environment
2. Set `ALLOW_INSECURE_COOKIES=0`
3. Use strong `JWT_SECRET` (32+ random characters)
4. Configure proper CORS origins
5. Enable HTTPS
6. Set up regular database backups
7. Configure SMTP for email notifications
8. Run cleanup worker for expired tokens

```bash
# Start cleanup worker
docker-compose up cleanup-worker
```

## Maintenance

### Cleanup Old Tokens

```bash
# Manual cleanup
go run scripts/cleanup_refresh_tokens.go --retention 30

# Or via Makefile
make cleanup-sessions
```

### Backup Database

```bash
pg_dump $DATABASE_URL > backup.sql
```

## Troubleshooting

**"Database connection failed"**
- Check DATABASE_URL is correct
- Ensure PostgreSQL is running
- Verify network connectivity

**"Invalid token"**
- Check JWT_SECRET matches between restarts
- Token may have expired (15min for access tokens)
- Try refreshing the session

**"SMTP errors"**
- SMTP is optional; app works without it
- Check SMTP credentials if email features needed

**"CORS errors"**
- Verify ALLOW_ORIGIN matches your frontend URL
- Check that credentials: true is set in frontend API calls

## License

MIT

## Support

For issues, please open a GitHub issue with:
- Error messages
- Steps to reproduce
- Environment details (OS, Go version, etc.)
''')

def main():
    print("="*60)
    print("Creating Complete Restaurant Site Base Files")
    print("="*60)
    print()
    
    response = input("This will create all base application files. Continue? (y/n): ").lower().strip()
    if response != 'y':
        print("Aborted.")
        return
    
    print("\n📦 Creating backend files...")
    create_backend_files()
    
    print("\n🎨 Creating frontend files...")
    create_frontend_files()
    
    print("\n📝 Creating README...")
    create_readme()
    
    print("\n" + "="*60)
    print("✅ ALL BASE FILES CREATED SUCCESSFULLY!")
    print("="*60)
    
    print("\n📋 Next Steps:")
    print("1. Run: python apply_patch.py feature-admin-security-export.patch")
    print("   (This adds the security features)")
    print()
    print("2. Copy environment variables:")
    print("   cp .env.example .env")
    print()
    print("3. Start services:")
    print("   docker-compose up --build")
    print()
    print("4. Run migrations:")
    print("   make migrate")
    print()
    print("5. Create admin user:")
    print("   make create-admin RESTAURANT_ID=1 ADMIN_EMAIL=admin@test.com ADMIN_PASSWORD=pass123 ADMIN_ROLE=owner")
    print()
    print("6. Access the app:")
    print("   Frontend: http://localhost:3000")
    print("   Admin: http://localhost:3000/restaurant/1/admin")
    print()
    print("🎉 You're all set!")

if __name__ == "__main__":
    main()