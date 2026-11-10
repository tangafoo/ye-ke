package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"byakugan/internal/scan"
	"byakugan/internal/store"
)

const (
	scanDefaultRadiusM = 500.0
	scanMaxRadiusM     = 1000.0 // sweep cells are sized for ~500 m scans

	sweepBudget  = 90 * time.Second    // cold sweeps take ~30 s; Overpass mirrors can be slow
	newsCooldown = 24 * time.Hour      // stories keep coming; re-sweep daily
	osmCooldown  = 30 * 24 * time.Hour // police stations rarely move
)

type incidentItem struct {
	ID             string        `json:"id"`
	Title          string        `json:"title"`
	PlaceName      string        `json:"place_name"`
	Precision      int16         `json:"precision"`
	Lat            float64       `json:"lat"`
	Lng            float64       `json:"lng"`
	DistanceM      float64       `json:"distance_m"`
	FirstPublished *time.Time    `json:"first_published"`
	LastPublished  *time.Time    `json:"last_published"`
	OutletCount    int           `json:"outlet_count"`
	Status         string        `json:"status"`
	Articles       []articleItem `json:"articles"`
}

type articleItem struct {
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Outlet      string     `json:"outlet"`
	PublishedAt *time.Time `json:"published_at"`
}

type findItem struct {
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Outlet      string     `json:"outlet"`
	PlaceName   string     `json:"place_name"`
	Precision   int16      `json:"precision"`
	Lat         float64    `json:"lat"`
	Lng         float64    `json:"lng"`
	DistanceM   float64    `json:"distance_m"`
	PublishedAt *time.Time `json:"published_at"`
	Status      string     `json:"status"`
}

// GET /scan?lat=..&lng=..&radius=..
//
// TABLE > SCAN: lanes only scrape when this cell's sweep for that lane has gone
// stale. Whatever they find goes into finds; the response is always read back
// out of finds, so a cold scan and a warm scan return the same shape.
func (s *byakuganServer) handleScan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lat, errLat := strconv.ParseFloat(q.Get("lat"), 64)
	lng, errLng := strconv.ParseFloat(q.Get("lng"), 64)
	if errLat != nil || errLng != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		http.Error(w, "lat and lng must be valid floats", http.StatusBadRequest)
		return
	}

	radius := scanDefaultRadiusM
	if f, err := strconv.ParseFloat(q.Get("radius"), 64); err == nil && f > 0 {
		radius = min(f, scanMaxRadiusM)
	}

	region, ok := scan.RegionFor(lat, lng)
	if !ok {
		http.Error(w, "scan is not available in this area yet", http.StatusUnprocessableEntity)
		return
	}

	ctx := r.Context()
	cell := scan.Cell(lat, lng)

	// A sweep is paid-for work (Overpass, Google): it finishes and lands in the
	// table even if the phone hangs up mid-scan, so the next scan gets it free.
	sweepCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sweepBudget)
	lanes := s.runLanes(sweepCtx, cell, region, lat, lng, radius)
	cancel()
	if ctx.Err() != nil {
		log.Printf("[scan] client left during the sweep; results saved for the next scan (cell %s, lanes %v)", cell, lanes)
		return
	}

	finds, err := s.store.FindsNear(ctx, lat, lng, radius, store.FindsMaxResults)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "could not load scan results right now", http.StatusServiceUnavailable)
		return
	}

	var news, police []store.Find
	for _, f := range finds {
		if f.SourceKind == "news" {
			news = append(news, f)
		} else {
			police = append(police, f)
		}
	}
	s.clusterNews(ctx, news, len(lanes) > 0)

	incidents, err := s.incidentItems(ctx, news, lat, lng)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "could not load scan results right now", http.StatusServiceUnavailable)
		return
	}

	if err := s.store.LogScan(ctx, s.userID(r), lat, lng, radius, lanes, len(incidents)); err != nil {
		log.Printf("%v", err) // the ledger failing shouldn't fail the scan
	}

	log.Printf("[scan] lat=%.6f lng=%.6f r=%.0fm region=%s cell=%s lanes=%v → %d articles in %d incidents, %d police",
		lat, lng, radius, region.Code, cell, lanes, len(news), len(incidents), len(police))

	policeItems := make([]findItem, 0, len(police))
	for _, f := range police {
		policeItems = append(policeItems, findItem{
			Kind:      f.SourceKind,
			Title:     f.Title,
			URL:       f.URL,
			Outlet:    f.Outlet,
			PlaceName: f.PlaceName,
			Precision: f.Precision,
			Lat:       f.Lat,
			Lng:       f.Lng,
			DistanceM: f.DistanceM,
			Status:    f.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"lat":       lat,
		"lng":       lng,
		"radius":    radius,
		"region":    region.Code,
		"lanes":     lanes,
		"incidents": incidents,
		"police":    policeItems,
	})
}

// clusterNews groups the news finds around this scan into incidents. It runs
// when a lane just added finds, or when some find hasn't been grouped yet.
// The finds' IncidentID fields are updated in place.
//
// Known limit: an incident's counts are recomputed from the members visible to
// THIS scan, so scanning from a neighbouring point can briefly undercount an
// incident whose articles sit on both sides.
func (s *byakuganServer) clusterNews(ctx context.Context, news []store.Find, lanesRan bool) {
	if !lanesRan {
		pending := false
		for _, f := range news {
			if f.IncidentID == nil {
				pending = true
				break
			}
		}
		if !pending {
			return
		}
	}

	groups := scan.Cluster(news)
	var orphans []string
	for gi, p := range scan.PlanIncidents(news, groups) {
		id, err := s.store.SaveIncident(ctx, p.Incident, p.MemberIDs)
		if err != nil {
			log.Printf("%v", err)
			continue
		}
		for _, i := range groups[gi] {
			news[i].IncidentID = &id
		}
		orphans = append(orphans, p.Orphans...)
	}
	if err := s.store.DeleteEmptyIncidents(ctx, orphans); err != nil {
		log.Printf("%v", err)
	}
}

// incidentItems nests each incident's articles under it, nearest incident first.
func (s *byakuganServer) incidentItems(ctx context.Context, news []store.Find, lat, lng float64) ([]incidentItem, error) {
	members := map[string][]store.Find{}
	var ids []string
	for _, f := range news {
		if f.IncidentID == nil {
			continue
		}
		if _, seen := members[*f.IncidentID]; !seen {
			ids = append(ids, *f.IncidentID)
		}
		members[*f.IncidentID] = append(members[*f.IncidentID], f)
	}

	byID, err := s.store.IncidentsByID(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]incidentItem, 0, len(ids))
	for _, id := range ids {
		inc, ok := byID[id]
		if !ok {
			continue
		}
		fs := members[id]
		sort.Slice(fs, func(i, j int) bool { return earlier(fs[i].PublishedAt, fs[j].PublishedAt) })

		arts := make([]articleItem, 0, len(fs))
		for _, f := range fs {
			arts = append(arts, articleItem{Title: f.Title, URL: f.URL, Outlet: f.Outlet, PublishedAt: f.PublishedAt})
		}
		items = append(items, incidentItem{
			ID:             inc.ID,
			Title:          inc.Title,
			PlaceName:      inc.PlaceName,
			Precision:      inc.Precision,
			Lat:            inc.Lat,
			Lng:            inc.Lng,
			DistanceM:      scan.DistanceM(lat, lng, inc.Lat, inc.Lng),
			FirstPublished: inc.FirstPublished,
			LastPublished:  inc.LastPublished,
			OutletCount:    inc.OutletCount,
			Status:         inc.Status,
			Articles:       arts,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].DistanceM < items[j].DistanceM })
	return items, nil
}

// earlier orders dated before undated, oldest first.
func earlier(a, b *time.Time) bool {
	switch {
	case a == nil:
		return false
	case b == nil:
		return true
	}
	return a.Before(*b)
}

// runLanes scrapes whatever is stale for this cell and stores the results.
// It returns the lanes that actually ran. Lanes feed each other: OSM's nearby
// names are what the news lane searches for.
func (s *byakuganServer) runLanes(ctx context.Context, cell string, region scan.Region, lat, lng, radius float64) []string {
	var ran []string

	osmFresh, err := s.store.SweepFresh(ctx, cell, "osm_police", osmCooldown)
	if err != nil {
		log.Printf("%v", err)
		return ran
	}
	newsFresh, err := s.store.SweepFresh(ctx, cell, "news", newsCooldown)
	if err != nil {
		log.Printf("%v", err)
		return ran
	}
	if osmFresh && newsFresh {
		return ran // everything here is already in the table
	}

	around, err := s.osm.Around(ctx, lat, lng, int(radius))
	if err != nil {
		log.Printf("%v", err) // no sweep recorded, so the next scan retries
		return ran
	}
	police, targets := scan.FromOSM(around, lat, lng)

	if !osmFresh {
		n, err := s.store.UpsertFinds(ctx, police)
		if err != nil {
			log.Printf("%v", err)
		} else {
			s.recordSweep(ctx, cell, "osm_police", len(police))
			ran = append(ran, "osm_police")
			log.Printf("[scan] osm: %d police stations (%d new)", len(police), n)
		}
	}

	if !newsFresh {
		names := make([]string, len(targets))
		for i, t := range targets {
			names[i] = t.Name
		}
		log.Printf("[scan] news targets %v in %v", names, around.Areas)

		finds, ok := scan.NewsLane(ctx, s.news, region, targets, around.Areas)
		if !ok {
			log.Printf("[scan] news: every search failed, not recording sweep")
			return ran
		}
		n, err := s.store.UpsertFinds(ctx, finds)
		if err != nil {
			log.Printf("%v", err)
		} else {
			s.recordSweep(ctx, cell, "news", len(finds))
			ran = append(ran, "news")
			log.Printf("[scan] news: %d stories (%d new)", len(finds), n)
		}
	}
	return ran
}

func (s *byakuganServer) recordSweep(ctx context.Context, cell, lane string, hits int) {
	if err := s.store.RecordSweep(ctx, cell, lane, hits, 0); err != nil {
		log.Printf("%v", err)
	}
}
