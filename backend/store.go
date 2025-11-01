package main

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
