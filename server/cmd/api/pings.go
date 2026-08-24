package main

import (
	"byakugan/internal/store"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type pingCreateRequest struct {
	Kind string  `json:"kind"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

type pingItem struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	CreatedAt time.Time `json:"created_at"`
	Mine      bool      `json:"mine"`
}

func toPingItem(p store.Ping, me string) pingItem {
	return pingItem{
		ID:        p.ID,
		Kind:      p.Kind,
		Lat:       p.Lat,
		Lng:       p.Lng,
		CreatedAt: p.CreatedAt,
		Mine:      me != "" && p.UserID == me,
	}
}

func (s *byakuganServer) handlePingCreate(w http.ResponseWriter, r *http.Request) {
	me := s.userID(r)
	if me == "" {
		// No user - save in local phone DB instead (use syncReq to sync unclaimed pings to a user if user decides to create an account)
		return
	}

	var req pingCreateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "request does not follow pingCreateRequest json schema", http.StatusBadRequest)
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		http.Error(w, "invalid location: not on Earth", http.StatusBadRequest)
		return
	}

	p, err := s.store.AddPing(r.Context(), me, req.Kind, req.Lat, req.Lng)
	if errors.Is(err, store.ErrPingKindInvalid) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, store.ErrPingCooldown) {
		http.Error(w, "easy - one ping every 3 mins", http.StatusTooManyRequests)
		return
	}

	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "could not record a ping right now", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err = json.NewEncoder(w).Encode(toPingItem(p, me)); err != nil {
		log.Printf("[ping] could not encode ping response: %v", err)
		return
	}
}

func (s *byakuganServer) handlePingList(w http.ResponseWriter, r *http.Request) {
	b, err := parseBBox(r.URL.Query().Get("bbox"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	since := time.Now().Add(-store.PingDefaultWindow)
	if raw := r.URL.Query().Get("since"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			http.Error(w, "invalid date format - try RFC3339", http.StatusBadRequest)
			return
		}
		since = t
	}

	pings, err := s.store.PingsInBBox(r.Context(), b.minLat, b.minLng, b.maxLat, b.maxLng, since, store.PingMaxResults)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "could not load pings right now", http.StatusServiceUnavailable)
		return
	}

	me := s.userID(r)

	items := make([]pingItem, 0, len(pings))
	for _, p := range pings {
		items = append(items, toPingItem(p, me))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]pingItem{"pings": items})
}

type bbox struct {
	minLat, minLng, maxLat, maxLng float64
}

func parseBBox(raw string) (bbox, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return bbox{}, errors.New("bbox have 4 items and follow the order minLat, minLng, maxLat, maxLng")
	}

	nums := make([]float64, 4)
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return bbox{}, errors.New("bbox values must be floats (float64)")
		}
		nums[i] = f
	}

	b := bbox{minLat: nums[0], minLng: nums[1], maxLat: nums[2], maxLng: nums[3]}
	if b.minLat >= b.maxLat || b.minLng >= b.maxLng {
		return bbox{}, errors.New("bbox min cannot be equal to or greater than max")
	}
	if b.minLat < -90 || b.maxLat > 90 || b.minLng < -180 || b.maxLng > 180 {
		return bbox{}, errors.New("bbox is not on this planet")
	}
	return b, nil
}
