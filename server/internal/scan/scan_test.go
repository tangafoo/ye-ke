package scan

import (
	"testing"
	"time"

	"byakugan/internal/osm"
	"byakugan/internal/store"
)

func TestGeohash(t *testing.T) {
	// reference value from the geohash spec's worked example
	if got := geohash(57.64911, 10.40744, 6); got != "u4pruy" {
		t.Fatalf("geohash = %q, want u4pruy", got)
	}
}

func TestQualifierStates(t *testing.T) {
	cases := []struct {
		name  string
		areas []osm.Area
		want  []string
	}{
		{"councils dropped, klang valley widened",
			[]osm.Area{{Name: "Ulu Kelang", Level: 8}, {Name: "Majlis Perbandaran Ampang Jaya", Level: 7}, {Name: "Selangor", Level: 4}},
			[]string{"Kuala Lumpur", "Selangor"}},
		{"other state kept as is", []osm.Area{{Name: "Kuantan", Level: 6}, {Name: "Pahang", Level: 4}}, []string{"Pahang"}},
		{"no state level", []osm.Area{{Name: "Gombak", Level: 6}}, nil},
	}
	for _, c := range cases {
		got := qualifierStates(c.areas)
		if len(got) != len(c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
			}
		}
	}
}

func TestBuildQuery(t *testing.T) {
	got := buildQuery(`Jalan "Jelatek"`, []string{"Kuala Lumpur", "Selangor"}, []string{"polis", "tular"}, []string{"Balai Polis Trafik"})
	want := `"Jalan Jelatek" ("Kuala Lumpur" OR "Selangor") (polis OR tular) -"Balai Polis Trafik"`
	if got != want {
		t.Fatalf("buildQuery =\n %s\nwant\n %s", got, want)
	}
}

func TestClassify(t *testing.T) {
	way := func(tags map[string]string) osm.Element { return osm.Element{Type: "way", Tags: tags} }
	cases := []struct {
		name string
		e    osm.Element
		keep bool
		kind string
	}{
		{"Jalan Jelatek", way(map[string]string{"highway": "primary"}), true, "street"},
		{"Jalan 1/27E", way(map[string]string{"highway": "residential"}), false, ""},
		{"Seksyen 10", way(map[string]string{"place": "neighbourhood"}), false, ""},
		{"Fruits", way(map[string]string{"amenity": "bar"}), false, ""},
		{"Caltex", way(map[string]string{"amenity": "fuel", "brand": "Caltex"}), false, ""},
		{"Caltex Setiawangsa", way(map[string]string{"amenity": "fuel", "brand": "Caltex"}), true, "landmark"},
		{"Taman Setiawangsa", osm.Element{Type: "node", Tags: map[string]string{"place": "neighbourhood"}}, true, "area"},
		{"Footpath Lane", way(map[string]string{"highway": "footway"}), false, ""},
	}
	for _, c := range cases {
		got, ok := classify(c.e, c.name)
		if ok != c.keep || (ok && got.Kind != c.kind) {
			t.Fatalf("%s: keep=%v kind=%q, want keep=%v kind=%q", c.name, ok, got.Kind, c.keep, c.kind)
		}
	}
}

func TestPinnable(t *testing.T) {
	street := Target{Name: "Jalan Raja Chulan", Kind: "street"}
	alor := Target{Name: "Jalan Alor", Kind: "street"}
	ramlee := Target{Name: "Jalan P. Ramlee", Kind: "street"}
	cases := []struct {
		title string
		t     Target
		want  bool
	}{
		{"Polis serbu 4 premis hiburan sekitar Jalan Raja Chulan", street, true},
		{"Polis KL Operasi Tahan Pemandu Mabuk di Jalan Raja Laut", street, false},
		{"Kemalangan jalan raya: dua maut", street, true},             // "jalan raya" isn't a street
		{"Polis kesan suspek curi kabel di Jalan Raja", street, true}, // stops early, no conflict
		{"Mayat ditemui di Jalan P Ramlee", ramlee, true},             // punctuation-insensitive
		{"166 PATI ditahan dalam dua operasi", alor, true},
		{"Remaja 14 Tahun Maut Terbabit Kemalangan #MGFlash Ketua Polis Daerah Kuala Selangor, Superintendan Azaharudin Tajudin berkata pihaknya menerima panggilan", alor, false},
		{"Mayat Lelaki Berulat Di Jalan Alor #MGFlash Mayat seorang lelaki dalam keadaan berulat ditemui di hutan kecil dekat kawasan berbukit, Kuala Lumpur.", alor, true},
	}
	for _, c := range cases {
		if got := pinnable(c.title, c.t); got != c.want {
			t.Errorf("pinnable(%q, %q) = %v, want %v", c.title, c.t.Name, got, c.want)
		}
	}
}

func TestCluster(t *testing.T) {
	day := func(d int) *time.Time { v := time.Date(2026, 8, d, 9, 0, 0, 0, time.UTC); return &v }
	at := func(title, outlet string, d int) store.Find {
		return store.Find{Title: title, Outlet: outlet, PlaceName: "Jalan Sultan Ismail",
			Lat: 3.1500, Lng: 101.7100, Precision: 4, PublishedAt: day(d)}
	}
	finds := []store.Find{
		at("Tiga maut dilanggar lori imigresen", "Utusan", 18),
		at("Kes 3 maut: Polis tahan pemandu lori Imigresen", "Harian Metro", 18),
		at("Tiga sekeluarga maut dirempuh lori di Jalan Sultan Ismail", "BH", 19),
		at("Nahas tiga beranak maut: Pemandu 'Black Maria' Imigresen digantung tugas", "Kosmo", 20),
		at("Pusat hiburan Jalan Sultan Ismail diserbu", "Sinar", 19),                 // same street, same week, other story
		at("Tiga maut dilanggar lori imigresen", "Utusan", 1),                        // same words, outside the window
		{Title: "Lori imigresen maut tiga", Outlet: "X", Lat: 3.1500, Lng: 101.7100}, // undated
	}
	groups := Cluster(finds)
	if len(groups) != 4 {
		t.Fatalf("got %d groups %v, want 4 (lorry crash, raid, old article, undated)", len(groups), groups)
	}
	if len(groups[0]) != 4 {
		t.Fatalf("lorry crash group = %v, want the first four", groups[0])
	}

	plans := PlanIncidents(finds, groups)
	if plans[0].Incident.Status != "corroborated" || plans[0].Incident.OutletCount != 4 {
		t.Fatalf("lorry crash plan = %+v, want corroborated by 4 outlets", plans[0].Incident)
	}
	if plans[1].Incident.Status != "reported" {
		t.Fatalf("raid plan status = %q, want reported", plans[1].Incident.Status)
	}
}

func TestClusterStemmedSameWeek(t *testing.T) {
	day := func(d int) *time.Time { v := time.Date(2022, 5, d, 9, 0, 0, 0, time.UTC); return &v }
	at := func(title string, d int) store.Find {
		return store.Find{Title: title, PlaceName: "Jalan Bukit Bintang", Lat: 3.146, Lng: 101.71, PublishedAt: day(d)}
	}
	finds := []store.Find{
		at("Lelaki tempatan ditemui mati dengan kesan tembakan", 3),
		at("Kes tembak: Polis buru dua lagi suspek", 4),
		at("Polis tahan suspek utama kes tembak di Bukit Bintang", 5),
		at("Medan selera tawar khidmat seks diserbu", 5), // same days, different story
	}
	if groups := Cluster(finds); len(groups) != 2 || len(groups[0]) != 3 {
		t.Fatalf("groups = %v, want the three shooting reports together and the raid alone", groups)
	}
}

func TestWordsMatch(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{unprefix("ditembak"), "tembakan", true},
		{unprefix("dilanggar"), "langgar", true},
		{"maut", "mautnya", false}, // too short to prefix-match
		{"serbu", "serbuan", true},
		{"tembak", "tempat", false},
	}
	for _, c := range cases {
		if got := wordsMatch(c.a, c.b); got != c.want {
			t.Errorf("wordsMatch(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestClusterAcrossLanguages(t *testing.T) {
	day := func(d int) *time.Time { v := time.Date(2024, 11, d, 9, 0, 0, 0, time.UTC); return &v }
	at := func(title, place string, d int) store.Find {
		return store.Find{Title: title, PlaceName: place, Lat: 3.146, Lng: 101.71, PublishedAt: day(d)}
	}
	cases := []struct {
		name string
		a, b store.Find
		want bool
	}{
		{"shot dead ↔ ditembak mati",
			at("Empat penjenayah warga Vietnam ditembak mati di Jalan Imbi", "Jalan Imbi", 12),
			at("Four Vietnamese robbers shot dead in Jalan Imbi", "Jalan Imbi", 12), true},
		{"money anchor alone, days apart",
			at("Kedai emas rugi RM9.7 juta disamun", "Jalan Tun Sambanthan", 10),
			at("Brickfields jeweller hit for RM9.7 million, cops hunt four", "Jalan Tun Sambanthan", 16), true},
		{"op codename anchor",
			at("Dua lelaki lari ketika Op Tapis Khas", "Jalan Sultan Salahuddin", 3),
			at("Narcotics officers in Op Tapis chase car", "Jalan Sultan Salahuddin", 9), true},
		{"lorry crash in both languages",
			at("Tiga sekeluarga maut dirempuh lori imigresen", "Jalan Sultan Ismail", 18),
			at("Family of three killed after immigration lorry rams car", "Jalan Sultan Ismail", 19), true},
		{"different stories, same street, same week",
			at("Polis serbu pusat hiburan, 60 positif dadah", "Jalan Sultan Ismail", 18),
			at("Family of three killed after immigration lorry rams car", "Jalan Sultan Ismail", 25), false},
	}
	for _, c := range cases {
		got := len(Cluster([]store.Find{c.a, c.b})) == 1
		if got != c.want {
			t.Errorf("%s: together=%v, want %v (words %v | %v)", c.name, got, c.want,
				distinctWords(c.a.Title, c.a.PlaceName), distinctWords(c.b.Title, c.b.PlaceName))
		}
	}
}

func TestAnchors(t *testing.T) {
	cases := map[string][]string{
		"Kedai emas rugi RM9.7 juta disamun":   {"@!rm9.7m"},
		"Jeweller loses RM9.7 million":         {"@!rm9.7m"},
		"Tipu pakej Umrah RM250,000":           {"@!rm250000"},
		"Remaja 14 tahun maut":                 {"@age14"},
		"14-year-old killed in crash":          {"@age14"},
		"Pegawai lari dalam Op Tapis Khas":     {"@!optapis"},
		"Seramai 1,101 PATI dicekup pada 2023": {"@n1101"}, // the year is skipped
	}
	for title, want := range cases {
		got := anchors(title)
		if len(got) != len(want) {
			t.Errorf("anchors(%q) = %v, want %v", title, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("anchors(%q) = %v, want %v", title, got, want)
			}
		}
	}
}

func TestElsewhere(t *testing.T) {
	local := []string{"brisbane", "queensland", "brisbane city"}
	cases := map[string]bool{
		"Armed robbery, Broadbeach":                     true,
		"Fatal traffic crash, Bundaberg North":          true,
		"Stolen vehicle, Brisbane City":                 false,
		"Queen Street Mall gunman jailed for 10 years":  false, // no suburb suffix
		"Man charged, police say he fled the scene now": false, // a clause, not a suburb
	}
	for title, want := range cases {
		if got := elsewhere(title, local); got != want {
			t.Errorf("elsewhere(%q) = %v, want %v", title, got, want)
		}
	}
}

func TestQualifierCities(t *testing.T) {
	got := qualifierCities([]osm.Area{{Name: "City of Brisbane", Level: 6}, {Name: "Queensland", Level: 4}})
	if len(got) != 1 || got[0] != "Brisbane" {
		t.Fatalf("qualifierCities = %v, want [Brisbane]", got)
	}
}
