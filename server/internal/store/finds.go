package store

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	FindsMaxResults = 200
	metersPerDegLat = 111_320.0

	// maxUncertaintyM is the widest uncertainty circle (town precision); the
	// bbox prefilter is widened by it so those finds aren't cut before the check.
	maxUncertaintyM = 2000.0
)

type Find struct {
	ID          string
	SourceKind  string
	SourceID    string
	Lat         float64
	Lng         float64
	Precision   int16
	PlaceName   string
	Title       string
	Body        string
	URL         string
	Outlet      string
	PublishedAt *time.Time
	FirstSeen   time.Time
	Status      string
	Raw         []byte // JSON; nil is stored as {}
	IncidentID  *string
	DistanceM   float64
}

// Existing rows are left alone: the first sighting wins, and re-scans cost nothing.
const upsertFindSQL = `
	INSERT INTO finds
		(source_kind, source_id, lat, lng, precision, place_name, title, body, url, outlet, published_at, raw)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (source_kind, source_id) DO NOTHING;
`

// UpsertFinds stores new finds in one round trip and reports how many were new.
func (s *Store) UpsertFinds(ctx context.Context, finds []Find) (int, error) {
	if len(finds) == 0 {
		return 0, nil
	}

	batch := &pgx.Batch{}
	for _, f := range finds {
		raw := f.Raw
		if raw == nil {
			raw = []byte("{}")
		}
		batch.Queue(upsertFindSQL,
			f.SourceKind, f.SourceID, f.Lat, f.Lng, f.Precision, f.PlaceName,
			f.Title, f.Body, f.URL, f.Outlet, f.PublishedAt, raw)
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	inserted := 0
	for range finds {
		tag, err := br.Exec()
		if err != nil {
			return inserted, fmt.Errorf("[finds] upsert: %w", err)
		}
		inserted += int(tag.RowsAffected())
	}
	return inserted, nil
}

// A find is pinned at its target's point, but the story could be anywhere in
// that target: ±50 m for a venue, ±500 m for a street or taman, ±2 km for a
// town. It's returned when that uncertainty circle overlaps the scan circle.
// bbox prefilter (uses finds_latlng_idx), then exact haversine distance.
// Most precise first, then nearest.
const findsNearSQL = `
	SELECT id, source_kind, source_id, lat, lng, precision, place_name, title, body,
	       url, outlet, published_at, first_seen, status, incident_id, dist
	FROM (
		SELECT *, 6371000 * 2 * asin(sqrt(
			power(sin(radians(lat - $1) / 2), 2) +
			cos(radians($1)) * cos(radians(lat)) * power(sin(radians(lng - $2) / 2), 2)
		)) AS dist
		FROM finds
		WHERE lat BETWEEN $4 AND $5
		  AND lng BETWEEN $6 AND $7
	) f
	WHERE dist <= $3 + CASE precision
		WHEN 5 THEN 50    -- venue
		WHEN 4 THEN 500   -- street / taman
		WHEN 3 THEN 2000  -- town / suburb
		ELSE 0
	END
	ORDER BY precision DESC, dist
	LIMIT $8;
`

func (s *Store) FindsNear(ctx context.Context, lat, lng, radiusM float64, limit int) ([]Find, error) {
	reach := radiusM + maxUncertaintyM
	dLat := reach / metersPerDegLat
	dLng := reach / (metersPerDegLat * math.Cos(lat*math.Pi/180))

	rows, err := s.pool.Query(ctx, findsNearSQL,
		lat, lng, radiusM, lat-dLat, lat+dLat, lng-dLng, lng+dLng, limit)
	if err != nil {
		return nil, fmt.Errorf("[finds] near: %w", err)
	}
	defer rows.Close()

	var out []Find
	for rows.Next() {
		var f Find
		if err := rows.Scan(
			&f.ID,
			&f.SourceKind,
			&f.SourceID,
			&f.Lat,
			&f.Lng,
			&f.Precision,
			&f.PlaceName,
			&f.Title,
			&f.Body,
			&f.URL,
			&f.Outlet,
			&f.PublishedAt,
			&f.FirstSeen,
			&f.Status,
			&f.IncidentID,
			&f.DistanceM); err != nil {
			return nil, fmt.Errorf("[finds] scan: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// SweepFresh reports whether this lane swept this cell within maxAge.
func (s *Store) SweepFresh(ctx context.Context, cell, lane string, maxAge time.Duration) (bool, error) {
	var fresh bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sweeps
			WHERE cell = $1 AND lane = $2 AND swept_at > now() - $3::interval
		);`, cell, lane, maxAge).Scan(&fresh)
	if err != nil {
		return false, fmt.Errorf("[sweeps] fresh check: %w", err)
	}
	return fresh, nil
}

func (s *Store) RecordSweep(ctx context.Context, cell, lane string, hits int, costUSD float64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sweeps (cell, lane, hits, cost_usd) VALUES ($1, $2, $3, $4);`,
		cell, lane, hits, costUSD)
	if err != nil {
		return fmt.Errorf("[sweeps] record: %w", err)
	}
	return nil
}

// LogScan records one scan. An empty userID logs it as anonymous.
func (s *Store) LogScan(ctx context.Context, userID string, lat, lng, radiusM float64, lanes []string, results int) error {
	var uid *string
	if userID != "" {
		uid = &userID
	}
	if lanes == nil {
		lanes = []string{}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO scans (user_id, lat, lng, radius_m, lanes, results)
		VALUES ($1, $2, $3, $4, $5, $6);`,
		uid, lat, lng, radiusM, lanes, results)
	if err != nil {
		return fmt.Errorf("[scans] log: %w", err)
	}
	return nil
}
