package news

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// countingServer wraps an httptest server with a request counter, so a test can
// assert not just the outcome but how many round trips it took to get there.
// "Did it retry?" and "did it retry too much?" are both failures we care about.
type countingServer struct {
	*httptest.Server
	hits atomic.Int32
}

// newCountingServer serves h, counting every request. h receives the 0-indexed
// attempt number so it can vary its response per attempt.
func newCountingServer(t *testing.T, h func(attempt int, w http.ResponseWriter, r *http.Request)) *countingServer {
	t.Helper()

	cs := &countingServer{}
	cs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := int(cs.hits.Add(1)) - 1
		h(attempt, w, r)
	}))
	t.Cleanup(cs.Close)
	return cs
}

// testClient points a Client at srv with a backoff small enough that the retry
// loop costs microseconds, not the ~2.25s a production backoff would.
func testClient(srv *countingServer) *Client {
	return New(MjolnirSettings{
		BaseURL: srv.URL,
		Backoff: time.Millisecond,
	})
}

// feedXML renders a minimal but realistically-shaped Google News RSS document.
func feedXML(items ...string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Google News</title>` +
		strings.Join(items, "") +
		`</channel></rss>`
}

// ---------------------------------------------------------------------------
// transport: the three behaviours that were unreachable before BaseURL existed
// ---------------------------------------------------------------------------

func TestFetchRetriesThenSucceeds(t *testing.T) {
	tests := []struct {
		name       string
		failStatus int
	}{
		{"retries after 429", http.StatusTooManyRequests},
		{"retries after 500", http.StatusInternalServerError},
		{"retries after 503", http.StatusServiceUnavailable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
				if attempt == 0 {
					w.WriteHeader(tc.failStatus)
					return
				}
				fmt.Fprint(w, feedXML())
			})

			if _, err := testClient(srv).Search(context.Background(), "roadblock"); err != nil {
				t.Fatalf("Search() error = %v, want nil (should recover on retry)", err)
			}
			if got := srv.hits.Load(); got != 2 {
				t.Errorf("requests = %d, want 2 (one failure + one success)", got)
			}
		})
	}
}

func TestFetchGivesUpAfterMaxAttempts(t *testing.T) {
	srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := testClient(srv).Search(context.Background(), "roadblock")
	if err == nil {
		t.Fatal("Search() error = nil, want an error after exhausting retries")
	}
	// The give-up wrap must preserve the sentinel so callers can classify a
	// degraded source without string-matching.
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("errors.Is(err, ErrUnavailable) = false, want true; err = %v", err)
	}
	if got := srv.hits.Load(); got != maxAttempts {
		t.Errorf("requests = %d, want %d", got, maxAttempts)
	}
}

func TestFetchDoesNotRetryTerminalStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"404 is terminal", http.StatusNotFound},
		{"403 is terminal", http.StatusForbidden},
		{"400 is terminal", http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
			})

			_, err := testClient(srv).Search(context.Background(), "roadblock")
			if err == nil {
				t.Fatalf("Search() error = nil, want an error for status %d", tc.status)
			}
			// A 4xx will not fix itself. Burning retries on it wastes the
			// scan's latency budget for nothing.
			if got := srv.hits.Load(); got != 1 {
				t.Errorf("requests = %d, want 1 (4xx must not be retried)", got)
			}
		})
	}
}

func TestFetchBailsOnExcessiveRetryAfter(t *testing.T) {
	srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		// An hour. Honouring this would hang a user-facing scan.
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	start := time.Now()
	_, err := testClient(srv).Search(context.Background(), "roadblock")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Search() error = nil, want an error")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("errors.Is(err, ErrUnavailable) = false, want true; err = %v", err)
	}
	// One request: the bail happens before the second attempt sleeps.
	if got := srv.hits.Load(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
	// The real assertion — it returned instead of sleeping for an hour.
	if elapsed > 5*time.Second {
		t.Errorf("elapsed = %v, want a near-immediate bail (maxRetryWait = %v)", elapsed, maxRetryWait)
	}
}

func TestFetchHonoursShortRetryAfter(t *testing.T) {
	srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		if attempt == 0 {
			// Under maxRetryWait, so this one should be obeyed, not bailed on.
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, feedXML())
	})

	start := time.Now()
	if _, err := testClient(srv).Search(context.Background(), "roadblock"); err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}

	if got := srv.hits.Load(); got != 2 {
		t.Errorf("requests = %d, want 2", got)
	}
	// Retry-After beats our own backoff, so this must take ~1s despite the
	// 1ms test backoff. If it returns instantly, the header was ignored.
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond {
		t.Errorf("elapsed = %v, want >= ~1s (Retry-After must override backoff)", elapsed)
	}
}

func TestFetchRejectsOversizeBodyWithoutRetrying(t *testing.T) {
	srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, maxBodyBytes+1024))
	})

	_, err := testClient(srv).Search(context.Background(), "roadblock")
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Search() error = %v, want ErrTooLarge", err)
	}
	// The regression guard: a 4MB body that gets retried downloads 12MB.
	if got := srv.hits.Load(); got != 1 {
		t.Errorf("requests = %d, want 1 (oversize body must not be retried)", got)
	}
}

func TestFetchStopsOnCancelledContext(t *testing.T) {
	srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := testClient(srv).Search(ctx, "roadblock")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Search() error = %v, want context.Canceled", err)
	}
}

// ---------------------------------------------------------------------------
// request shape
// ---------------------------------------------------------------------------

func TestSearchSendsExpectedRequest(t *testing.T) {
	var got *http.Request
	srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		fmt.Fprint(w, feedXML())
	})

	if _, err := testClient(srv).Search(context.Background(), "sekatan jalan"); err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	q := got.URL.Query()
	for key, want := range map[string]string{
		"q":    "sekatan jalan",
		"hl":   "ms-MY",
		"gl":   "MY",
		"ceid": "MY:ms",
	} {
		if q.Get(key) != want {
			t.Errorf("query param %q = %q, want %q", key, q.Get(key), want)
		}
	}
	if ua := got.Header.Get("User-Agent"); ua != browserUA {
		t.Errorf("User-Agent = %q, want %q", ua, browserUA)
	}
}

func TestSearchRejectsEmptyQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"empty", ""},
		{"spaces only", "   "},
		{"tab and newline", "\t\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, feedXML())
			})

			_, err := testClient(srv).Search(context.Background(), tc.query)
			if !errors.Is(err, ErrEmptyQuery) {
				t.Fatalf("Search() error = %v, want ErrEmptyQuery", err)
			}
			// Validation happens before any network call.
			if got := srv.hits.Load(); got != 0 {
				t.Errorf("requests = %d, want 0", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// settings
// ---------------------------------------------------------------------------

func TestSettingsWithDefaults(t *testing.T) {
	t.Run("zero value is production-shaped", func(t *testing.T) {
		got := MjolnirSettings{}.withDefaults()

		if got.BaseURL != searchURL {
			t.Errorf("BaseURL = %q, want %q", got.BaseURL, searchURL)
		}
		if got.HTTP == nil {
			t.Error("HTTP = nil, want a client")
		}
		if got.Backoff != baseBackoff {
			t.Errorf("Backoff = %v, want %v", got.Backoff, baseBackoff)
		}
	})

	t.Run("caller values survive", func(t *testing.T) {
		custom := &http.Client{Timeout: time.Second}
		got := MjolnirSettings{
			BaseURL: "https://example.test/rss",
			HTTP:    custom,
			Backoff: 42 * time.Millisecond,
		}.withDefaults()

		if got.BaseURL != "https://example.test/rss" {
			t.Errorf("BaseURL = %q, want it preserved", got.BaseURL)
		}
		if got.HTTP != custom {
			t.Error("HTTP was replaced, want the injected client")
		}
		if got.Backoff != 42*time.Millisecond {
			t.Errorf("Backoff = %v, want it preserved", got.Backoff)
		}
	})

	t.Run("does not mutate the caller", func(t *testing.T) {
		// The value receiver is the guarantee here: a shared settings struct
		// must be safe to hand to New more than once.
		original := MjolnirSettings{}
		_ = original.withDefaults()

		if original.BaseURL != "" || original.HTTP != nil || original.Backoff != 0 {
			t.Errorf("withDefaults mutated the receiver: %+v", original)
		}
	})
}

func TestBackoffDelay(t *testing.T) {
	c := New(MjolnirSettings{Backoff: 100 * time.Millisecond})

	t.Run("grows exponentially within the jitter band", func(t *testing.T) {
		// attempt n => [base<<n, base<<n * 1.5)
		for attempt, want := range map[int]struct{ min, max time.Duration }{
			0: {100 * time.Millisecond, 150 * time.Millisecond},
			1: {200 * time.Millisecond, 300 * time.Millisecond},
			2: {400 * time.Millisecond, 600 * time.Millisecond},
		} {
			for range 50 { // jitter is random; sample it
				got := c.backoffDelay(attempt, 0)
				if got < want.min || got >= want.max {
					t.Fatalf("backoffDelay(%d, 0) = %v, want in [%v, %v)", attempt, got, want.min, want.max)
				}
			}
		}
	})

	t.Run("retryAfter wins", func(t *testing.T) {
		if got := c.backoffDelay(0, 7*time.Second); got != 7*time.Second {
			t.Errorf("backoffDelay(0, 7s) = %v, want 7s", got)
		}
	})

	t.Run("tiny backoff does not panic", func(t *testing.T) {
		// d/2 == 0 would panic rand.N. The jitter guard must skip instead.
		tiny := New(MjolnirSettings{Backoff: 1})
		if got := tiny.backoffDelay(0, 0); got != 1 {
			t.Errorf("backoffDelay(0, 0) = %v, want 1ns", got)
		}
	})
}

// ---------------------------------------------------------------------------
// parsing: pure functions, no server needed
// ---------------------------------------------------------------------------

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		outlet string
		want   string
	}{
		{
			name:   "strips the outlet suffix",
			title:  "Polis tahan lima lelaki di Kampung Baru - Kosmo Digital",
			outlet: "Kosmo Digital",
			want:   "Polis tahan lima lelaki di Kampung Baru",
		},
		{
			name:   "keeps a dash that is not the outlet",
			title:  "Ops Selamat - 12 ditahan",
			outlet: "Berita Harian",
			want:   "Ops Selamat - 12 ditahan",
		},
		{
			name:   "keeps a dash when the outlet is unknown",
			title:  "Ops Selamat - 12 ditahan",
			outlet: "",
			want:   "Ops Selamat - 12 ditahan",
		},
		{
			name:   "only cuts a real suffix, not a mid-string match",
			title:  "Kosmo Digital - laporan penuh - Kosmo Digital",
			outlet: "Kosmo Digital",
			want:   "Kosmo Digital - laporan penuh",
		},
		{
			// Verified live: 16/107 items arrive with the outlet twice —
			// once from the publisher's own <title> ("| Sinar Harian") and
			// once appended by Google (" - Sinar Harian").
			name:   "strips a doubled outlet suffix with a pipe",
			title:  "Polis cari pemandu uji kelajuan di Putrajaya | Sinar Harian - Sinar Harian",
			outlet: "Sinar Harian",
			want:   "Polis cari pemandu uji kelajuan di Putrajaya",
		},
		{
			name:   "strips a doubled outlet suffix with an en dash",
			title:  "Polis rakam keterangan enam pendaki – Sinar Harian - Sinar Harian",
			outlet: "Sinar Harian",
			want:   "Polis rakam keterangan enam pendaki",
		},
		{
			name:   "leaves a pipe that is not followed by the outlet",
			title:  "Ops Selamat | 12 ditahan - Berita Harian",
			outlet: "Berita Harian",
			want:   "Ops Selamat | 12 ditahan",
		},
		{
			name:   "title that is only the outlet is left alone",
			title:  "Sinar Harian",
			outlet: "Sinar Harian",
			want:   "Sinar Harian",
		},
		{
			name:   "trims surrounding whitespace",
			title:  "  Roadblock di NKVE - FMT  ",
			outlet: "FMT",
			want:   "Roadblock di NKVE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanTitle(tc.title, tc.outlet); got != tc.want {
				t.Errorf("cleanTitle() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCleanSnippet(t *testing.T) {
	tests := []struct {
		name   string
		desc   string
		title  string
		outlet string
		want   string
	}{
		{
			// Google's search feed has no real summary: the description is
			// always an anchor holding the *cleaned* title (verified against
			// 107 live items — the anchor never carries the " - Outlet"
			// suffix that <title> does), then the outlet again. Rendering it
			// would print the headline twice on a card.
			name:   "drops Google's stock filler description",
			desc:   `<a href="https://news.google.com/rss/articles/CBMiK">Polis tahan lima lelaki</a>&nbsp;&nbsp;<font color="#6f6f6f">Kosmo Digital</font>`,
			title:  "Polis tahan lima lelaki",
			outlet: "Kosmo Digital",
			want:   "",
		},
		{
			// cleanTitle and cleanSnippet are coupled: the filler check
			// compares the description against the *cleaned* title, so the
			// description has to lose the publisher's own "| Outlet" too or
			// the comparison misses. Regression guard for exactly that.
			name:   "drops filler carrying the publisher's own suffix",
			desc:   `<a href="#">Polis cari pemandu uji kelajuan di Putrajaya | Sinar Harian</a>&nbsp;&nbsp;<font color="#6f6f6f">Sinar Harian</font>`,
			title:  "Polis cari pemandu uji kelajuan di Putrajaya",
			outlet: "Sinar Harian",
			want:   "",
		},
		{
			name:   "case differences still count as filler",
			desc:   `<a href="#">POLIS TAHAN LIMA LELAKI</a>&nbsp;&nbsp;<font color="#6f6f6f">Kosmo Digital</font>`,
			title:  "Polis tahan lima lelaki",
			outlet: "Kosmo Digital",
			want:   "",
		},
		{
			name:   "keeps a genuine snippet",
			desc:   `<p>Sekatan jalan raya didirikan di sepanjang Lebuhraya NKVE sejak pagi tadi.</p>`,
			title:  "Roadblock di NKVE",
			outlet: "FMT",
			want:   "Sekatan jalan raya didirikan di sepanjang Lebuhraya NKVE sejak pagi tadi.",
		},
		{
			name:   "empty description stays empty",
			desc:   "",
			title:  "Roadblock di NKVE",
			outlet: "FMT",
			want:   "",
		},
		{
			name:   "tags-only description collapses to empty",
			desc:   `<p></p><br/>`,
			title:  "Roadblock di NKVE",
			outlet: "FMT",
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanSnippet(tc.desc, tc.title, tc.outlet); got != tc.want {
				t.Errorf("cleanSnippet() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text passes through", "sekatan jalan", "sekatan jalan"},
		{"removes tags", "<p>polis <b>tahan</b> lima</p>", "polis tahan lima"},
		{
			// HTML nested inside XML arrives double-escaped: &amp;nbsp;
			// becomes &nbsp; on XML decode, then U+00A0 here. Fields() must
			// collapse it, or the snippet keeps invisible padding.
			name: "collapses non-breaking spaces",
			in:   "Polis tahan lima&nbsp;&nbsp;Kosmo Digital",
			want: "Polis tahan lima Kosmo Digital",
		},
		{"collapses runs of whitespace", "polis   \n\t tahan", "polis tahan"},
		{"unescapes entities", "&lt;bukan tag&gt; &amp; lagi", "<bukan tag> & lagi"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripHTML(tc.in); got != tc.want {
				t.Errorf("stripHTML() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParsePublished(t *testing.T) {
	want := time.Date(2026, 9, 8, 4, 1, 5, 0, time.UTC)

	tests := []struct {
		name string
		raw  string
		want time.Time
	}{
		{
			name: "RFC1123Z numeric offset",
			raw:  "Tue, 08 Sep 2026 12:01:05 +0800",
			want: want,
		},
		{
			name: "RFC1123 GMT, as Google sends it",
			raw:  "Tue, 08 Sep 2026 04:01:05 GMT",
			want: want,
		},
		{
			name: "RFC3339, as Atom feeds send it",
			raw:  "2026-09-08T12:01:05+08:00",
			want: want,
		},
		{
			// time.Parse does not know MYT and would silently fabricate a
			// zero offset — publishing a timestamp eight hours wrong. The
			// alias table rewrites it before parsing.
			name: "MYT alias resolves to +0800",
			raw:  "Tue, 08 Sep 2026 12:01:05 MYT",
			want: want,
		},
		{
			name: "MYT alias survives surrounding whitespace",
			raw:  "  Tue, 08 Sep 2026 12:01:05 MYT\n",
			want: want,
		},
		{
			// Fail closed: an unknown zone yields "unknown", never a
			// plausible-looking wrong time.
			name: "unknown zone abbreviation is rejected",
			raw:  "Tue, 08 Sep 2026 12:01:05 XYZ",
			want: time.Time{},
		},
		{"empty input", "", time.Time{}},
		{"garbage input", "not a date", time.Time{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parsePublished(tc.raw)
			if !got.Equal(tc.want) {
				t.Errorf("parsePublished(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{"delay-seconds", "120", 2 * time.Minute},
		{"delay-seconds with whitespace", "  30  ", 30 * time.Second},
		{"empty header", "", 0},
		{"zero is not a delay", "0", 0},
		{"negative is ignored", "-5", 0},
		{"unparseable", "soon", 0},
		{"HTTP-date in the past", "Mon, 01 Jan 2001 00:00:00 GMT", 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseRetryAfter(tc.raw); got != tc.want {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}

	t.Run("HTTP-date in the future", func(t *testing.T) {
		raw := time.Now().Add(90 * time.Second).UTC().Format(http.TimeFormat)
		got := parseRetryAfter(raw)
		// Second-granularity format plus clock movement: assert a band.
		if got < 80*time.Second || got > 90*time.Second {
			t.Errorf("parseRetryAfter(%q) = %v, want ~90s", raw, got)
		}
	})
}

func TestRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"429 is retryable", &httpError{status: http.StatusTooManyRequests}, true},
		{"500 is retryable", &httpError{status: http.StatusInternalServerError}, true},
		{"503 is retryable", &httpError{status: http.StatusServiceUnavailable}, true},
		{"404 is terminal", &httpError{status: http.StatusNotFound}, false},
		{"403 is terminal", &httpError{status: http.StatusForbidden}, false},
		{"oversize body is terminal", ErrTooLarge, false},
		{"wrapped oversize body is terminal", fmt.Errorf("read: %w", ErrTooLarge), false},
		{"wrapped httpError is unwrapped", fmt.Errorf("attempt: %w", &httpError{status: 500}), true},
		{"unknown errors are retried", errors.New("connection reset"), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryable(tc.err); got != tc.want {
				t.Errorf("retryable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestHTTPErrorUnwrapsToUnavailable(t *testing.T) {
	// Callers classify a degraded source with errors.Is; if the sentinel is
	// lost, a partial sweep becomes indistinguishable from a bug.
	err := error(&httpError{status: http.StatusTooManyRequests})
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("errors.Is(httpError, ErrUnavailable) = false, want true")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("Error() = %q, want it to name the status", err.Error())
	}
}

// ---------------------------------------------------------------------------
// end to end: a real-shaped feed through Search
// ---------------------------------------------------------------------------

func TestSearchParsesFeed(t *testing.T) {
	srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprint(w, feedXML(
			`<item>`+
				`<title>Polis tahan lima lelaki di Kampung Baru - Kosmo Digital</title>`+
				`<link>https://news.google.com/rss/articles/CBMiK1234</link>`+
				`<pubDate>Tue, 08 Sep 2026 04:01:05 GMT</pubDate>`+
				`<description>&lt;a href="https://news.google.com/rss/articles/CBMiK1234"&gt;Polis tahan lima lelaki di Kampung Baru&lt;/a&gt;&amp;nbsp;&amp;nbsp;&lt;font color="#6f6f6f"&gt;Kosmo Digital&lt;/font&gt;</description>`+
				`<source url="https://www.kosmo.com.my">Kosmo Digital</source>`+
				`</item>`,
			`<item>`+
				`<title>Sekatan jalan raya di NKVE - Free Malaysia Today</title>`+
				`<link>https://news.google.com/rss/articles/CBMiK5678</link>`+
				`<pubDate>Tue, 08 Sep 2026 12:01:05 +0800</pubDate>`+
				`<description>&lt;p&gt;Sekatan didirikan sejak pagi tadi.&lt;/p&gt;</description>`+
				`<source url="https://www.freemalaysiatoday.com">Free Malaysia Today</source>`+
				`</item>`,
		))
	})

	got, err := testClient(srv).Search(context.Background(), "polis")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(articles) = %d, want 2", len(got))
	}

	want := []Article{
		{
			Title:     "Polis tahan lima lelaki di Kampung Baru",
			Link:      "https://news.google.com/rss/articles/CBMiK1234",
			Outlet:    "Kosmo Digital",
			OutletURL: "https://www.kosmo.com.my",
			Published: time.Date(2026, 9, 8, 4, 1, 5, 0, time.UTC),
			Snippet:   "", // stock filler, correctly discarded
		},
		{
			Title:     "Sekatan jalan raya di NKVE",
			Link:      "https://news.google.com/rss/articles/CBMiK5678",
			Outlet:    "Free Malaysia Today",
			OutletURL: "https://www.freemalaysiatoday.com",
			Published: time.Date(2026, 9, 8, 4, 1, 5, 0, time.UTC),
			Snippet:   "Sekatan didirikan sejak pagi tadi.",
		},
	}

	for i, w := range want {
		if got[i] != w {
			t.Errorf("articles[%d]:\n got %+v\nwant %+v", i, got[i], w)
		}
	}
}

func TestSearchOnEmptyAndMalformedFeeds(t *testing.T) {
	t.Run("empty channel yields no articles, not an error", func(t *testing.T) {
		srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, feedXML())
		})

		got, err := testClient(srv).Search(context.Background(), "polis")
		if err != nil {
			t.Fatalf("Search() error = %v, want nil", err)
		}
		// "Swept, nothing here" is a real state and must be distinguishable
		// from a failure — the map renders them differently.
		if len(got) != 0 {
			t.Errorf("len(articles) = %d, want 0", len(got))
		}
	})

	t.Run("malformed xml is an error", func(t *testing.T) {
		srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, `<rss><channel><item><title>unclosed`)
		})

		if _, err := testClient(srv).Search(context.Background(), "polis"); err == nil {
			t.Fatal("Search() error = nil, want a parse error")
		}
	})

	t.Run("item missing every optional field", func(t *testing.T) {
		srv := newCountingServer(t, func(attempt int, w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, feedXML(`<item><title>Bare</title></item>`))
		})

		got, err := testClient(srv).Search(context.Background(), "polis")
		if err != nil {
			t.Fatalf("Search() error = %v, want nil", err)
		}
		if len(got) != 1 {
			t.Fatalf("len(articles) = %d, want 1", len(got))
		}
		if got[0].Title != "Bare" {
			t.Errorf("Title = %q, want %q", got[0].Title, "Bare")
		}
		if !got[0].Published.IsZero() {
			t.Errorf("Published = %v, want zero", got[0].Published)
		}
	})
}
