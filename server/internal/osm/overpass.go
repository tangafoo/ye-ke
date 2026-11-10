// Package osm reads OpenStreetMap through the public Overpass API: what is
// physically around a point (police stations, named streets, landmarks) and
// which admin areas contain it. Free, no key; be polite with the User-Agent.
package osm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultUserAgent = "YeKe/0.1 (+https://github.com/tangafoo/ye-ke)"
	maxBodyBytes     = 16 << 20
)

// mirrors are public Overpass instances, tried in order. The main one returns
// 502/504 when it's overloaded, which is often.
var mirrors = []string{
	"https://overpass-api.de/api/interpreter",
	"https://overpass.private.coffee/api/interpreter",
	"https://overpass.kumi.systems/api/interpreter",
}

type Client struct {
	http      *http.Client
	endpoints []string
	userAgent string
}

func New() *Client {
	return &Client{
		http:      &http.Client{Timeout: 40 * time.Second},
		endpoints: mirrors,
		userAgent: defaultUserAgent,
	}
}

// errRetryable marks a mirror failure worth trying the next mirror for.
type errRetryable struct{ err error }

func (e errRetryable) Error() string { return e.err.Error() }

// Element is one OSM node, way or relation. Ways and relations come back with a
// Center instead of their own Lat/Lon (we ask for "out center").
type Element struct {
	Type   string            `json:"type"` // node | way | relation | area
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Center *LatLon           `json:"center"`
	Tags   map[string]string `json:"tags"`
}

type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Point is where the element sits: its own coordinate, or its center for ways.
func (e Element) Point() (lat, lng float64, ok bool) {
	if e.Center != nil {
		return e.Center.Lat, e.Center.Lon, true
	}
	if e.Type == "node" {
		return e.Lat, e.Lon, true
	}
	return 0, 0, false
}

// Key is a stable id across scans, e.g. "way/123456".
func (e Element) Key() string { return fmt.Sprintf("%s/%d", e.Type, e.ID) }

type Around struct {
	Features []Element // police, named amenities/landmarks, named roads, neighbourhoods
	Areas    []Area    // containing admin areas, most local first
}

// Area is an admin boundary containing the point. Level follows OSM
// admin_level: 4 = state / federal territory … 8 = town.
type Area struct {
	Name  string
	Level int
}

// Around returns what's within radiusM of the point, plus the admin areas the
// point is inside. One request.
func (c *Client) Around(ctx context.Context, lat, lng float64, radiusM int) (Around, error) {
	at := fmt.Sprintf("(around:%d,%.6f,%.6f)", radiusM, lat, lng)
	q := fmt.Sprintf(`[out:json][timeout:30];
(
  nwr["amenity"="police"]%[1]s;
  nwr["name"]["amenity"~"^(marketplace|bus_station|place_of_worship|school|college|university|hospital|nightclub|bar|pub|cinema|townhall|courthouse|prison|community_centre|fuel|food_court|casino)$"]%[1]s;
  nwr["name"]["tourism"~"^(attraction|museum|viewpoint)$"]%[1]s;
  nwr["name"]["shop"="mall"]%[1]s;
  nwr["name"]["railway"="station"]%[1]s;
  way["highway"]["name"]%[1]s;
  nwr["place"~"^(neighbourhood|suburb|quarter|village|hamlet)$"]["name"]%[1]s;
)->.f;
.f out center tags qt;
is_in(%.6f,%.6f)->.a;
area.a["admin_level"~"^[4-8]$"]["name"];
out tags;`, at, lat, lng)

	var (
		elements []Element
		lastErr  error
	)
	for _, endpoint := range c.endpoints {
		elements, lastErr = c.query(ctx, endpoint, q)
		if lastErr == nil {
			break
		}
		var retry errRetryable
		if !errors.As(lastErr, &retry) || ctx.Err() != nil {
			return Around{}, lastErr
		}
		log.Printf("%v; trying next mirror", lastErr)
	}
	if lastErr != nil {
		return Around{}, lastErr
	}
	body := struct{ Elements []Element }{elements}

	var out Around
	for _, e := range body.Elements {
		if e.Type == "area" {
			n := e.Tags["name"]
			lvl, err := strconv.Atoi(e.Tags["admin_level"])
			if n != "" && err == nil {
				out.Areas = append(out.Areas, Area{Name: n, Level: lvl})
			}
			continue
		}
		out.Features = append(out.Features, e)
	}

	// higher admin_level = more local (8 town … 4 state); most local first
	sort.SliceStable(out.Areas, func(i, j int) bool { return out.Areas[i].Level > out.Areas[j].Level })
	return out, nil
}

func (c *Client) query(ctx context.Context, endpoint, q string) ([]Element, error) {
	form := url.Values{"data": {q}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("[osm] build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, errRetryable{fmt.Errorf("[osm] %s: %w", endpoint, err)}
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusOK:
	case res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500:
		return nil, errRetryable{fmt.Errorf("[osm] %s: status %d", endpoint, res.StatusCode)}
	default:
		return nil, fmt.Errorf("[osm] %s: status %d", endpoint, res.StatusCode)
	}

	var body struct {
		Elements []Element `json:"elements"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBodyBytes)).Decode(&body); err != nil {
		return nil, errRetryable{fmt.Errorf("[osm] %s decode: %w", endpoint, err)}
	}
	return body.Elements, nil
}
