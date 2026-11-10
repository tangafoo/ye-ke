package scan

import (
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strings"

	"byakugan/internal/news"
	"byakugan/internal/osm"
	"byakugan/internal/store"
)

// Target is a named thing near the tap that's worth searching the news for.
// A story that names it is pinned at the target's coordinates.
type Target struct {
	Name      string
	Lat, Lng  float64
	Precision news.Precision
	Kind      string // landmark | area | street
	distM     float64
	roadRank  int // streets only: 0 trunk/primary … 3 minor
}

const (
	MaxTargets   = 12
	maxLandmarks = 4 // streets are where Malaysian crime reports pin incidents
)

// amenities that crime stories actually name. Restaurants and cafés are left
// out on purpose: there are hundreds per square km downtown and reports say
// "sebuah restoran di Jalan X", not the restaurant's name.
var landmarkAmenities = map[string]bool{
	"marketplace": true, "bus_station": true, "place_of_worship": true, "school": true,
	"college": true, "university": true, "hospital": true, "nightclub": true, "bar": true,
	"pub": true, "cinema": true, "townhall": true, "courthouse": true, "prison": true,
	"community_centre": true, "fuel": true, "food_court": true, "casino": true,
}

// keep in sync with the tourism filter in osm.Around
var landmarkTourism = map[string]bool{"attraction": true, "museum": true, "viewpoint": true}

// road class → rank. Main roads first: that's where incidents get reported
// ("dikejar di Jalan Jelatek"), and their names are the ones news uses.
var streetClasses = map[string]int{
	"trunk": 0, "primary": 0, "secondary": 1, "tertiary": 2,
	"unclassified": 3, "residential": 3, "pedestrian": 3, "living_street": 3,
}

// too generic to search on their own: they match stories nationwide
var genericNames = map[string]bool{
	"surau": true, "masjid": true, "sekolah": true, "klinik": true, "tandas": true,
	"toilet": true, "parking": true, "letak kereta": true, "atm": true, "bank": true,
	"restoran": true, "restaurant": true, "kedai": true, "cafe": true, "balai polis": true,
	"pondok polis": true, "bus stop": true, "hentian bas": true, "police station": true,
}

// "Jalan 1/2", "Lorong 3" — numbered streets exist in every town
var numberedStreetRE = regexp.MustCompile(`(?i)^(jalan|lorong|persiaran|lebuh|jln)\s+\d[\d\s/\-]*[a-z]?$`)

// "Seksyen 10" is also a statute section ("didakwa mengikut Seksyen 10"), so a
// bare one matches court reports nationwide. Skipped until the Seksyen gate exists.
var bareSeksyenRE = regexp.MustCompile(`(?i)^seksyen\s+\d+[a-z]?$`)

// FromOSM splits what Overpass found into police-station finds (stored as-is,
// exact) and the named targets the news lane should search for.
func FromOSM(around osm.Around, lat, lng float64) (police []store.Find, targets []Target) {
	best := map[string]Target{} // lowercase name → nearest element with that name

	for _, e := range around.Features {
		eLat, eLng, ok := e.Point()
		if !ok {
			continue
		}
		name := strings.TrimSpace(e.Tags["name"])

		if e.Tags["amenity"] == "police" {
			title := name
			if title == "" {
				title = "Police station"
			}
			raw, _ := json.Marshal(e.Tags)
			police = append(police, store.Find{
				SourceKind: "osm_police",
				SourceID:   e.Key(),
				Lat:        eLat,
				Lng:        eLng,
				Precision:  int16(news.PrecVenue),
				PlaceName:  name,
				Title:      title,
				URL:        "https://www.openstreetmap.org/" + e.Key(),
				Outlet:     "OpenStreetMap",
				Raw:        raw,
			})
			continue // a station's name is where police are, not where stories happen
		}

		t, ok := classify(e, name)
		if !ok {
			continue
		}
		t.Lat, t.Lng = eLat, eLng
		t.distM = distanceM(lat, lng, eLat, eLng)

		key := strings.ToLower(name)
		if prev, seen := best[key]; seen {
			t.roadRank = min(t.roadRank, prev.roadRank) // a road is as major as its biggest stretch
			if prev.distM <= t.distM {
				t.Lat, t.Lng, t.distM = prev.Lat, prev.Lng, prev.distM // keep the stretch nearest the tap
			}
		}
		best[key] = t
	}

	var landmarks, streets []Target
	for _, t := range best {
		if t.Kind == "street" {
			streets = append(streets, t)
		} else {
			landmarks = append(landmarks, t)
		}
	}

	// Streets: main roads first; then more words = more specific ("Jalan Tun Tan
	// Cheng Lock" beats "Jalan Sultan", which also matches "Jalan Sultan Salahuddin").
	sort.Slice(streets, func(i, j int) bool {
		if streets[i].roadRank != streets[j].roadRank {
			return streets[i].roadRank < streets[j].roadRank
		}
		wi, wj := len(strings.Fields(streets[i].Name)), len(strings.Fields(streets[j].Name))
		if wi != wj {
			return wi > wj
		}
		return streets[i].distM < streets[j].distM
	})
	// Landmarks: the opposite. News says "Pasar Seni", not the full official name
	// ("Historical Museum of Traditional Chinese Medicine in Malaysia" never hits).
	sort.Slice(landmarks, func(i, j int) bool {
		wi, wj := len(strings.Fields(landmarks[i].Name)), len(strings.Fields(landmarks[j].Name))
		if wi != wj {
			return wi < wj
		}
		return landmarks[i].distM < landmarks[j].distM
	})

	if len(landmarks) > maxLandmarks {
		landmarks = landmarks[:maxLandmarks]
	}
	for _, s := range streets {
		if len(targets) >= MaxTargets-len(landmarks) {
			break
		}
		targets = append(targets, s)
	}
	targets = append(targets, landmarks...)
	return police, targets
}

func classify(e osm.Element, name string) (Target, bool) {
	if len([]rune(name)) < 5 || genericNames[strings.ToLower(name)] ||
		numberedStreetRE.MatchString(name) || bareSeksyenRE.MatchString(name) {
		return Target{}, false
	}
	// a bare chain name ("McDonald's", "Shell") matches that chain's stories anywhere;
	// "Caltex Setiawangsa" is fine
	if brand := e.Tags["brand"]; brand != "" && strings.EqualFold(name, strings.TrimSpace(brand)) {
		return Target{}, false
	}

	tags := e.Tags
	switch {
	case landmarkAmenities[tags["amenity"]], landmarkTourism[tags["tourism"]],
		tags["shop"] == "mall", tags["railway"] == "station":
		// one-word shop/bar names ("Fruits", "Penrose") are ordinary words in news
		if len(strings.Fields(name)) < 2 {
			return Target{}, false
		}
		return Target{Name: name, Precision: news.PrecVenue, Kind: "landmark"}, true

	case tags["place"] == "neighbourhood" || tags["place"] == "quarter" || tags["place"] == "hamlet":
		return Target{Name: name, Precision: news.PrecStreet, Kind: "area"}, true

	case tags["place"] == "suburb" || tags["place"] == "village":
		return Target{Name: name, Precision: news.PrecTown, Kind: "area"}, true

	case e.Type == "way":
		if rank, ok := streetClasses[tags["highway"]]; ok {
			return Target{Name: name, Precision: news.PrecStreet, Kind: "street", roadRank: rank}, true
		}
	}
	return Target{}, false
}

// DistanceM is the great-circle (haversine) distance between two points in metres.
func DistanceM(lat1, lng1, lat2, lng2 float64) float64 { return distanceM(lat1, lng1, lat2, lng2) }

func distanceM(lat1, lng1, lat2, lng2 float64) float64 {
	const earthR = 6_371_000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthR * math.Asin(math.Sqrt(a))
}
