package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionInvalid     = errors.New("invalid or expired session")
)

const sessionTTL = 30 * 24 * time.Hour

// SignUp creates a user, a tenant owned by that user, and a logged-in
// session, all in one transaction — self-serve equivalent of the
// admin-only CreateTenant path, but starting from an email/password
// instead of an operator's request. Returns the plaintext API key and
// session token, each shown/usable only from this call onward (the API
// key never again; the session token until it expires or is logged out).
func (s *Store) SignUp(ctx context.Context, email, password, tenantName string) (*User, *Tenant, string, string, error) {
	email = normalizeEmail(email)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("hashing password: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, "", "", err
	}
	defer tx.Rollback(ctx)

	var u User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING id, email, created_at
	`, email, string(passwordHash)).Scan(&u.ID, &u.Email, &u.CreatedAt)
	if isUniqueViolation(err) {
		return nil, nil, "", "", ErrEmailTaken
	}
	if err != nil {
		return nil, nil, "", "", err
	}

	tenant, apiKey, err := createTenant(ctx, tx, tenantName, 60, &u.ID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("creating tenant: %w", err)
	}

	sessionToken, err := createSession(ctx, tx, u.ID)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("creating session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, "", "", err
	}
	return &u, tenant, apiKey, sessionToken, nil
}

// VerifyLogin checks email/password and returns the user's own tenant —
// callers should then call CreateSession to log the browser in.
func (s *Store) VerifyLogin(ctx context.Context, email, password string) (*User, *Tenant, error) {
	email = normalizeEmail(email)

	var u User
	var passwordHash string
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &passwordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tenant, err := s.GetTenantByOwner(ctx, u.ID)
	if err != nil {
		return nil, nil, err
	}
	return &u, tenant, nil
}

// CreateSession issues a fresh session token for an already-authenticated
// user (e.g. after VerifyLogin succeeds) — same one-time-plaintext
// contract as API keys: only the hash is persisted.
func (s *Store) CreateSession(ctx context.Context, userID string) (string, error) {
	return createSession(ctx, s.pool, userID)
}

func createSession(ctx context.Context, q querier, userID string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generating session token: %w", err)
	}
	_, err = q.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hashAPIKey(token), time.Now().Add(sessionTTL))
	if err != nil {
		return "", err
	}
	return token, nil
}

// GetSessionOwner resolves a session token to the user and tenant it
// belongs to, rejecting expired sessions the same way an unknown token is
// rejected — callers shouldn't be able to distinguish the two.
func (s *Store) GetSessionOwner(ctx context.Context, token string) (*User, *Tenant, error) {
	var u User
	var expiresAt time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.created_at, s.expires_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
	`, hashAPIKey(token)).Scan(&u.ID, &u.Email, &u.CreatedAt, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, nil, err
	}
	if time.Now().After(expiresAt) {
		return nil, nil, ErrSessionInvalid
	}

	tenant, err := s.GetTenantByOwner(ctx, u.ID)
	if err != nil {
		return nil, nil, err
	}
	return &u, tenant, nil
}

// DeleteSession logs a session out. Deleting an already-invalid token is
// not an error — logout is idempotent from the caller's point of view.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashAPIKey(token))
	return err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
