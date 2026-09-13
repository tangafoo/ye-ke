package news

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	searchURL = "https://news.google.com/rss/search"

	maxBodyBytes = 4 << 20

	maxAttempts = 3
	baseBackoff = 500 * time.Millisecond

	maxRetryWait = 20 * time.Second

	defaultUserAgent = "YeKe/0.1 (+https://github.com/tangafoo/ye-ke)"
)

type MjolnirSettings struct {
	BaseURL   string
	UserAgent string
	HTTP      *http.Client
	Backoff   time.Duration
}

func (m MjolnirSettings) withDefaults() MjolnirSettings {
	if m.BaseURL == "" {
		m.BaseURL = searchURL
	}
	if m.UserAgent == "" {
		m.UserAgent = defaultUserAgent
	}
	if m.HTTP == nil {
		m.HTTP = &http.Client{Timeout: 15 * time.Second}
	}
	if m.Backoff <= 0 {
		m.Backoff = baseBackoff
	}
	return m
}

type Client struct {
	http      *http.Client
	baseURL   string
	userAgent string
	backoff   time.Duration
}

func New(m MjolnirSettings) *Client {
	m = m.withDefaults()
	return &Client{
		http:      m.HTTP,
		baseURL:   m.BaseURL,
		userAgent: m.UserAgent,
		backoff:   m.Backoff,
	}
}

type Article struct {
	Title     string
	Link      string
	Outlet    string
	OutletURL string
	Published time.Time
	Snippet   string
}

type rssFeed struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string    `xml:"title"`
	Link    string    `xml:"link"`
	PubDate string    `xml:"pubDate"`
	Desc    string    `xml:"description"`
	Source  rssSource `xml:"source"`
}

func (it rssItem) toArticle() Article {
	outlet := strings.TrimSpace(it.Source.Name)
	title := cleanTitle(it.Title, outlet)

	return Article{
		Title:     title,
		Link:      strings.TrimSpace(it.Link),
		Outlet:    outlet,
		OutletURL: strings.TrimSpace(it.Source.URL),
		Published: parsePublished(it.PubDate),
		Snippet:   cleanSnippet(it.Desc, title, outlet),
	}
}

type rssSource struct {
	Name string `xml:",chardata"`
	URL  string `xml:"url,attr"`
}

var tagRE = regexp.MustCompile(`<[^>]*>`)

var titleSeps = []string{" - ", " | ", " \u2013 ", " \u2014 "}

var (
	ErrEmptyQuery  = errors.New("[news] empty query")
	ErrUnavailable = errors.New("[news] upstream unavailable")
	ErrTooLarge    = errors.New("[news] response too large")
)

func stripHTML(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tagRE.ReplaceAllString(s, " "))), " ")
}

func cleanTitle(title, outlet string) string {
	title = strings.TrimSpace(title)
	if outlet == "" {
		return title
	}

	for {
		trimmed := title
		for _, sep := range titleSeps {
			if cut, ok := strings.CutSuffix(trimmed, sep+outlet); ok {
				trimmed = strings.TrimSpace(cut)
				break
			}
		}
		if trimmed == title {
			return title
		}
		title = trimmed
	}
}

func cleanSnippet(desc, title, outlet string) string {
	snippet := stripHTML(desc)
	if snippet == "" {
		return ""
	}
	filler := cleanTitle(strings.TrimSuffix(snippet, outlet), outlet)
	if strings.EqualFold(filler, title) {
		return ""
	}
	return snippet
}

var pubDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC3339,
}

func isUTCName(name string) bool {
	switch name {
	case "UTC", "GMT", "UT", "Z", "":
		return true
	}
	return false
}

var zoneAliases = strings.NewReplacer(" MYT", " +0800")

func parsePublished(raw string) time.Time {
	raw = zoneAliases.Replace(strings.TrimSpace(raw))

	for _, layout := range pubDateLayouts {
		t, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}

		if name, offset := t.Zone(); offset == 0 && !isUTCName(name) {
			return time.Time{}
		}
		return t.UTC()
	}

	return time.Time{}
}

func (c *Client) attempt(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("[news] build request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml;q=0.9, */*;q=0.8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[news] google news fetch: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, &httpError{
			status:     resp.StatusCode,
			retryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("[news] read body: %w", err)
	}
	if len(body) > maxBodyBytes {
		return nil, ErrTooLarge
	}
	return body, nil
}

func (c *Client) fetch(ctx context.Context, endpoint string) ([]byte, error) {
	var lastErr error

	for attempt := range maxAttempts {
		if attempt > 0 {

			var retryAfter time.Duration
			if he, ok := errors.AsType[*httpError](lastErr); ok {
				retryAfter = he.retryAfter
			}

			wait := c.backoffDelay(attempt-1, retryAfter)
			if wait > maxRetryWait {
				return nil, lastErr
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		body, err := c.attempt(ctx, endpoint)
		if err == nil {
			return body, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !retryable(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("[news] gave up after %d attempts: %w", maxAttempts, lastErr)
}

func (c *Client) backoffDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	d := c.backoff << attempt
	if j := d / 2; j > 0 {
		d += rand.N(j)
	}
	return d
}

func parseRetryAfter(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(raw); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

func (c *Client) Search(ctx context.Context, query string) ([]Article, error) {
	if strings.TrimSpace(query) == "" {
		return nil, ErrEmptyQuery
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("hl", "ms-MY")
	params.Set("gl", "MY")
	params.Set("ceid", "MY:ms")

	body, err := c.fetch(ctx, c.baseURL+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("[news] parse rss: %w", err)
	}

	articles := make([]Article, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		articles = append(articles, it.toArticle())
	}
	return articles, nil
}

type httpError struct {
	status     int
	retryAfter time.Duration
}

func (e *httpError) Error() string {
	return fmt.Sprintf("[news] google news returned: %d", e.status)
}

func (e *httpError) Unwrap() error { return ErrUnavailable }
func (e *httpError) retryable() bool {
	return e.status == http.StatusTooManyRequests || e.status >= 500
}

func retryable(err error) bool {
	if errors.Is(err, ErrTooLarge) {
		return false
	}

	if he, ok := errors.AsType[*httpError](err); ok {
		return he.retryable()
	}
	return true
}
