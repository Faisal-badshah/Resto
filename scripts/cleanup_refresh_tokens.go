package main

// Usage: go run scripts/cleanup_refresh_tokens.go --retention 30
// Deletes revoked or expired refresh tokens older than retention days.

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	retention := flag.Int("retention", 30, "retention days for revoked/expired refresh tokens")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/resto?sslmode=disable"
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	threshold := time.Now().AddDate(0, 0, -*retention)
	res, err := db.Exec(`DELETE FROM refresh_tokens WHERE (revoked = true AND created_at < $1) OR (expires_at IS NOT NULL AND expires_at < $1)`, threshold)
	if err != nil {
		log.Fatalf("cleanup failed: %v", err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("Cleaned up %d refresh token(s) older than %d days\n", n, *retention)
}
