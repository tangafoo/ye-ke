package news

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseBatchexecute(t *testing.T) {
	// trimmed real response from 2026-11-10
	body := ")]}'\n\n[[\"wrb.fr\",\"Fbv4je\",\"[\\\"garturlres\\\",\\\"https://www.kosmo.com.my/2026/11/01/417-pelanggan-pusat-hiburan-diserbu/\\\",1]\",null,null,null,\"generic\"],[\"di\",17],[\"af.httprm\",16,\"-7275710102215806246\",8]]"
	got, err := parseBatchexecute([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://www.kosmo.com.my/2026/11/01/417-pelanggan-pusat-hiburan-diserbu/"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := parseBatchexecute([]byte(")]}'\n\n[[\"di\",17]]")); err == nil {
		t.Fatal("want an error when no url is present")
	}
}

func TestResolveURLRejectsNonGoogle(t *testing.T) {
	c := New(MjolnirSettings{})
	if _, err := c.ResolveURL(context.Background(), "https://www.kosmo.com.my/x"); err != ErrNotGoogleNews {
		t.Fatalf("err = %v, want ErrNotGoogleNews", err)
	}
}

func TestOGImage(t *testing.T) {
	pages := map[string]string{
		"/kosmo":    `<head><meta property="og:image" content="https://www.kosmo.com.my/wp-content/uploads/2026/11/pusat-hiburan-diserbu.jpeg" /></head>`,
		"/relative": `<meta content="/img/a.jpg?x=1&amp;y=2" property="og:image">`,
		"/twitter":  `<meta name="twitter:image" content="https://cdn.example/t.jpg">`,
		"/logo":     `<meta property="og:image" content="https://site.example/static/logo-share.png">`,
		"/none":     `<title>no image</title>`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(pages[r.URL.Path]))
	}))
	defer srv.Close()

	c := New(MjolnirSettings{})
	cases := map[string]string{
		"/kosmo":    "https://www.kosmo.com.my/wp-content/uploads/2026/11/pusat-hiburan-diserbu.jpeg",
		"/relative": srv.URL + "/img/a.jpg?x=1&y=2",
		"/twitter":  "https://cdn.example/t.jpg",
		"/logo":     "",
		"/none":     "",
	}
	for path, want := range cases {
		got, err := c.OGImage(context.Background(), srv.URL+path)
		if want == "" {
			if err != ErrNoImage {
				t.Errorf("%s: got %q, %v; want ErrNoImage", path, got, err)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("%s: got %q, %v; want %q", path, got, err, want)
		}
	}
}
