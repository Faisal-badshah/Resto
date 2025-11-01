package main

// Usage:
// go run scripts/create_admin.go --restaurant 1 --email owner@example.com --password 'StrongPass123' --role owner

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	_ "github.com/lib/pq"
)

func main() {
	restaurant := flag.Int("restaurant", 0, "restaurant id")
	email := flag.String("email", "", "admin email")
	password := flag.String("password", "", "password in plain (will be hashed)")
	role := flag.String("role", "chef", "role (chef|owner)")
	flag.Parse()

	if *restaurant == 0 || *email == "" || *password == "" {
		flag.Usage()
		os.Exit(2)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/resto?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt: %v", err)
	}

	perms := "[]"
	_, err = db.Exec(`INSERT INTO admins (restaurant_id, email, password_hash, role, permissions) VALUES ($1,$2,$3,$4,$5)
	ON CONFLICT (restaurant_id, email) DO UPDATE SET password_hash=EXCLUDED.password_hash, role=EXCLUDED.role, permissions=EXCLUDED.permissions`,
		*restaurant, *email, string(hash), *role, perms)
	if err != nil {
		log.Fatalf("insert admin: %v", err)
	}
	fmt.Printf("Admin %s@restaurant:%d (%s) created/updated\n", *email, *restaurant, *role)
}
