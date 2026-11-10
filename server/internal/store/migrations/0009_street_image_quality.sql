-- Rank street photos by quality, not recency.
ALTER TABLE street_images ADD COLUMN IF NOT EXISTS quality_score DOUBLE PRECISION;
ALTER TABLE street_images ADD COLUMN IF NOT EXISTS camera_type TEXT NOT NULL DEFAULT '';
