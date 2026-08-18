package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Tenant struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	RateLimitRPM int       `json:"rate_limit_rpm"`
	CreatedAt    time.Time `json:"created_at"`
}

var ErrTenantNotFound = errors.New("tenant not found")

// GenerateAPIKey returns a fresh random key in the "vg_<64 hex chars>"
// shape — prefixed so a leaked key is recognizable as Verigate's at a
// glance (the same convention OpenAI/Stripe/Groq use for their own keys).
// Callers must show this to the user immediately: only its SHA-256 hash
// is ever persisted, so it cannot be recovered later.
func GenerateAPIKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "vg_" + hex.EncodeToString(buf), nil
}

func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// CreateTenant generates a new API key, stores only its hash, and returns
// both the Tenant record and the plaintext key — the ONLY time the
// plaintext is ever available.
func (s *Store) CreateTenant(ctx context.Context, name string, rateLimitRPM int) (*Tenant, string, error) {
	plaintext, err := GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("generating API key: %w", err)
	}

	var t Tenant
	err = s.pool.QueryRow(ctx, `
		INSERT INTO tenants (name, api_key_hash, rate_limit_rpm)
		VALUES ($1, $2, $3)
		RETURNING id, name, rate_limit_rpm, created_at
	`, name, hashAPIKey(plaintext), rateLimitRPM).Scan(&t.ID, &t.Name, &t.RateLimitRPM, &t.CreatedAt)
	if err != nil {
		return nil, "", err
	}
	return &t, plaintext, nil
}

// GetTenantByAPIKey hashes the presented key and looks up the matching
// tenant — the comparison happens in SQL on the hash, so the plaintext
// key never needs to be stored to be validated later.
func (s *Store) GetTenantByAPIKey(ctx context.Context, plaintextKey string) (*Tenant, error) {
	var t Tenant
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, rate_limit_rpm, created_at FROM tenants WHERE api_key_hash = $1
	`, hashAPIKey(plaintextKey)).Scan(&t.ID, &t.Name, &t.RateLimitRPM, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) ListTenants(ctx context.Context) ([]Tenant, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, rate_limit_rpm, created_at FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.RateLimitRPM, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
