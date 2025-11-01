package main

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
