package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

func (s *byakuganServer) handleScan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lat, errLat := strconv.ParseFloat(q.Get("lat"), 64)
	lng, errLng := strconv.ParseFloat(q.Get("lng"), 64)

	if errLat != nil || errLng != nil {
		http.Error(w, "lat and lng must be floats", http.StatusBadRequest)
		return
	}

	radius := 50.0
	if f, err := strconv.ParseFloat(q.Get("radius"), 64); err == nil && f > 0 {
		radius = f
	}

	log.Printf("[scan] hit lat=%.6f lng=%.6f radius=%.0fm", lat, lng, radius)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"lat":      lat,
		"lng":      lng,
		"radius":   radius,
		"articles": []any{},
	})

}
