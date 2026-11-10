-- Article cards: the real outlet link (Google News gives a per-query redirect)
-- and the article's own photo (og:image). Filled in the background after a
-- sweep; enriched_at marks an attempt, successful or not, so it isn't retried.
ALTER TABLE finds ADD COLUMN IF NOT EXISTS article_url TEXT NOT NULL DEFAULT '';
ALTER TABLE finds ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';
ALTER TABLE finds ADD COLUMN IF NOT EXISTS enriched_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS finds_enrich_pending_idx ON finds (published_at DESC)
WHERE source_kind = 'news' AND enriched_at IS NULL;

-- Street-level photos near scanned points (Mapillary, CC BY-SA 4.0: credit the
-- creator). URLs point at our own R2 copies.
CREATE TABLE IF NOT EXISTS street_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    source TEXT NOT NULL DEFAULT 'mapillary',
    source_id TEXT NOT NULL,
    lat DOUBLE PRECISION NOT NULL,
    lng DOUBLE PRECISION NOT NULL,
    thumb_url TEXT NOT NULL DEFAULT '', -- ~256 px, for the loading strip
    image_url TEXT NOT NULL DEFAULT '', -- ~1024 px
    captured_at TIMESTAMPTZ,
    compass_angle DOUBLE PRECISION,
    is_pano BOOLEAN NOT NULL DEFAULT false,
    sequence_id TEXT NOT NULL DEFAULT '',
    creator TEXT NOT NULL DEFAULT '',
    first_seen TIMESTAMPTZ NOT NULL DEFAULT now (),
    UNIQUE (source, source_id)
);

CREATE INDEX IF NOT EXISTS street_images_latlng_idx ON street_images (lat, lng);
