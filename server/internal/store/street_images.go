package store

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

// StreetImage is one street-level photo near a scanned point. ThumbURL and
// ImageURL are our own R2 copies, so they never expire.
type StreetImage struct {
	ID           string
	Source       string
	SourceID     string
	Lat          float64
	Lng          float64
	ThumbURL     string
	ImageURL     string
	CapturedAt   *time.Time
	CompassAngle *float64
	IsPano       bool
	SequenceID   string
	Creator      string
	QualityScore *float64
	CameraType   string
	DistanceM    float64
}

// UpsertStreetImages stores new photos; a photo already stored is left as is.
func (s *Store) UpsertStreetImages(ctx context.Context, imgs []StreetImage) error {
	if len(imgs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, im := range imgs {
		batch.Queue(`
			INSERT INTO street_images
				(source, source_id, lat, lng, thumb_url, image_url,
				 captured_at, compass_angle, is_pano, sequence_id, creator,
				 quality_score, camera_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (source, source_id) DO NOTHING;`,
			im.Source, im.SourceID, im.Lat, im.Lng, im.ThumbURL, im.ImageURL,
			im.CapturedAt, im.CompassAngle, im.IsPano, im.SequenceID, im.Creator,
			im.QualityScore, im.CameraType)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range imgs {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("[street_images] upsert: %w", err)
		}
	}
	return nil
}

// StreetImagesNear returns stored photos within radiusM, nearest first.
func (s *Store) StreetImagesNear(ctx context.Context, lat, lng, radiusM float64, limit int) ([]StreetImage, error) {
	dLat := radiusM / metersPerDegLat
	dLng := radiusM / (metersPerDegLat * math.Cos(lat*math.Pi/180))

	rows, err := s.pool.Query(ctx, `
		SELECT id, source, source_id, lat, lng, thumb_url, image_url,
		       captured_at, compass_angle, is_pano, sequence_id, creator,
		       quality_score, camera_type, dist
		FROM (
			SELECT *, 6371000 * 2 * asin(sqrt(
				power(sin(radians(lat - $1) / 2), 2) +
				cos(radians($1)) * cos(radians(lat)) * power(sin(radians(lng - $2) / 2), 2)
			)) AS dist
			FROM street_images
			WHERE lat BETWEEN $4 AND $5 AND lng BETWEEN $6 AND $7
		) s
		WHERE dist <= $3
		ORDER BY dist
		LIMIT $8;`,
		lat, lng, radiusM, lat-dLat, lat+dLat, lng-dLng, lng+dLng, limit)
	if err != nil {
		return nil, fmt.Errorf("[street_images] near: %w", err)
	}
	defer rows.Close()

	var out []StreetImage
	for rows.Next() {
		var im StreetImage
		if err := rows.Scan(&im.ID, &im.Source, &im.SourceID, &im.Lat, &im.Lng, &im.ThumbURL,
			&im.ImageURL, &im.CapturedAt, &im.CompassAngle, &im.IsPano,
			&im.SequenceID, &im.Creator, &im.QualityScore, &im.CameraType,
			&im.DistanceM); err != nil {
			return nil, fmt.Errorf("[street_images] scan: %w", err)
		}
		out = append(out, im)
	}
	return out, rows.Err()
}
