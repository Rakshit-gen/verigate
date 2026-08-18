package store

import (
	"context"
	"strings"
	"testing"
)

func TestCreateAndLookupTenant(t *testing.T) {
	s := testStore(t)
	defer s.Close()
	ctx := context.Background()

	name := "test-tenant-" + t.Name()
	tenant, apiKey, err := s.CreateTenant(ctx, name, 42)
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	defer s.pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenant.ID)

	if !strings.HasPrefix(apiKey, "vg_") {
		t.Errorf("expected generated key to have the vg_ prefix, got %q", apiKey)
	}
	if tenant.RateLimitRPM != 42 {
		t.Errorf("expected rate limit 42, got %d", tenant.RateLimitRPM)
	}

	found, err := s.GetTenantByAPIKey(ctx, apiKey)
	if err != nil {
		t.Fatalf("GetTenantByAPIKey with the real key: %v", err)
	}
	if found.ID != tenant.ID {
		t.Errorf("expected lookup to return the same tenant, got a different ID")
	}
}

func TestGetTenantByAPIKey_WrongKeyNotFound(t *testing.T) {
	s := testStore(t)
	defer s.Close()
	ctx := context.Background()

	name := "test-tenant-wrongkey-" + t.Name()
	tenant, _, err := s.CreateTenant(ctx, name, 60)
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	defer s.pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenant.ID)

	_, err = s.GetTenantByAPIKey(ctx, "vg_definitely_not_the_real_key")
	if err != ErrTenantNotFound {
		t.Errorf("expected ErrTenantNotFound for a wrong key, got %v", err)
	}
}

func TestCreateTenant_PlaintextKeyNeverStored(t *testing.T) {
	s := testStore(t)
	defer s.Close()
	ctx := context.Background()

	name := "test-tenant-plaintext-" + t.Name()
	tenant, apiKey, err := s.CreateTenant(ctx, name, 60)
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	defer s.pool.Exec(ctx, "DELETE FROM tenants WHERE id = $1", tenant.ID)

	var storedHash string
	if err := s.pool.QueryRow(ctx, "SELECT api_key_hash FROM tenants WHERE id = $1", tenant.ID).Scan(&storedHash); err != nil {
		t.Fatalf("reading back stored hash: %v", err)
	}
	if storedHash == apiKey {
		t.Fatal("the plaintext API key must never be stored as-is")
	}
	if len(storedHash) != 64 { // sha256 hex digest length
		t.Errorf("expected a 64-char sha256 hex digest, got %d chars", len(storedHash))
	}
}
