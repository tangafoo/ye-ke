package news

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Google News RSS links are per-query redirects (news.google.com/rss/articles/
// CBMi…), and the token inside is signed, not decodable offline. The real
// outlet URL comes from Google's own web app: the article page carries a
// signature + timestamp, which its batchexecute endpoint (rpc "Fbv4je")
// exchanges for the publisher URL. Undocumented: if Google changes it, this
// fails loudly and finds simply keep their CBMi link.
const (
	gnArticleURL   = "https://news.google.com/rss/articles/"
	batchexecute   = "https://news.google.com/_/DotsSplashUi/data/batchexecute"
	browserUA      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0 Safari/537.36"
	maxPageBytes   = 3 << 20
	maxOutletBytes = 2 << 20
	maxImageBytes  = 12 << 20
)

var (
	ErrNotGoogleNews = errors.New("[news] not a Google News link")
	ErrNoImage       = errors.New("[news] no og:image")

	gnIDRE  = regexp.MustCompile(`/articles/([A-Za-z0-9_\-]+)`)
	gnSigRE = regexp.MustCompile(`data-n-a-sg="([^"]+)"`)
	gnTsRE  = regexp.MustCompile(`data-n-a-ts="([^"]+)"`)
)

// ResolveURL turns a Google News CBMi link into the publisher's URL.
func (c *Client) ResolveURL(ctx context.Context, link string) (string, error) {
	m := gnIDRE.FindStringSubmatch(link)
	if m == nil || !strings.Contains(link, "news.google.com") {
		return "", ErrNotGoogleNews
	}
	id := m[1]

	page, err := c.get(ctx, gnArticleURL+id, browserUA, maxPageBytes)
	if err != nil {
		return "", fmt.Errorf("[news] resolve page: %w", err)
	}
	sig, ts := gnSigRE.FindSubmatch(page), gnTsRE.FindSubmatch(page)
	if sig == nil || ts == nil {
		return "", errors.New("[news] resolve: no signature on article page")
	}

	inner := fmt.Sprintf(`["garturlreq",[["X","X",["X","X"],null,null,1,1,"US:en",null,1,null,null,null,null,null,0,1],"X","X",1,[1,1,1],1,1,null,0,0,null,0],"%s",%s,"%s"]`,
		id, ts[1], sig[1])
	freq, err := json.Marshal([][][]any{{{"Fbv4je", inner, nil, "generic"}}})
	if err != nil {
		return "", fmt.Errorf("[news] resolve marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, batchexecute,
		strings.NewReader("f.req="+url.QueryEscape(string(freq))))
	if err != nil {
		return "", fmt.Errorf("[news] resolve request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", browserUA)

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("[news] resolve: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("[news] resolve: status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, maxPageBytes))
	if err != nil {
		return "", fmt.Errorf("[news] resolve read: %w", err)
	}
	return parseBatchexecute(body)
}

// parseBatchexecute reads `)]}'\n\n[["wrb.fr","Fbv4je","[\"garturlres\",\"<url>\",1]",…]]`.
func parseBatchexecute(body []byte) (string, error) {
	s := string(body)
	if i := strings.Index(s, "\n\n"); i >= 0 {
		s = s[i+2:]
	}
	var outer [][]any
	if err := json.Unmarshal([]byte(s), &outer); err != nil {
		return "", fmt.Errorf("[news] resolve parse: %w", err)
	}
	for _, row := range outer {
		if len(row) < 3 || row[0] != "wrb.fr" {
			continue
		}
		payload, ok := row[2].(string)
		if !ok {
			continue
		}
		var res []any
		if err := json.Unmarshal([]byte(payload), &res); err != nil || len(res) < 2 {
			continue
		}
		if u, ok := res[1].(string); ok && strings.HasPrefix(u, "http") {
			return u, nil
		}
	}
	return "", errors.New("[news] resolve: no url in response")
}

var (
	metaTagRE   = regexp.MustCompile(`(?is)<meta\s[^>]*>`)
	metaKeyRE   = regexp.MustCompile(`(?i)(?:property|name)\s*=\s*["'](og:image(?::secure_url)?|og:image:url|twitter:image(?::src)?)["']`)
	metaValueRE = regexp.MustCompile(`(?i)content\s*=\s*["']([^"']+)["']`)
)

// OGImage fetches an article page and returns its share image (og:image, then
// twitter:image), resolved against the page URL. Site-wide default images
// (logos) are rejected: a card is better with no photo than the outlet's logo.
func (c *Client) OGImage(ctx context.Context, pageURL string) (string, error) {
	page, err := c.get(ctx, pageURL, c.userAgent, maxOutletBytes)
	if err != nil {
		return "", err
	}
	base, _ := url.Parse(pageURL)

	for _, tag := range metaTagRE.FindAll(page, -1) {
		if !metaKeyRE.Match(tag) {
			continue
		}
		v := metaValueRE.FindSubmatch(tag)
		if v == nil {
			continue
		}
		img := strings.TrimSpace(htmlUnescape(string(v[1])))
		if img == "" || looksLikeLogo(img) {
			continue
		}
		if ref, err := url.Parse(img); err == nil && base != nil {
			img = base.ResolveReference(ref).String()
		}
		return img, nil
	}
	return "", ErrNoImage
}

func looksLikeLogo(img string) bool {
	l := strings.ToLower(img)
	// site-wide share images, not the article's photo ("st-bg-1200x630.jpg" is The Straits Times' banner)
	for _, w := range []string{"logo", "default", "placeholder", "favicon", "share-image", "og-default", "st-bg-"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

func htmlUnescape(s string) string {
	return strings.NewReplacer("&amp;", "&", "&#038;", "&", "&#38;", "&", "&quot;", `"`, "&#039;", "'").Replace(s)
}

// get fetches a page with the given User-Agent, capped at limit bytes.
func (c *Client) get(ctx context.Context, u, ua string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("[news] build request: %w", err)
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[news] get %s: %w", u, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("[news] get %s: status %d", u, res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, limit))
}

// FetchImage downloads an image (the og:image of an article) for us to make
// our own thumbnail of it.
func (c *Client) FetchImage(ctx context.Context, u string) ([]byte, error) {
	return c.get(ctx, u, c.userAgent, maxImageBytes)
}
