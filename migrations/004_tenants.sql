CREATE TABLE IF NOT EXISTS tenants (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL UNIQUE,
    api_key_hash   TEXT NOT NULL UNIQUE, -- sha256 hex of the real key; the plaintext key is shown once at creation and never stored
    rate_limit_rpm INT NOT NULL DEFAULT 60,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants(id);

CREATE INDEX IF NOT EXISTS idx_requests_tenant_id ON requests (tenant_id);
