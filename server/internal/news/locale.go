package news

import "strings"

type headPos uint8

const (
	headInitial headPos = iota
	headFinal
	headEither
)

type Locale struct {
	Code string

	Heads   map[string]Precision
	HeadPos headPos

	Preps     map[string]bool
	Blocklist map[string]bool

	BBox [4]float64 // minLat, minLng, maxLat, maxLng

	maxHeadTokens int

	Gate func(head, text string, loc []int) bool
}

func (l Locale) compile() *Locale {
	l.maxHeadTokens = 1
	heads := make(map[string]Precision, len(l.Heads))

	for h, prec := range l.Heads {
		f := strings.Fields(h)
		if len(f) > l.maxHeadTokens {
			l.maxHeadTokens = len(f)
		}
		heads[strings.ToLower(strings.Join(f, " "))] = prec
	}
	l.Heads = heads

	block := make(map[string]bool, len(l.Blocklist))
	for b := range l.Blocklist {
		block[normKey(b)] = true
	}
	l.Blocklist = block

	return &l
}

func normKey(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func (l *Locale) contains(lat, lng float64) bool {
	return lat >= l.BBox[0] && lat <= l.BBox[2] && lng >= l.BBox[1] && lng <= l.BBox[3]
}

var localeMS = Locale{
	Code:    "ms-MY",
	HeadPos: headInitial,
	Gate:    msGate,
	Heads: map[string]Precision{
		"taman": PrecStreet, "kampung": PrecStreet, "kg": PrecStreet,
		"jalan": PrecStreet, "lorong": PrecStreet, "persiaran": PrecStreet,
		"seksyen": PrecStreet,

		"bandar": PrecTown, "pekan": PrecTown, "mukim": PrecTown,

		"stadium": PrecVenue, "hospital": PrecVenue, "sekolah": PrecVenue,
		"masjid": PrecVenue, "kompleks": PrecVenue, "pasar": PrecVenue,
		"ipd": PrecVenue, "ipk": PrecVenue, "balai polis": PrecVenue,
	},
	Preps: map[string]bool{"di": true, "ke": true, "dari": true, "berhampiran": true},
	Blocklist: map[string]bool{
		"bukit aman": true, "dewan rakyat": true, "parlimen": true,
		"petaling jaya": true, "kuala lumpur": true, "putrajaya": true,
		"malaysia": true,
	},
	BBox: [4]float64{0.85, 99.6, 7.4, 119.3},
}.compile()

var localeAny = Locale{
	Code:    "any",
	HeadPos: headEither,
	BBox:    [4]float64{-90, -180, 90, 180},
}.compile()

var locales = []*Locale{localeMS}

func LocaleFor(lat, lng float64) *Locale {
	for _, l := range locales {
		if l.contains(lat, lng) {
			return l
		}
	}
	return localeAny
}

func (l *Locale) headAt(s string, toks []token, i int) (prec Precision, n int, ok bool) {
	for span := min(l.maxHeadTokens, len(toks)-i); span >= 1; span-- {
		word := s[toks[i].start:toks[i+span-1].end]
		if span > 1 {
			word = strings.Join(strings.Fields(word), " ")
		}
		if p, hit := l.Heads[strings.ToLower(word)]; hit {
			return p, span, true
		}
	}
	return PrecNone, 0, false
}
