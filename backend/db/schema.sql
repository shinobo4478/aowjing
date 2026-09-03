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
    -- Which Generator this profile defaults to (see internal/generate).
    -- Stores the setting only — nothing reads it automatically yet.
    provider    text        NOT NULL DEFAULT 'text',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

-- A channel is one distribution target under a profile (a YouTube channel, a
-- TikTok account, ...). Phase 1 only stores the record — no platform APIs yet.
-- Deleting a profile removes its channels.
CREATE TABLE IF NOT EXISTS channels (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    name        text        NOT NULL,
    platform    text        NOT NULL,
    handle      text        NOT NULL DEFAULT '',
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS channels_profile_id_idx ON channels (profile_id);

-- Reusable prompt templates associated with a profile. `body` is the template
-- text; Phase 1 stores it verbatim (no variable substitution yet).
CREATE TABLE IF NOT EXISTS prompt_templates (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    name        text        NOT NULL,
    body        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS prompt_templates_profile_id_idx ON prompt_templates (profile_id);

-- One run of a prompt template through the AI provider. An immutable record:
-- the exact prompt that was sent, what came back, and which provider/model
-- produced it. Kept even if the source template is later deleted (the link is
-- nulled); removed with its profile.
CREATE TABLE IF NOT EXISTS generations (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id         uuid        NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    prompt_template_id uuid            REFERENCES prompt_templates (id) ON DELETE SET NULL,
    input_prompt       text        NOT NULL,
    output             text        NOT NULL DEFAULT '',
    status             text        NOT NULL,
    error              text        NOT NULL DEFAULT '',
    provider           text        NOT NULL,
    model              text        NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS generations_profile_id_idx ON generations (profile_id);

-- Global key/value settings — provider credentials and the like. One row per
-- key, not duplicated per profile. The API whitelists which keys it accepts.
CREATE TABLE IF NOT EXISTS settings (
    key        text        PRIMARY KEY,
    value      text        NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Login sessions for the single admin user. We store only a SHA-256 hash of
-- the session token, so a leak of this table does not expose live sessions.
CREATE TABLE IF NOT EXISTS sessions (
    token_hash bytea       PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_expires_at_idx ON sessions (expires_at);
