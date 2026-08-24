CREATE TABLE IF NOT EXISTS pings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    lat         DOUBLE PRECISION NOT NULL,
    lng         DOUBLE PRECISION NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS pings_bbox_idx ON pings (lat, lng);

CREATE INDEX IF NOT EXISTS pings_user_recent_idx ON pings (user_id, created_at DESC);