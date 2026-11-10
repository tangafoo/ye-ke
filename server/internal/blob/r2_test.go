package blob

import (
	"bytes"
	"context"
	"image"
	pngenc "image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOffWithoutConfig(t *testing.T) {
	if New(Config{AccountID: "a", Bucket: "b"}) != nil {
		t.Fatal("incomplete config should leave R2 off (nil)")
	}
}

func TestPut(t *testing.T) {
	var gotPath, gotType, gotCache string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotType, gotCache = r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("Cache-Control")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := newWithEndpoint(Config{AccessKeyID: "k", SecretAccessKey: "s", Bucket: "yeke", PublicBaseURL: "https://img.example/"}, srv.URL)
	u, err := s.Put(context.Background(), "street/mapillary/1_256.jpg", "image/jpeg", []byte("jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if u != "https://img.example/street/mapillary/1_256.jpg" {
		t.Errorf("url = %q", u)
	}
	if gotPath != "/yeke/street/mapillary/1_256.jpg" || gotType != "image/jpeg" || string(gotBody) != "jpg" {
		t.Errorf("request path=%q type=%q body=%q", gotPath, gotType, gotBody)
	}
	if gotCache == "" {
		t.Error("want a long Cache-Control on immutable objects")
	}
}

func TestThumbnail(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1200, 800))
	var png bytes.Buffer
	if err := pngenc.Encode(&png, src); err != nil {
		t.Fatal(err)
	}
	out, err := Thumbnail(png.Bytes(), 640, 80)
	if err != nil {
		t.Fatal(err)
	}
	got, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if format != "jpeg" || got.Width != 640 || got.Height != 426 {
		t.Fatalf("thumbnail = %s %dx%d, want jpeg 640x426", format, got.Width, got.Height)
	}

	small, _ := Thumbnail(png.Bytes(), 2000, 80) // never upscaled
	if c, _, _ := image.DecodeConfig(bytes.NewReader(small)); c.Width != 1200 {
		t.Fatalf("width = %d, want 1200 (no upscaling)", c.Width)
	}
	if _, err := Thumbnail([]byte("not an image"), 640, 80); err == nil {
		t.Fatal("want an error for garbage input")
	}
}
