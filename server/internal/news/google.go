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
	browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
)

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: 15 * time.Second}}
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

var (
	ErrEmptyQuery  = errors.New("[news] empty query")
	ErrUnavailable = errors.New("[news] upstream unavailable")
	ErrTooLarge    = errors.New("[news] response too large")
)

const (
	maxBodyBytes = 4 << 20

	maxAttempts = 3
	baseBackoff = 500 * time.Millisecond
)

func stripHTML(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tagRE.ReplaceAllString(s, " "))), " ")
}

func cleanTitle(title, outlet string) string {
	title = strings.TrimSpace(title)
	if outlet == "" {
		return title
	}
	if trimmed, ok := strings.CutSuffix(title, " - "+outlet); ok {
		return strings.TrimSpace(trimmed)
	}
	return title
}

func cleanSnippet(desc, rawTitle, outlet string) string {
	snippet := stripHTML(desc)
	if snippet == "" {
		return ""
	}
	filler := strings.TrimSpace(strings.TrimSuffix(snippet, outlet))
	if strings.EqualFold(filler, strings.TrimSpace(rawTitle)) {
		return ""
	}
	return snippet
}

var pubDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
}

func parsePublished(raw string) time.Time {
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, strings.TrimSpace(raw)); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func (c *Client) attempt(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("[news] build request: %w", err)
	}
	req.Header.Set("User-Agent", browserUA)
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
			var he *httpError
			var retryAfter time.Duration
			if errors.As(lastErr, &he) {
				retryAfter = he.retryAfter
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffDelay(attempt-1, retryAfter)):
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

func backoffDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	d := baseBackoff << attempt
	return d + rand.N(d/2)
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

	body, err := c.fetch(ctx, searchURL+"?"+params.Encode())
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
	var he *httpError
	if errors.As(err, &he) {
		return he.retryable()
	}
	return true
}
