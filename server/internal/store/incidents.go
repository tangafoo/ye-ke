package store

import (
	"context"
	"fmt"
	"time"
)

// Incident is one real-world event that one or more finds (articles) report.
type Incident struct {
	ID             string
	Title          string
	PlaceName      string
	Lat            float64
	Lng            float64
	Precision      int16
	FirstPublished *time.Time
	LastPublished  *time.Time
	ArticleCount   int
	OutletCount    int
	Status         string
}

// SaveIncident creates the incident (when inc.ID is empty) or refreshes it, and
// points every member find at it, in one transaction. It returns the id.
func (s *Store) SaveIncident(ctx context.Context, inc Incident, memberIDs []string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("[incidents] begin: %w", err)
	}
	defer tx.Rollback(ctx) // no-op after Commit

	id := inc.ID
	if id == "" {
		err = tx.QueryRow(ctx, `
			INSERT INTO incidents
				(title, place_name, lat, lng, precision, first_published, last_published,
				 article_count, outlet_count, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id;`,
			inc.Title, inc.PlaceName, inc.Lat, inc.Lng, inc.Precision, inc.FirstPublished,
			inc.LastPublished, inc.ArticleCount, inc.OutletCount, inc.Status).Scan(&id)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE incidents SET
				title = $2, place_name = $3, lat = $4, lng = $5, precision = $6,
				first_published = $7, last_published = $8, article_count = $9,
				outlet_count = $10, status = $11, updated_at = now()
			WHERE id = $1;`,
			id, inc.Title, inc.PlaceName, inc.Lat, inc.Lng, inc.Precision, inc.FirstPublished,
			inc.LastPublished, inc.ArticleCount, inc.OutletCount, inc.Status)
	}
	if err != nil {
		return "", fmt.Errorf("[incidents] save: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE finds SET incident_id = $1 WHERE id = ANY($2::uuid[]);`, id, memberIDs); err != nil {
		return "", fmt.Errorf("[incidents] attach finds: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("[incidents] commit: %w", err)
	}
	return id, nil
}

// DeleteEmptyIncidents removes incidents left with no finds after a re-cluster
// merged them into another one.
func (s *Store) DeleteEmptyIncidents(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM incidents i
		WHERE i.id = ANY($1::uuid[])
		  AND NOT EXISTS (SELECT 1 FROM finds f WHERE f.incident_id = i.id);`, ids)
	if err != nil {
		return fmt.Errorf("[incidents] delete empty: %w", err)
	}
	return nil
}

func (s *Store) IncidentsByID(ctx context.Context, ids []string) (map[string]Incident, error) {
	out := map[string]Incident{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, place_name, lat, lng, precision, first_published, last_published,
		       article_count, outlet_count, status
		FROM incidents WHERE id = ANY($1::uuid[]);`, ids)
	if err != nil {
		return nil, fmt.Errorf("[incidents] by id: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var i Incident
		if err := rows.Scan(&i.ID, &i.Title, &i.PlaceName, &i.Lat, &i.Lng, &i.Precision,
			&i.FirstPublished, &i.LastPublished, &i.ArticleCount, &i.OutletCount, &i.Status); err != nil {
			return nil, fmt.Errorf("[incidents] scan: %w", err)
		}
		out[i.ID] = i
	}
	return out, rows.Err()
}
