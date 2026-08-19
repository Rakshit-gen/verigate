#!/bin/sh
set -eu

# Migrations are idempotent (CREATE TABLE/ADD COLUMN IF NOT EXISTS), so
# running the full set on every container start is safe — this replaces a
# Render preDeployCommand, which free-tier services can't use.
for f in migrations/*.sql; do
  psql "$DATABASE_URL" -f "$f"
done

exec ./gateway
