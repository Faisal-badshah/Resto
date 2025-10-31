package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"
)

func generateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (s *Store) CreateRefreshToken(ctx context.Context, restaurantID int, email, ip, ua string, expiresIn time.Duration) (rawToken string, insertedID int, err error) {
	raw, err := generateRandomToken(32)
	if err != nil {
		return "", 0, err
	}
	hash := hashToken(raw)
	expires := time.Now().Add(expiresIn)

	var id int
	err = s.DB.QueryRowContext(ctx, "INSERT INTO refresh_tokens (restaurant_id, admin_email, token_hash, created_at, expires_at, ip, user_agent, revoked) VALUES ($1,$2,$3,$4,$5,$6,$7,false) RETURNING id",
		restaurantID, email, hash, time.Now(), expires, ip, ua).Scan(&id)
	if err != nil {
		return "", 0, err
	}
	return raw, id, nil
}

type RefreshRow struct {
	ID           int
	RestaurantID int
	Email        string
	CreatedAt    time.Time
	ExpiresAt    sql.NullTime
	Revoked      bool
	IP           string
	UserAgent    string
}

func (s *Store) FindRefreshToken(ctx context.Context, raw string) (RefreshRow, error) {
	var out RefreshRow
	hash := hashToken(raw)
	row := s.DB.QueryRowContext(ctx, "SELECT id, restaurant_id, admin_email, created_at, expires_at, revoked, ip, user_agent FROM refresh_tokens WHERE token_hash=$1", hash)
	if err := row.Scan(&out.ID, &out.RestaurantID, &out.Email, &out.CreatedAt, &out.ExpiresAt, &out.Revoked, &out.IP, &out.UserAgent); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Store) RevokeRefreshTokenByID(ctx context.Context, id int) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE refresh_tokens SET revoked=true WHERE id=$1", id)
	return err
}

func (s *Store) RevokeRefreshTokenByRaw(ctx context.Context, raw string) error {
	hash := hashToken(raw)
	_, err := s.DB.ExecContext(ctx, "UPDATE refresh_tokens SET revoked=true WHERE token_hash=$1", hash)
	return err
}

func (s *Store) RotateRefreshToken(ctx context.Context, oldRaw string, ip, ua string, expiresIn time.Duration) (newRaw string, newID int, err error) {
	found, err := s.FindRefreshToken(ctx, oldRaw)
	if err != nil {
		return "", 0, err
	}
	_ = s.RevokeRefreshTokenByID(ctx, found.ID)
	newRaw, newID, err = s.CreateRefreshToken(ctx, found.RestaurantID, found.Email, ip, ua, expiresIn)
	if err != nil {
		return "", 0, err
	}
	return newRaw, newID, nil
}

func (s *Store) ListRefreshTokens(ctx context.Context, restaurantID int, emailFilter *string, limit int) ([]RefreshRow, error) {
	if limit <= 0 {
		limit = 200
	}
	var rows *sql.Rows
	var err error
	if emailFilter == nil {
		rows, err = s.DB.QueryContext(ctx, "SELECT id, restaurant_id, admin_email, created_at, expires_at, revoked, ip, user_agent FROM refresh_tokens WHERE restaurant_id=$1 ORDER BY created_at DESC LIMIT $2", restaurantID, limit)
	} else {
		rows, err = s.DB.QueryContext(ctx, "SELECT id, restaurant_id, admin_email, created_at, expires_at, revoked, ip, user_agent FROM refresh_tokens WHERE restaurant_id=$1 AND admin_email=$2 ORDER BY created_at DESC LIMIT $3", restaurantID, *emailFilter, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RefreshRow
	for rows.Next() {
		var r RefreshRow
		_ = rows.Scan(&r.ID, &r.RestaurantID, &r.Email, &r.CreatedAt, &r.ExpiresAt, &r.Revoked, &r.IP, &r.UserAgent)
		out = append(out, r)
	}
	return out, nil
}

func (s *Store) CleanupExpiredRevokedTokens(ctx context.Context, retentionDays int) (int64, error) {
	threshold := time.Now().AddDate(0, 0, -retentionDays)
	res, err := s.DB.ExecContext(ctx, "DELETE FROM refresh_tokens WHERE (revoked = true AND created_at < $1) OR (expires_at IS NOT NULL AND expires_at < $1)", threshold)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
