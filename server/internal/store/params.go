package store

import "time"

const (
	SearchK = 20
	TopK    = 5
	// 20-07-2026
	MaxDist           = 0.7100
	PingCooldown      = 5 * time.Minute
	PingMaxResults    = 500
	PingDefaultWindow = 30 * 24 * time.Hour
)
