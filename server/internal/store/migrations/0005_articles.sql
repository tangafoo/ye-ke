CREATE TABLE
    IF NOT EXISTS operations (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        op_code TEXT NOT NULL UNIQUE,
        display_name TEXT NOT NULL,
        lat DOUBLE PRECISION,
        lng DOUBLE PRECISION,
        precision SMALLINT NOT NULL DEFAULT 0,
        status TEXT NOT NULL DEFAULT 'reported',
        first_seen TIMESTAMPTZ NOT NULL DEFAULT now (),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT now ()
    );

CREATE TABLE
    IF NOT EXISTS articles (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        dedup_key TEXT NOT NULL UNIQUE,
        url TEXT NOT NULL,
        title TEXT NOT NULL,
        outlet TEXT NOT NULL,
        outlet_url TEXT NOT NULL DEFAULT '',
        lang TEXT NOT NULL DEFAULT 'ms',
        category TEXT NOT NULL DEFAULT 'unknown',
        snippet TEXT NOT NULL DEFAULT '',
        caption TEXT NOT NULL DEFAULT '',
        place_name TEXT NOT NULL DEFAULT '',
        lat DOUBLE PRECISION,
        lng DOUBLE PRECISION,
        precision SMALLINT NOT NULL DEFAULT 0,
        source_lane TEXT NOT NULL,
        published_at TIMESTAMPTZ,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now ()
    );

CREATE TABLE
    IF NOT EXISTS article_operations (
        PRIMARY KEY (article_id, operation_id),
        article_id UUID NOT NULL REFERENCES articles (id) ON DELETE CASCADE,
        operation_id UUID NOT NULL REFERENCES operations (id) ON DELETE CASCADE,
        mentions INTEGER NOT NULL DEFAULT 1
    );

CREATE INDEX IF NOT EXISTS articles_bbox_idx ON articles (lat, lng);

CREATE INDEX IF NOT EXISTS articles_recent_idx ON articles (published_at DESC);

CREATE INDEX IF NOT EXISTS operations_bbox_idx ON operations (lat, lng);

CREATE INDEX IF NOT EXISTS artops_by_op_idx ON article_operations (operation_id);