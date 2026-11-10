package main

import (
	"context"
	"errors"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"byakugan/internal/blob"
	"byakugan/internal/news"
	"byakugan/internal/store"
)

const (
	enrichBatch       = 40
	enrichConcurrency = 3
	enrichBudget      = 15 * time.Minute
	enrichPause       = 300 * time.Millisecond // per worker, between articles: be gentle with Google

	// news photos are stored as credited thumbnails, not the outlet's full image
	newsThumbWidth   = 640
	newsThumbQuality = 80
)

// hosts whose pages need a login to read, so there's no og:image to fetch
var noOGHosts = []string{"facebook.com", "fb.com", "instagram.com", "x.com", "twitter.com", "tiktok.com"}

// kickEnrich starts the background enricher unless it's already running.
//  1. resolve each news find's publisher URL (Google News links are redirects)
//     and note the article's photo (og:image);
//  2. copy that photo into R2 as a small JPEG thumbnail, credited to the
//     outlet, so cards never depend on someone else's server.
//
// Scans never wait on it: photos show up on the next look at that spot.
func (s *byakuganServer) kickEnrich() {
	if !s.enriching.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer s.enriching.Store(false)
		ctx, cancel := context.WithTimeout(context.Background(), enrichBudget)
		defer cancel()

		total, images := 0, 0
		for ctx.Err() == nil {
			pending, err := s.store.PendingEnrich(ctx, enrichBatch)
			if err != nil {
				log.Printf("%v", err)
				return
			}
			if len(pending) == 0 {
				break
			}
			images += s.enrichBatch(ctx, pending)
			total += len(pending)
		}
		if total > 0 {
			log.Printf("[enrich] %d articles enriched, %d with photos", total, images)
		}

		if s.blob == nil {
			return // photos stay unhosted (and unshown) until R2 is configured
		}
		hosted, tried := 0, 0
		for ctx.Err() == nil {
			pending, err := s.store.PendingHost(ctx, enrichBatch)
			if err != nil {
				log.Printf("%v", err)
				return
			}
			if len(pending) == 0 {
				break
			}
			hosted += s.hostBatch(ctx, pending)
			tried += len(pending)
		}
		if tried > 0 {
			log.Printf("[enrich] hosted %d of %d news photos", hosted, tried)
		}
	}()
}

func (s *byakuganServer) hostBatch(ctx context.Context, finds []store.Find) int {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		hosted int
		jobs   = make(chan store.Find)
	)
	for range enrichConcurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				u, err := s.hostNewsPhoto(ctx, f)
				if err != nil {
					log.Printf("[enrich] host photo %s: %v", f.Outlet, err)
				}
				if err := s.store.SaveHosted(ctx, f.ID, u); err != nil {
					log.Printf("%v", err)
				}
				if u != "" {
					mu.Lock()
					hosted++
					mu.Unlock()
				}
			}
		}()
	}
	for _, f := range finds {
		jobs <- f
	}
	close(jobs)
	wg.Wait()
	return hosted
}

// hostNewsPhoto stores a thumbnail of the article's photo as news/<find id>.jpg.
func (s *byakuganServer) hostNewsPhoto(ctx context.Context, f store.Find) (string, error) {
	src, err := s.news.FetchImage(ctx, f.ImageSourceURL)
	if err != nil {
		return "", err
	}
	thumb, err := blob.Thumbnail(src, newsThumbWidth, newsThumbQuality)
	if err != nil {
		return "", err
	}
	return s.blob.Put(ctx, "news/"+f.ID+".jpg", "image/jpeg", thumb)
}

func (s *byakuganServer) enrichBatch(ctx context.Context, finds []store.Find) int {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		images int
		jobs   = make(chan store.Find)
	)
	for range enrichConcurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				articleURL, imageURL := s.enrichOne(ctx, f)
				if err := s.store.SaveEnrichment(ctx, f.ID, articleURL, imageURL); err != nil {
					log.Printf("%v", err)
				}
				if imageURL != "" {
					mu.Lock()
					images++
					mu.Unlock()
				}
				time.Sleep(enrichPause)
			}
		}()
	}
	for _, f := range finds {
		jobs <- f
	}
	close(jobs)
	wg.Wait()
	return images
}

func (s *byakuganServer) enrichOne(ctx context.Context, f store.Find) (articleURL, imageURL string) {
	articleURL = f.URL
	resolved, err := s.news.ResolveURL(ctx, f.URL)
	switch {
	case err == nil:
		articleURL = resolved
	case errors.Is(err, news.ErrNotGoogleNews):
		// already a publisher link
	default:
		log.Printf("[enrich] resolve %s: %v", f.Outlet, err)
		return "", "" // keep the CBMi link in url; nothing to fetch a photo from
	}

	if u, err := url.Parse(articleURL); err == nil {
		for _, h := range noOGHosts {
			if strings.HasSuffix(u.Hostname(), h) {
				return articleURL, ""
			}
		}
	}

	img, err := s.news.OGImage(ctx, articleURL)
	if err != nil && !errors.Is(err, news.ErrNoImage) {
		log.Printf("[enrich] photo %s: %v", f.Outlet, err) // e.g. NSTP returns 403 to fetchers
	}
	return articleURL, img
}
