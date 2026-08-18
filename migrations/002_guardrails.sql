ALTER TABLE requests
    ADD COLUMN IF NOT EXISTS pii_redacted BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS injection_score NUMERIC(3,2) NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_requests_injection_score ON requests (injection_score) WHERE injection_score >= 0.5;
