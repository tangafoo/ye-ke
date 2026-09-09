package news

import (
	"context"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
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

func (c *Client) Search(ctx context.Context, query string) ([]Article, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("hl", "ms-MY")
	params.Set("gl", "MY")
	params.Set("ceid", "MY:ms")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("[news] build request: %w", err)
	}
	req.Header.Set("User-Agent", browserUA)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[news] google news fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[news] google news returned: %d", resp.StatusCode)
	}

	var feed rssFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("[news] parse rss: %w", err)
	}

	articles := make([]Article, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		articles = append(articles, it.toArticle())
	}
	return articles, nil
}
