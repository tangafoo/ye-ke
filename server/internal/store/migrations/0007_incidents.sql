-- One real-world event, many articles. finds are the articles; an incident
-- groups the ones about the same thing (same place, close dates, shared
-- words). The map shows one marker per incident.

CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    title TEXT NOT NULL, -- earliest article's headline
    place_name TEXT NOT NULL DEFAULT '',
    lat DOUBLE PRECISION NOT NULL, -- most precise member's pin
    lng DOUBLE PRECISION NOT NULL,
    precision SMALLINT NOT NULL DEFAULT 0,
    first_published TIMESTAMPTZ,
    last_published TIMESTAMPTZ,
    article_count INTEGER NOT NULL DEFAULT 0,
    outlet_count INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'reported', -- reported → corroborated (3+ outlets) → verified
    created_at TIMESTAMPTZ NOT NULL DEFAULT now (),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now ()
);

CREATE INDEX IF NOT EXISTS incidents_latlng_idx ON incidents (lat, lng);

ALTER TABLE finds
ADD COLUMN IF NOT EXISTS incident_id UUID REFERENCES incidents (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS finds_incident_idx ON finds (incident_id);
