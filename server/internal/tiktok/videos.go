package tiktok

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// seemuapps' actor takes place NAMES (many per run), resolves each to TikTok
// places, and returns the videos tagged there, every row carrying the place's
// coordinates. One run per scan instead of one run per keyword.
const placeVideosURL = "https://api.apify.com/v2/acts/seemuapps~tiktok-location-scraper/run-sync-get-dataset-items"

type Video struct {
	Type         string  `json:"type"` // "video" or "place"
	ID           string  `json:"videoId"`
	URL          string  `json:"videoUrl"`
	Caption      string  `json:"caption"`
	CreatedAt    int64   `json:"createTimestamp"` // unix seconds
	Author       string  `json:"authorUsername"`
	Views        int64   `json:"views"`
	PlaceID      string  `json:"placeId"`
	PlaceName    string  `json:"placeName"`
	PlaceAddress string  `json:"placeAddress"`
	PlaceLat     float64 `json:"placeLatitude"`
	PlaceLng     float64 `json:"placeLongitude"`
}

type placeVideosInput struct {
	Queries           []string `json:"queries"`
	PlacesPerQuery    int      `json:"placesPerQuery"`
	MaxVideosPerPlace int      `json:"maxVideosPerPlace"`
}

// VideosAtPlaces resolves each query to up to placesPerQuery TikTok places and
// returns up to maxVideosPerPlace videos tagged at each.
func (c *Client) VideosAtPlaces(ctx context.Context, queries []string, placesPerQuery, maxVideosPerPlace int) ([]Video, error) {
	in := placeVideosInput{
		Queries:           queries,
		PlacesPerQuery:    placesPerQuery,
		MaxVideosPerPlace: maxVideosPerPlace,
	}

	body, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("[tiktok] marshal input: %w", err)
	}

	// the actor runs synchronously; Apify caps sync runs at 300s
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, placeVideosURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("[tiktok] build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[tiktok] place videos: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("[tiktok] place videos: status %d", res.StatusCode)
	}

	var rows []Video
	if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
		return nil, fmt.Errorf("[tiktok] decode videos: %w", err)
	}

	videos := rows[:0]
	for _, v := range rows {
		if v.Type == "video" {
			videos = append(videos, v)
		}
	}
	return videos, nil
}
