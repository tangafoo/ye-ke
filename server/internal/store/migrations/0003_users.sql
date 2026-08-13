-- 0003: users. Anonymous-first identity — the app registers a user on first
-- launch with no signup screen, and OAuth credentials attach to the SAME row
-- later (user_identities), so linking never changes the user id and never
-- migrates data. Same idempotent style as 0001/0002.

CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind       TEXT NOT NULL DEFAULT 'anonymous',  -- anonymous | registered
    token_hash TEXT NOT NULL UNIQUE,               -- sha256 hex of the bearer token; the token itself is never stored
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- OAuth credentials live beside users, not on them: one row per linked
-- provider, keyed by the provider's stable subject id. Linking inserts a row
-- and flips users.kind to 'registered'; the anonymous token keeps working.
CREATE TABLE IF NOT EXISTS user_identities (
    provider   TEXT NOT NULL,                      -- google | apple
    subject    TEXT NOT NULL,                      -- the provider's stable user id ("sub" claim)
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);

CREATE INDEX IF NOT EXISTS user_identities_user_idx ON user_identities (user_id);
