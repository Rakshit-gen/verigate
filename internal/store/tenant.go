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
	"github.com/jackc/pgx/v5/pgconn"
)

type Tenant struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	RateLimitRPM int       `json:"rate_limit_rpm"`
	CreatedAt    time.Time `json:"created_at"`
}

// querier is the subset of *pgxpool.Pool and pgx.Tx that tenant/user
// queries need — letting the same query functions run either directly
// against the pool or inside a transaction (SignUp needs the latter, to
// create a user + tenant + session atomically).
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
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
// plaintext is ever available. Used by the admin-only tenant-creation path
// (HTTP route and CLI), which has no owning user.
func (s *Store) CreateTenant(ctx context.Context, name string, rateLimitRPM int) (*Tenant, string, error) {
	return createTenant(ctx, s.pool, name, rateLimitRPM, nil)
}

// createTenant is the shared insert used by both the ownerless admin path
// (CreateTenant) and self-serve signup, which passes ownerUserID and a
// transaction so the user/tenant/session rows commit atomically.
func createTenant(ctx context.Context, q querier, name string, rateLimitRPM int, ownerUserID *string) (*Tenant, string, error) {
	plaintext, err := GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("generating API key: %w", err)
	}

	var t Tenant
	err = q.QueryRow(ctx, `
		INSERT INTO tenants (name, api_key_hash, rate_limit_rpm, owner_user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, rate_limit_rpm, created_at
	`, name, hashAPIKey(plaintext), rateLimitRPM, ownerUserID).Scan(&t.ID, &t.Name, &t.RateLimitRPM, &t.CreatedAt)
	if err != nil {
		return nil, "", err
	}
	return &t, plaintext, nil
}

// RegenerateTenantKey issues a fresh API key for an existing tenant and
// invalidates the old one (only its hash is ever stored, so the old key
// simply stops matching). Used when a tenant/user has lost their
// one-time-shown key.
func (s *Store) RegenerateTenantKey(ctx context.Context, tenantID string) (string, error) {
	plaintext, err := GenerateAPIKey()
	if err != nil {
		return "", fmt.Errorf("generating API key: %w", err)
	}
	tag, err := s.pool.Exec(ctx, `UPDATE tenants SET api_key_hash = $1 WHERE id = $2`, hashAPIKey(plaintext), tenantID)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", ErrTenantNotFound
	}
	return plaintext, nil
}

// GetTenantByID looks up a tenant by its own ID — used to render a single
// tenant/session-scoped caller's own tenant back to them (as opposed to
// GetTenantByOwner, which starts from a user ID).
func (s *Store) GetTenantByID(ctx context.Context, id string) (*Tenant, error) {
	var t Tenant
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, rate_limit_rpm, created_at FROM tenants WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.RateLimitRPM, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetTenantByOwner returns the tenant owned by the given user — today's
// model is one user : one tenant, created together at signup.
func (s *Store) GetTenantByOwner(ctx context.Context, userID string) (*Tenant, error) {
	var t Tenant
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, rate_limit_rpm, created_at FROM tenants WHERE owner_user_id = $1
	`, userID).Scan(&t.ID, &t.Name, &t.RateLimitRPM, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
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
