// Package mapillary fetches street-level photos near a point from Mapillary's
// Graph API (v4). Imagery is CC BY-SA 4.0: show "© Mapillary" and the creator.
// Mapillary's thumbnail URLs are signed and expire, so callers Download the
// bytes and serve their own copy.
package mapillary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	graphURL = "https://graph.mapillary.com"
	fields   = "id,thumb_256_url,thumb_1024_url,captured_at,compass_angle,geometry,is_pano,sequence,creator,quality_score,camera_type"
	maxBody  = 8 << 20
	maxImage = 3 << 20
)

var ErrNoToken = errors.New("[mapillary] no access token")

type Client struct {
	token   string
	http    *http.Client
	baseURL string
}

// New returns nil when token is empty, so callers can treat Mapillary as off.
func New(token string) *Client {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return &Client{token: token, http: &http.Client{Timeout: 20 * time.Second}, baseURL: graphURL}
}

type Image struct {
	ID           string
	Lat, Lng     float64
	Thumb256     string
	Thumb1024    string
	CapturedAt   *time.Time
	CompassAngle *float64
	IsPano       bool
	SequenceID   string
	Creator      string
	QualityScore *float64 // Mapillary's own 0–1 image quality estimate
	CameraType   string   // perspective | fisheye | spherical
}

type apiImage struct {
	ID           string   `json:"id"`
	Thumb256     string   `json:"thumb_256_url"`
	Thumb1024    string   `json:"thumb_1024_url"`
	CapturedAt   *int64   `json:"captured_at"` // unix milliseconds
	CompassAngle *float64 `json:"compass_angle"`
	IsPano       bool     `json:"is_pano"`
	Sequence     string   `json:"sequence"`
	Geometry     struct {
		Coordinates []float64 `json:"coordinates"` // [lng, lat]
	} `json:"geometry"`
	Creator struct {
		Username string `json:"username"`
	} `json:"creator"`
	QualityScore *float64 `json:"quality_score"`
	CameraType   string   `json:"camera_type"`
}

func (a apiImage) toImage() (Image, bool) {
	if len(a.Geometry.Coordinates) != 2 || a.ID == "" {
		return Image{}, false
	}
	im := Image{
		ID:           a.ID,
		Lng:          a.Geometry.Coordinates[0],
		Lat:          a.Geometry.Coordinates[1],
		Thumb256:     a.Thumb256,
		Thumb1024:    a.Thumb1024,
		CompassAngle: a.CompassAngle,
		IsPano:       a.IsPano,
		SequenceID:   a.Sequence,
		Creator:      a.Creator.Username,
		QualityScore: a.QualityScore,
		CameraType:   a.CameraType,
	}
	if a.CapturedAt != nil {
		t := time.UnixMilli(*a.CapturedAt).UTC()
		im.CapturedAt = &t
	}
	return im, true
}

// Near returns up to limit photos within halfM metres of the point (a square
// bbox; Mapillary rejects large boxes, so keep halfM in the low hundreds).
func (c *Client) Near(ctx context.Context, lat, lng, halfM float64, limit int) ([]Image, error) {
	if c == nil {
		return nil, ErrNoToken
	}
	dLat := halfM / 111_320.0
	dLng := halfM / (111_320.0 * math.Cos(lat*math.Pi/180))

	q := url.Values{}
	q.Set("fields", fields)
	q.Set("bbox", fmt.Sprintf("%.6f,%.6f,%.6f,%.6f", lng-dLng, lat-dLat, lng+dLng, lat+dLat))
	q.Set("limit", fmt.Sprint(limit))

	var body struct {
		Data []apiImage `json:"data"`
	}
	if err := c.getJSON(ctx, c.baseURL+"/images?"+q.Encode(), &body); err != nil {
		return nil, err
	}
	out := make([]Image, 0, len(body.Data))
	for _, a := range body.Data {
		if im, ok := a.toImage(); ok {
			out = append(out, im)
		}
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, u string, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("[mapillary] build request: %w", err)
	}
	req.Header.Set("Authorization", "OAuth "+c.token)

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("[mapillary] request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("[mapillary] status %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, maxBody)).Decode(into); err != nil {
		return fmt.Errorf("[mapillary] decode: %w", err)
	}
	return nil
}

// Score ranks a photo for a "see the street" strip: Mapillary's quality
// estimate, plus a bonus for an ordinary camera. Fisheye (dashcams, action
// cams) is wide and bent; spherical is a 360° panorama that reads badly as a
// thumbnail. Recency doesn't count: a sharp 2021 photo beats a blurry new one.
func Score(im Image) float64 {
	q := 0.4 // unknown quality: middling
	if im.QualityScore != nil {
		q = *im.QualityScore
	}
	switch {
	case im.IsPano || im.CameraType == "spherical":
		q -= 0.5
	case im.CameraType == "perspective":
		q += 0.3
	}
	return q
}

// Pick chooses the n best photos (by Score) for a strip, one per capture
// sequence (a sequence is one drive; its frames look alike) and at least
// minGapM apart, so the strip shows different spots.
func Pick(imgs []Image, n int, minGapM float64) []Image {
	sorted := append([]Image(nil), imgs...)
	sort.SliceStable(sorted, func(i, j int) bool { return Score(sorted[i]) > Score(sorted[j]) })

	var picked []Image
	seenSeq := map[string]bool{}
	for _, im := range sorted {
		if len(picked) >= n {
			break
		}
		if im.SequenceID != "" && seenSeq[im.SequenceID] {
			continue
		}
		tooClose := false
		for _, p := range picked {
			if distanceM(p.Lat, p.Lng, im.Lat, im.Lng) < minGapM {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}
		seenSeq[im.SequenceID] = true
		picked = append(picked, im)
	}
	return picked
}

// Download fetches an image's bytes (to host it ourselves).
func (c *Client) Download(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("[mapillary] build download: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[mapillary] download: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[mapillary] download: status %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, maxImage))
}

func distanceM(lat1, lng1, lat2, lng2 float64) float64 {
	const earthR = 6_371_000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthR * math.Asin(math.Sqrt(a))
}
