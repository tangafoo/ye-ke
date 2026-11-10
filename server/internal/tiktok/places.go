package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const placeSearchURL = "https://api.apify.com/v2/acts/romy~tiktok-place-location-search-scraper/run-sync-get-dataset-items"

type Client struct {
	token string
	http  *http.Client
}

type Place struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Lat      float64 `json:"latitude"`
	Lng      float64 `json:"longitude"`
	Category string  `json:"display_category"`
}

func New(token string) *Client {
	return &Client{token: token, http: &http.Client{}}
}

type proxyConfig struct {
	UseApifyProxy bool     `json:"useApifyProxy"`
	Groups        []string `json:"apifyProxyGroups"`
	Country       string   `json:"apifyProxyCountry"`
}

type placeSearchInput struct {
	Keyword   string      `json:"keyword"`
	Region    string      `json:"region"`
	Latitude  string      `json:"latitude"` // the actor's schema wants strings
	Longitude string      `json:"longitude"`
	Radius    int         `json:"radius"`
	MaxPages  int         `json:"maxPages"`
	Proxy     proxyConfig `json:"proxyConfiguration"`
}

// SearchPlaces finds TikTok places matching keyword within radiusM of the point.
// region is an ISO country code (MY, DE, AU, FR). TikTok ranks place results by
// the caller's IP, so the residential proxy exits in that same country.
func (c *Client) SearchPlaces(ctx context.Context, keyword, region string, lat, lng float64, radiusM int) ([]Place, error) {
	in := placeSearchInput{
		Keyword:   keyword,
		Region:    region,
		Latitude:  strconv.FormatFloat(lat, 'f', 6, 64),
		Longitude: strconv.FormatFloat(lng, 'f', 6, 64),
		Radius:    radiusM,
		MaxPages:  1, // 20 places per page; keep it cheap while testing
		Proxy: proxyConfig{
			UseApifyProxy: true,
			Groups:        []string{"RESIDENTIAL"},
			Country:       region,
		},
	}

	body, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("[tiktok] marshal input: %w", err)
	}

	// the actor runs synchronously; Apify caps sync runs at 300s
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, placeSearchURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("[tiktok] build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[tiktok] place search: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("[tiktok] place search: status %d", res.StatusCode)
	}

	var places []Place
	if err := json.NewDecoder(res.Body).Decode(&places); err != nil {
		return nil, fmt.Errorf("[tiktok] decode places: %w", err)
	}
	return places, nil
}
