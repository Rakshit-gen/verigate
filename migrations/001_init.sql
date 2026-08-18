CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    provider      TEXT NOT NULL,
    model         TEXT NOT NULL,
    prompt        TEXT NOT NULL,
    response      TEXT NOT NULL,
    latency_ms    INT NOT NULL,
    cache_hit     BOOLEAN NOT NULL DEFAULT false,
    tokens_in     INT,
    tokens_out    INT,
    cost_usd      NUMERIC(10,6),
    status        TEXT NOT NULL DEFAULT 'ok'
);

CREATE TABLE IF NOT EXISTS evals (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id    UUID NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    rubric        TEXT NOT NULL,
    score         NUMERIC(3,2) NOT NULL,
    reasoning     TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_requests_created_at ON requests (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_evals_created_at ON evals (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_evals_request_id ON evals (request_id);
