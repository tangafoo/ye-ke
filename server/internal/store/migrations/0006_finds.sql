-- TABLE > SCAN. Every located item any scan lane digs up lands in finds, once.
-- A scan reads finds first; lanes only scrape a cell whose sweep has gone stale.

CREATE TABLE IF NOT EXISTS finds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    source_kind TEXT NOT NULL, -- news | osm_police | tiktok | gmaps_review | threads
    source_id TEXT NOT NULL, -- stable id per source (dedup key, osm id, video id…)
    lat DOUBLE PRECISION NOT NULL,
    lng DOUBLE PRECISION NOT NULL,
    precision SMALLINT NOT NULL DEFAULT 0, -- news.Precision: 0 none … 4 street, 5 venue
    place_name TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    outlet TEXT NOT NULL DEFAULT '',
    published_at TIMESTAMPTZ,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now (),
    status TEXT NOT NULL DEFAULT 'reported', -- reported → corroborated → verified
    raw JSONB NOT NULL DEFAULT '{}', -- full scraped payload, so nothing is paid for twice
    UNIQUE (source_kind, source_id)
);

CREATE INDEX IF NOT EXISTS finds_latlng_idx ON finds (lat, lng);

-- The ledger: "this cell, this lane, swept at T". Remembers a sweep even when it
-- found nothing, which is what stops paying for the same empty spot twice.
CREATE TABLE IF NOT EXISTS sweeps (
    id BIGSERIAL PRIMARY KEY,
    cell TEXT NOT NULL, -- geohash
    lane TEXT NOT NULL,
    swept_at TIMESTAMPTZ NOT NULL DEFAULT now (),
    hits INTEGER NOT NULL DEFAULT 0,
    cost_usd NUMERIC(10, 4) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS sweeps_cell_lane_idx ON sweeps (cell, lane, swept_at DESC);

-- Who scanned where, when, which lanes actually ran, and what came back.
CREATE TABLE IF NOT EXISTS scans (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    lat DOUBLE PRECISION NOT NULL,
    lng DOUBLE PRECISION NOT NULL,
    radius_m DOUBLE PRECISION NOT NULL,
    lanes TEXT[] NOT NULL DEFAULT '{}',
    results INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now ()
);
