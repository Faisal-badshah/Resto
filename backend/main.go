package main

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
