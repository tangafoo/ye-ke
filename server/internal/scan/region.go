// Package scan holds the scan lanes: each one digs located items out of a
// source and turns them into store.Finds. The HTTP handler only orchestrates.
package scan

import "byakugan/internal/news"

// Region is a country the scanner supports: where it is, and the news
// languages that cover it. Malaysia is searched in Malay and English; the same
// incident reported in both is glued together by clustering (see lexicon.go).
type Region struct {
	Code  string
	BBox  [4]float64 // minLat, minLng, maxLat, maxLng
	Langs []Lang

	// QualifyByCity: qualify every search with the city ("Queen Street"
	// "Brisbane"). Street names repeat in every Australian/German/French town.
	// Off for Malaysia, where the state is used and long Malay street names
	// are specific enough on their own.
	QualifyByCity bool
}

// Lang is one news edition and the words that mark a cop/crime story in it.
type Lang struct {
	Edition news.Edition

	// QueryTerms go into the search as an OR group (keep it short).
	QueryTerms []string
	// TitleTerms gate results: a headline must contain one (lowercase, substring,
	// so "kejar" also catches "dikejar").
	TitleTerms []string
	// RejectTerms drop a headline even if it passed TitleTerms: police notices
	// that aren't incidents (road closures, diversions).
	RejectTerms []string
	// Exclude phrases are subtracted in the query itself. Used for places that
	// stories mention as a contact address rather than where anything happened.
	Exclude []string
}

var (
	langMS = Lang{
		Edition:    news.EditionMY,
		QueryTerms: []string{"polis", "tular", "samun", "rompak", "ragut", "kejar", "tembak", "bunuh", "dadah", "gaduh"},
		TitleTerms: []string{
			"polis", "pdrm", "ipd", "tular", "viral", "samun", "rompak", "ragut", "curi",
			"gaduh", "belasah", "serang", "serbu", "kejar", "tembak", "pistol", "parang",
			"senjata", "bunuh", "mayat", "maut", "dadah", "tahan", "berkas", "cekup",
			"reman", "suspek", "jenayah", "kongsi gelap", "langgar", "rempuh", "culik",
			"rogol", "cabul", "penipuan", "scam", "police", "arrest", "robbery", "murder",
		},
		RejectTerms: []string{"ditutup", "penutupan", "tutup jalan", "jalan utama", "lencong", "laluan alternatif", "pelan trafik"},
		// KL traffic stories end with "contact Balai Polis Trafik Jalan Tun H.S. Lee",
		// which pins highway crashes from anywhere in the city to Chinatown
		Exclude: []string{"Balai Polis Trafik"},
	}

	// Malaysia's English press (The Star, NST, FMT, Malay Mail, The Sun)
	langMSEnglish = Lang{
		Edition:    news.EditionMYEnglish,
		QueryTerms: []string{"police", "cops", "robbery", "murder", "shot", "stabbed", "chase", "arrested", "viral", "drugs"},
		TitleTerms: []string{
			"police", "cops", "cop ", "pdrm", "arrest", "nabbed", "detained", "remand",
			"charged", "robbery", "robbed", "snatch", "murder", "killed", "dead", "death",
			"shot", "shooting", "stab", "machete", "parang", "gang", "drug", "raid",
			"crash", "rammed", "chase", "viral", "body", "theft", "stolen", "assault",
			"brawl", "fight", "kidnap", "rape", "molest", "scam", "suspect",
		},
		RejectTerms: []string{"road closure", "roads closed", "closures", "diversion", "traffic plan"},
		Exclude:     []string{"Traffic Police Station"},
	}
)

// Order matters: the DE and FR boxes overlap along the Rhine (roughly
// Strasbourg–Saarbrücken), and the first match wins there.
var Regions = []Region{
	{
		Code:  "MY",
		BBox:  [4]float64{0.85, 99.6, 7.4, 119.3},
		Langs: []Lang{langMS, langMSEnglish},
	},
	{
		Code:          "AU",
		QualifyByCity: true,
		BBox:          [4]float64{-43.7, 112.9, -10.0, 153.7},
		Langs: []Lang{{
			Edition:    news.EditionAU,
			QueryTerms: []string{"police", "arrested", "robbery", "assault", "stabbing", "shooting", "chase", "charged"},
			TitleTerms: []string{
				"police", "cop", "arrest", "charged", "robbery", "robbed", "assault", "stab",
				"shoot", "shot", "chase", "pursuit", "murder", "dead", "death", "drug",
				"crash", "burglar", "break-in", "theft", "stolen", "attack", "gang", "knife",
			},
			RejectTerms: []string{"road closure", "closures", "diversion"},
		}},
	},
	{
		Code:          "DE",
		QualifyByCity: true,
		BBox:          [4]float64{47.27, 5.87, 55.06, 15.04},
		Langs: []Lang{{
			Edition:    news.EditionDE,
			QueryTerms: []string{"Polizei", "Festnahme", "Raub", "Überfall", "Messer", "Einbruch", "Verfolgungsjagd"},
			TitleTerms: []string{
				"polizei", "festnahme", "festgenommen", "raub", "überfall", "messer", "schüsse",
				"einbruch", "verfolgungsjagd", "tot", "mord", "angriff", "drogen", "unfall",
				"täter", "gestohlen", "diebstahl", "schlägerei",
			},
			RejectTerms: []string{"sperrung", "umleitung"},
		}},
	},
	{
		Code:          "FR",
		QualifyByCity: true,
		BBox:          [4]float64{41.3, -5.2, 51.1, 9.6},
		Langs: []Lang{{
			Edition:    news.EditionFR,
			QueryTerms: []string{"police", "interpellé", "vol", "agression", "braquage", "couteau", "cambriolage"},
			TitleTerms: []string{
				"police", "policier", "gendarme", "interpell", "vol", "agress", "braquage",
				"couteau", "cambriol", "mort", "meurtre", "tué", "drogue", "rixe", "fusillade",
				"accident", "suspect", "garde à vue",
			},
			RejectTerms: []string{"fermeture", "déviation"},
		}},
	},
}

func RegionFor(lat, lng float64) (Region, bool) {
	for _, r := range Regions {
		if lat >= r.BBox[0] && lat <= r.BBox[2] && lng >= r.BBox[1] && lng <= r.BBox[3] {
			return r, true
		}
	}
	return Region{}, false
}
