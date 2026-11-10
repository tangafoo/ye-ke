package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"byakugan/internal/mapillary"
	"byakugan/internal/scan"
	"byakugan/internal/store"
)

const (
	previewRadiusM   = 250.0 // photos of the streets right around the tap
	previewWant      = 6     // enough for a loading strip
	previewMinGapM   = 30.0  // don't show five frames of the same corner
	previewCellPrec  = 7     // ~150 m cells: one Mapillary sweep per block
	mapillaryCool    = 90 * 24 * time.Hour
	mapillaryBudget  = 45 * time.Second // ask + download + upload ~12 photos
	mapillaryFetchHM = 200.0            // half-size of the bbox asked of Mapillary
)

type streetImageItem struct {
	ID           string     `json:"id"`
	ThumbURL     string     `json:"thumb_url"` // ~256 px, our R2 copy
	ImageURL     string     `json:"image_url"` // ~1024 px, our R2 copy
	Lat          float64    `json:"lat"`
	Lng          float64    `json:"lng"`
	DistanceM    float64    `json:"distance_m"`
	CapturedAt   *time.Time `json:"captured_at"`
	CompassAngle *float64   `json:"compass_angle"`
	Creator      string     `json:"creator"` // Mapillary username (CC BY-SA 4.0: credit them in the UI)
	Source       string     `json:"source"`
}

// GET /scan/preview?lat=..&lng=..
//
// The loading strip: street photos around the tap, shown while /scan sweeps.
// Table first — stored photos come back in milliseconds. Only a block nobody
// has previewed yet goes to Mapillary, once; the best photos are copied into
// our R2 bucket and only those copies are ever served.
func (s *byakuganServer) handleScanPreview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, errLat := strconv.ParseFloat(q.Get("lat"), 64)
	lng, errLng := strconv.ParseFloat(q.Get("lng"), 64)
	if errLat != nil || errLng != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		http.Error(w, "lat and lng must be valid floats", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	imgs, err := s.store.StreetImagesNear(ctx, lat, lng, previewRadiusM, 60)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "could not load street images right now", http.StatusServiceUnavailable)
		return
	}

	from := "db"
	if len(imgs) < previewWant && s.streetPhotosOn() {
		// paid-for work outlives the request, like a sweep
		mctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), mapillaryBudget)
		if s.fetchStreetImages(mctx, lat, lng) {
			from = "mapillary"
			if fresh, err := s.store.StreetImagesNear(mctx, lat, lng, previewRadiusM, 60); err == nil {
				imgs = fresh
			}
		}
		cancel()
	}

	picked := pickStreetImages(imgs, previewWant)
	if len(picked) == 0 {
		from = "none"
	}

	items := make([]streetImageItem, 0, len(picked))
	for _, im := range picked {
		items = append(items, streetImageItem{
			ID: im.SourceID, ThumbURL: im.ThumbURL, ImageURL: im.ImageURL, Lat: im.Lat, Lng: im.Lng,
			DistanceM: im.DistanceM, CapturedAt: im.CapturedAt, CompassAngle: im.CompassAngle,
			Creator: im.Creator, Source: im.Source,
		})
	}
	log.Printf("[preview] lat=%.6f lng=%.6f → %d street images (%s)", lat, lng, len(items), from)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"lat":    lat,
		"lng":    lng,
		"from":   from, // db | mapillary | none
		"images": items,
	})
}

// streetPhotosOn: Mapillary photos are only ever served from our own copies,
// so fetching needs both the token and R2.
func (s *byakuganServer) streetPhotosOn() bool { return s.mapillary != nil && s.blob != nil }

// fetchStreetImages asks Mapillary for photos around the point, unless this
// block was already asked within the cooldown (even if it had none), copies
// the best ones into R2 and stores those. It reports whether Mapillary was asked.
func (s *byakuganServer) fetchStreetImages(ctx context.Context, lat, lng float64) bool {
	cell := scan.CellAt(lat, lng, previewCellPrec)
	fresh, err := s.store.SweepFresh(ctx, cell, "mapillary", mapillaryCool)
	if err != nil {
		log.Printf("%v", err)
		return false
	}
	if fresh {
		return false // asked before: "none here" is an answer too
	}

	got, err := s.mapillary.Near(ctx, lat, lng, mapillaryFetchHM, 500)
	if err != nil {
		log.Printf("%v", err) // no sweep recorded, so the next preview retries
		return false
	}

	var rows []store.StreetImage
	for _, im := range mapillary.Pick(got, 12, previewMinGapM) {
		row := toStreetImage(im)
		if err := s.host(ctx, &row); err != nil {
			log.Printf("%v", err) // skipped: we never store a photo we don't host
			continue
		}
		rows = append(rows, row)
	}
	if err := s.store.UpsertStreetImages(ctx, rows); err != nil {
		log.Printf("%v", err)
		return false
	}
	if err := s.store.RecordSweep(ctx, cell, "mapillary", len(rows), 0); err != nil {
		log.Printf("%v", err)
	}
	log.Printf("[preview] mapillary: %d photos in the box, hosted the best %d (cell %s)", len(got), len(rows), cell)
	return true
}

// host copies a photo's two sizes into R2 and points the row at our copies.
func (s *byakuganServer) host(ctx context.Context, row *store.StreetImage) error {
	thumb, err := s.copyToBlob(ctx, row.ThumbURL, "street/mapillary/"+row.SourceID+"_256.jpg")
	if err != nil {
		return err
	}
	full, err := s.copyToBlob(ctx, row.ImageURL, "street/mapillary/"+row.SourceID+"_1024.jpg")
	if err != nil {
		return err
	}
	row.ThumbURL, row.ImageURL = thumb, full
	return nil
}

func (s *byakuganServer) copyToBlob(ctx context.Context, from, key string) (string, error) {
	body, err := s.mapillary.Download(ctx, from)
	if err != nil {
		return "", err
	}
	return s.blob.Put(ctx, key, "image/jpeg", body)
}

func toStreetImage(im mapillary.Image) store.StreetImage {
	return store.StreetImage{
		Source: "mapillary", SourceID: im.ID, Lat: im.Lat, Lng: im.Lng,
		ThumbURL: im.Thumb256, ImageURL: im.Thumb1024, CapturedAt: im.CapturedAt,
		CompassAngle: im.CompassAngle, IsPano: im.IsPano, SequenceID: im.SequenceID, Creator: im.Creator,
		QualityScore: im.QualityScore, CameraType: im.CameraType,
	}
}

// pickStreetImages chooses the strip: the best photos, spread over different spots.
func pickStreetImages(imgs []store.StreetImage, n int) []store.StreetImage {
	byID := map[string]store.StreetImage{}
	asMap := make([]mapillary.Image, 0, len(imgs))
	for _, im := range imgs {
		byID[im.SourceID] = im
		asMap = append(asMap, mapillary.Image{
			ID: im.SourceID, Lat: im.Lat, Lng: im.Lng, CapturedAt: im.CapturedAt,
			IsPano: im.IsPano, SequenceID: im.SequenceID,
			QualityScore: im.QualityScore, CameraType: im.CameraType,
		})
	}
	var out []store.StreetImage
	for _, p := range mapillary.Pick(asMap, n, previewMinGapM) {
		out = append(out, byID[p.ID])
	}
	return out
}
