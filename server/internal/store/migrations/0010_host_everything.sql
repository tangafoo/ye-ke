-- Every photo we show is our own R2 copy: street photos (Mapillary, CC BY-SA)
-- and news photos (a credited thumbnail of the article's og:image). No more
-- expiring third-party links, so no refresh bookkeeping.
-- (no-ops on a fresh DB; clean up dev databases that ran an earlier 0008/0009)
ALTER TABLE street_images DROP COLUMN IF EXISTS thumb_fetched_at;
ALTER TABLE street_images DROP COLUMN IF EXISTS hosted;

-- image_source_url: the outlet's og:image (credit + takedown trail).
-- image_url: our hosted thumbnail. image_hosted_at marks the hosting attempt.
ALTER TABLE finds ADD COLUMN IF NOT EXISTS image_source_url TEXT NOT NULL DEFAULT '';
ALTER TABLE finds ADD COLUMN IF NOT EXISTS image_hosted_at TIMESTAMPTZ;

UPDATE finds SET image_source_url = image_url, image_url = ''
WHERE image_url <> '' AND image_source_url = '';

CREATE INDEX IF NOT EXISTS finds_host_pending_idx ON finds (published_at DESC)
WHERE image_source_url <> '' AND image_hosted_at IS NULL;
