package mapillary

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewWithoutToken(t *testing.T) {
	if New("  ") != nil {
		t.Fatal("New with no token should return nil (Mapillary off)")
	}
	var c *Client
	if _, err := c.Near(context.Background(), 3.14, 101.69, 200, 10); err != ErrNoToken {
		t.Fatalf("err = %v, want ErrNoToken", err)
	}
}

func TestNear(t *testing.T) {
	var gotAuth, gotBBox string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path == "/images" {
			gotBBox = r.URL.Query().Get("bbox")
			w.Write([]byte(`{"data":[{"id":"111","thumb_256_url":"https://t/111","thumb_1024_url":"https://T/111",
				"captured_at":1700000000000,"compass_angle":90.5,"is_pano":false,"sequence":"s1",
				"geometry":{"type":"Point","coordinates":[101.6956,3.1423]},"creator":{"username":"amir"}},
				{"id":"bad","geometry":{"coordinates":[]}}]}`))
			return
		}
		t.Errorf("unexpected request %s", r.URL.Path)
	}))
	defer srv.Close()

	c := New("tok")
	c.baseURL = srv.URL

	imgs, err := c.Near(context.Background(), 3.1440, 101.6960, 200, 50)
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "OAuth tok" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if parts := strings.Split(gotBBox, ","); len(parts) != 4 || parts[0] >= parts[2] {
		t.Errorf("bbox = %q, want minLng,minLat,maxLng,maxLat", gotBBox)
	}
	if len(imgs) != 1 {
		t.Fatalf("got %d images, want 1 (the one without coordinates is dropped)", len(imgs))
	}
	im := imgs[0]
	if im.Lat != 3.1423 || im.Lng != 101.6956 || im.Creator != "amir" || im.SequenceID != "s1" {
		t.Errorf("image = %+v", im)
	}
	if im.CapturedAt == nil || !im.CapturedAt.Equal(time.UnixMilli(1700000000000)) {
		t.Errorf("captured_at = %v", im.CapturedAt)
	}

}

func TestPick(t *testing.T) {
	q := func(v float64) *float64 { return &v }
	at := func(id, seq, cam string, lat, quality float64) Image {
		return Image{ID: id, SequenceID: seq, CameraType: cam, IsPano: cam == "spherical", Lat: lat, Lng: 101.69, QualityScore: q(quality)}
	}
	imgs := []Image{
		at("dashcam", "a", "fisheye", 3.1400, 0.80),              // 0.80
		at("phone", "b", "perspective", 3.1420, 0.60),            // 0.90: best
		at("phone-same-drive", "b", "perspective", 3.1440, 0.70), // same sequence as "phone" but scores 1.00
		at("pano", "c", "spherical", 3.1460, 0.95),               // 0.45
		at("close", "d", "perspective", 3.14001, 0.55),           // 0.85 but 1 m from "dashcam"
		at("far", "e", "fisheye", 3.1480, 0.50),                  // 0.50
	}
	got := Pick(imgs, 4, 30)
	ids := []string{}
	for _, g := range got {
		ids = append(ids, g.ID)
	}
	// best first; one per drive keeps the higher-scoring frame of sequence b;
	// "close" (0.85) beats "dashcam" (0.80) and so takes that corner
	if strings.Join(ids, ",") != "phone-same-drive,close,far,pano" {
		t.Fatalf("picked %v", ids)
	}
}
