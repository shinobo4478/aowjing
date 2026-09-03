-- Database schema. This file is the single source of truth for the DDL:
--   * docker-compose mounts it as a Postgres init script for local dev
--   * sqlc reads it to type-check queries and generate Go models
--
-- When a real migration tool is added later, its first migration should match
-- this file exactly.

CREATE TABLE IF NOT EXISTS profiles (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text        NOT NULL,
    niche       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- Login sessions for the single admin user. We store only a SHA-256 hash of
-- the session token, so a leak of this table does not expose live sessions.
CREATE TABLE IF NOT EXISTS sessions (
    token_hash bytea       PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);
