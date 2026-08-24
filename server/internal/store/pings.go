package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrPingCooldown = errors.New("ping cooldown active")

type Ping struct {
	ID        string
	UserID    string
	Kind      string
	Lat       float64
	Lng       float64
	CreatedAt time.Time
}

const addPingSQL = `
	INSERT INTO pings (user_id, kind, lat, lng)
	SELECT $1, $2, $3, $4
	WHERE NOT EXISTS (
		SELECT 1 FROM pings WHERE user_id = $1 AND created_at > now() - $5::interval
	)
	RETURNING id, created_at;
`

func (s *Store) AddPing(ctx context.Context, userID, kind string, lat, lng float64) (Ping, error) {
	p := Ping{UserID: userID, Kind: kind, Lat: lat, Lng: lng}
	err := s.pool.QueryRow(ctx, addPingSQL, userID, kind, lat, lng, PingCooldown).Scan(&p.ID, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Ping{}, ErrPingCooldown
	}
	if err != nil {
		return Ping{}, fmt.Errorf("[ping] insert: %w", err)
	}
	return p, nil
}

const getPingsSQL = `
	SELECT id, user_id, kind, lat, lng, created_at
	FROM pings
	WHERE lat BETWEEN $1 and $2
	  AND lng BETWEEN $3 and $4
	  AND created_at > $5
	ORDER BY created_at DESC
	LIMIT $6;
`

func (s *Store) PingsInBBox(ctx context.Context, minLat, minLng, maxLat, maxLng float64, since time.Time, limit int) ([]Ping, error) {
	rows, err := s.pool.Query(ctx, getPingsSQL, minLat, maxLat, minLng, maxLng, since, limit)
	if err != nil {
		return nil, fmt.Errorf("[ping] retrieval: %w", err)
	}
	defer rows.Close()

	var pings []Ping
	for rows.Next() {
		var p Ping
		if err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Kind,
			&p.Lat,
			&p.Lng,
			&p.CreatedAt); err != nil {
			return nil, fmt.Errorf("[ping] scan: %w", err)
		}
		pings = append(pings, p)
	}

	return pings, rows.Err()
}
