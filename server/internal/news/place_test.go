package news

import (
	"strings"
	"testing"
)

// Helpers ────────────────────────────────────────────────────────────────────

func names(cands []PlaceCandidate) []string {
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.Name)
	}
	return out
}

// hasName returns the highest-precision candidate with this name. placesIn
// over-generates — the same string can arrive from both the prep path (unranked)
// and the head path (ranked) — and ExtractPlaces is what collapses them, so the
// best reading is what callers care about.
func hasName(cands []PlaceCandidate, want string) (PlaceCandidate, bool) {
	var best PlaceCandidate
	var found bool
	for _, c := range cands {
		if c.Name == want && (!found || c.Prec > best.Prec) {
			best, found = c, true
		}
	}
	return best, found
}

// tokenize ───────────────────────────────────────────────────────────────────

func TestTokenizeSplitsAndFlags(t *testing.T) {
	const s = "Polis ke Taman. Melati, Ampang ditahan"

	want := []struct {
		text  string
		brk   bool
		comma bool
	}{
		{"Polis", false, false},
		{"ke", false, false},
		{"Taman", false, false},
		{"Melati", true, false}, // after the full stop
		{"Ampang", false, true}, // after the comma
		{"ditahan", false, false},
	}

	toks := tokenize(s)
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens %q, want %d", len(toks), texts(s, toks), len(want))
	}
	for i, w := range want {
		got := toks[i]
		if got.text(s) != w.text {
			t.Errorf("token %d = %q, want %q", i, got.text(s), w.text)
		}
		if got.breakBefore != w.brk {
			t.Errorf("token %d %q breakBefore = %v, want %v", i, w.text, got.breakBefore, w.brk)
		}
		if got.commaBefore != w.comma {
			t.Errorf("token %d %q commaBefore = %v, want %v", i, w.text, got.commaBefore, w.comma)
		}
	}
}

func texts(s string, toks []token) []string {
	out := make([]string, len(toks))
	for i, tk := range toks {
		out[i] = tk.text(s)
	}
	return out
}

func TestTokenizeEdges(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"separators only", " ,.;: ", nil},
		{"no trailing separator", "Jalan Ampang", []string{"Jalan", "Ampang"}},
		{"leading separator", "  Jalan", []string{"Jalan"}},
		{"apostrophe stays inside", "King's Road", []string{"King's", "Road"}},
		{"hyphen stays inside", "Sungai Petani-Alor", []string{"Sungai", "Petani-Alor"}},
		{"em dash splits", "Ampang — Hilir", []string{"Ampang", "Hilir"}},
		{"parens split", "IPD Serian (183)", []string{"IPD", "Serian", "183"}},
		{"unicode letters", "Ñusta Ampang", []string{"Ñusta", "Ampang"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := texts(tc.in, tokenize(tc.in))
			if len(got) != len(tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("token %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// A token's offsets must index back into the exact source substring.
func TestTokenOffsetsRoundTrip(t *testing.T) {
	const s = "Kejadian di Taman Titiwangsa, Kluang. Ñusta juga"
	for _, tk := range tokenize(s) {
		if tk.start < 0 || tk.end > len(s) || tk.start >= tk.end {
			t.Fatalf("bad offsets %d:%d for %q", tk.start, tk.end, s)
		}
		if strings.ContainsFunc(tk.text(s), isSep) {
			t.Errorf("token %q contains a separator", tk.text(s))
		}
	}
}

// runForward / runBackward ───────────────────────────────────────────────────

func TestRunForward(t *testing.T) {
	tests := []struct {
		name string
		s    string
		from int
		opts runOpts
		want string // the phrase the run covers, "" for no run
	}{
		{"plain run", "di Jalan Ampang Hilir semalam", 1, runOpts{limit: 4}, "Jalan Ampang Hilir"},
		{"stops at full stop", "Polis ke Taman. Melati ditahan", 2, runOpts{limit: 4}, "Taman"},
		{"stops at newline", "Jalan Ampang\nKuala Lumpur", 1, runOpts{limit: 4}, "Ampang"},
		{"stops at paren", "IPD Serian (183) tinggi", 1, runOpts{limit: 4}, "Serian"},
		{"lowercase halts immediately", "di taman melati", 1, runOpts{limit: 4}, ""},
		{"digits allowed after start", "di Seksyen 7 Shah Alam", 1, runOpts{limit: 4}, "Seksyen 7 Shah Alam"},
		{"comma crossed when allowed", "di Taman Titiwangsa, Kluang lagi", 1, runOpts{limit: 4}, "Taman Titiwangsa, Kluang"},
		{"comma stops when forbidden", "di Kluang, Seorang lelaki", 1, runOpts{limit: 4, stopOnComma: true}, "Kluang"},
		{"digit start rejected", "dari 1980 hingga kini", 1, runOpts{limit: 4, needAlphaStart: true}, ""},
		{"digit start allowed without opt", "dari 1980 hingga kini", 1, runOpts{limit: 4}, "1980"},
		{"from past end", "Jalan Ampang", 9, runOpts{limit: 4}, ""},
		{"from at end", "Jalan Ampang", 2, runOpts{limit: 4}, ""},
		{"empty string", "", 0, runOpts{limit: 4}, ""},
		{"zero limit", "di Jalan Ampang", 1, runOpts{limit: 0}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			toks := tokenize(tc.s)
			n := runForward(tc.s, toks, tc.from, tc.opts)

			got := ""
			if n > 0 {
				got = phrase(tc.s, toks, tc.from, tc.from+n-1)
			}
			if got != tc.want {
				t.Errorf("runForward(%q, from=%d) covered %q, want %q", tc.s, tc.from, got, tc.want)
			}
		})
	}
}

// The limit must not leave a comma segment half-consumed.
func TestRunForwardTrimsPartialCommaSegment(t *testing.T) {
	const s = "ditahan di Kampung Sungai Ara, Bayan Lepas"
	toks := tokenize(s)

	// head "Kampung" is token 2, so the name run starts at 3 with 3 to spend:
	// Sungai, Ara, Bayan — which would split "Bayan Lepas".
	n := runForward(s, toks, 3, runOpts{limit: 3})

	got := phrase(s, toks, 3, 3+n-1)
	if got != "Sungai Ara" {
		t.Errorf("run covered %q, want %q (must not end mid-segment)", got, "Sungai Ara")
	}
}

func TestRunBackward(t *testing.T) {
	tests := []struct {
		name string
		s    string
		to   int
		want string
	}{
		{"single name token", "on Murray Street", 1, "Murray"},
		{"two name tokens", "at Royal Hobart Hospital", 2, "Royal Hobart"},
		{"stops at sentence start", "ditahan. Melati Street", 1, "Melati"},
		{"lowercase halts", "the murray Street", 1, ""},
		{"to = 0", "Murray Street", 0, "Murray"},
		{"negative index is safe", "Street alone", -1, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			toks := tokenize(tc.s)
			n := runBackward(tc.s, toks, tc.to)

			got := ""
			if n > 0 {
				got = phrase(tc.s, toks, tc.to-n+1, tc.to)
			}
			if got != tc.want {
				t.Errorf("runBackward(%q, to=%d) covered %q, want %q", tc.s, tc.to, got, tc.want)
			}
		})
	}
}

// phrase ─────────────────────────────────────────────────────────────────────

// phrase rebuilds from tokens, so punctuation that sat between them is dropped —
// except a comma, which is meaningful inside an address.
func TestPhraseDropsPunctuationKeepsComma(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"comma survives", "Taman Titiwangsa, Kluang", "Taman Titiwangsa, Kluang"},
		{"paren dropped", "Hospital Pulau Pinang (HPP", "Hospital Pulau Pinang HPP"},
		{"double space collapsed", "Balai  Polis   Sentul", "Balai Polis Sentul"},
		{"newline collapsed", "Jalan\nAmpang", "Jalan Ampang"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			toks := tokenize(tc.in)
			if got := phrase(tc.in, toks, 0, len(toks)-1); got != tc.want {
				t.Errorf("phrase = %q, want %q", got, tc.want)
			}
		})
	}
}

// ExtractCaption ─────────────────────────────────────────────────────────────

func TestExtractCaption(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"strips tags and trailing credit",
			`<figure><img src="x"><figcaption class="wp-caption-text">Kejadian di <b>Taman Melati</b> semalam. (Gambar Bernama)</figcaption></figure>`,
			"Kejadian di Taman Melati semalam.",
		},
		{
			"keeps parens that are not a credit",
			`<figcaption>Bacaan IPU di IPD Serian (183) tinggi</figcaption>`,
			"Bacaan IPU di IPD Serian (183) tinggi",
		},
		{
			"mid-text credit is not stripped",
			`<figcaption>Foto lama (Gambar fail) masih relevan</figcaption>`,
			"Foto lama (Gambar fail) masih relevan",
		},
		{"no figcaption", `<p>Tiada gambar di sini</p>`, ""},
		{"empty figcaption", `<figcaption></figcaption>`, ""},
		{"credit only", `<figcaption>(Gambar Bernama)</figcaption>`, ""},
		{
			"first figcaption wins",
			`<figcaption>Pertama</figcaption><figcaption>Kedua</figcaption>`,
			"Pertama",
		},
		{
			"entities unescaped",
			`<figcaption>Ibu &amp; bapa mangsa</figcaption>`,
			"Ibu & bapa mangsa",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractCaption(tc.in); got != tc.want {
				t.Errorf("ExtractCaption = %q, want %q", got, tc.want)
			}
		})
	}
}

// splitDateline ──────────────────────────────────────────────────────────────

func TestSplitDateline(t *testing.T) {
	tests := []struct {
		name, in, wantLine, wantRest string
	}{
		{
			"all caps dateline",
			"GEORGE TOWN: Seorang bayi maut.",
			"GEORGE TOWN", "Seorang bayi maut.",
		},
		{
			"two words",
			"KUALA LUMPUR: Polis sedang menyiasat.",
			"KUALA LUMPUR", "Polis sedang menyiasat.",
		},
		{"no dateline", "Seorang bayi maut di Kluang.", "", "Seorang bayi maut di Kluang."},
		{
			"mixed case is not a dateline",
			"George Town: Seorang bayi maut.",
			"", "George Town: Seorang bayi maut.",
		},
		{
			"colon later in the body is ignored",
			"Polis berkata: siasatan diteruskan.",
			"", "Polis berkata: siasatan diteruskan.",
		},
		{"empty", "", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			line, rest := splitDateline(tc.in)
			if line != tc.wantLine {
				t.Errorf("dateline = %q, want %q", line, tc.wantLine)
			}
			if rest != tc.wantRest {
				t.Errorf("rest = %q, want %q", rest, tc.wantRest)
			}
		})
	}
}

// SourcesFrom ────────────────────────────────────────────────────────────────

// The figcaption sits before the first <p>, so it must be removed before the
// dateline anchor sees the body. Measured: without this, 0/50 live FMT articles
// yielded a dateline; with it, 49/50.
func TestSourcesFromStripsCaptionBeforeDateline(t *testing.T) {
	const html = `<figure><img src="x"><figcaption>Jawatankuasa yang dianggotai penasihat.</figcaption></figure>` +
		`<p>PETALING JAYA: Jawatankuasa khas akan mengemukakan laporan.</p>`

	src := SourcesFrom("Tajuk berita", html)

	if src.Caption != "Jawatankuasa yang dianggotai penasihat." {
		t.Errorf("Caption = %q", src.Caption)
	}
	if src.Dateline != "PETALING JAYA" {
		t.Errorf("Dateline = %q, want %q", src.Dateline, "PETALING JAYA")
	}
	if src.Title != "Tajuk berita" {
		t.Errorf("Title = %q", src.Title)
	}
	if strings.Contains(src.Body, "Jawatankuasa yang dianggotai") {
		t.Errorf("Body still contains the caption text: %q", src.Body)
	}
	if strings.Contains(src.Body, "PETALING JAYA") {
		t.Errorf("Body still contains the dateline: %q", src.Body)
	}
}

func TestSourcesFromNoCaptionNoDateline(t *testing.T) {
	src := SourcesFrom("Tajuk", `<p>Seorang lelaki ditahan di Kluang.</p>`)

	if src.Caption != "" {
		t.Errorf("Caption = %q, want empty", src.Caption)
	}
	if src.Dateline != "" {
		t.Errorf("Dateline = %q, want empty", src.Dateline)
	}
	if src.Body != "Seorang lelaki ditahan di Kluang." {
		t.Errorf("Body = %q", src.Body)
	}
}

// placesIn ───────────────────────────────────────────────────────────────────

func TestPlacesInHeadPath(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		wantName string
		wantPrec Precision
	}{
		{"venue", "Kejadian di Stadium Darul Makmur, Kuantan.", "Stadium Darul Makmur, Kuantan", PrecVenue},
		{"street", "Rumah teres di Taman Titiwangsa, Kluang.", "Taman Titiwangsa, Kluang", PrecStreet},
		{"town", "Beliau tinggal di Bandar Baru Sri Petaling.", "Bandar Baru Sri Petaling", PrecTown},
		{"multi-token head", "Ditahan di Balai Polis Sentul pagi tadi.", "Balai Polis Sentul", PrecVenue},
		{"abbreviated head", "Mangsa dibawa ke IPD Serian.", "IPD Serian", PrecVenue},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := localeMS.placesIn(tc.s, SourceBody)
			c, ok := hasName(got, tc.wantName)
			if !ok {
				t.Fatalf("no candidate %q; got %q", tc.wantName, names(got))
			}
			if c.Prec != tc.wantPrec {
				t.Errorf("%q Prec = %d, want %d", tc.wantName, c.Prec, tc.wantPrec)
			}
			if c.Source != SourceBody {
				t.Errorf("Source = %q, want %q", c.Source, SourceBody)
			}
		})
	}
}

// A head noun only counts when capitalised: lowercase "bandar" is the ordinary
// noun "city", and letting it through smuggles a blocklisted name past the
// filter as "Bandar Kuala Lumpur".
func TestPlacesInRejectsLowercaseHead(t *testing.T) {
	got := localeMS.placesIn("Beliau menyokong pemilihan di bandar Kuala Lumpur.", SourceBody)

	if c, ok := hasName(got, "Bandar Kuala Lumpur"); ok {
		t.Errorf("lowercase head produced ranked candidate %+v", c)
	}
	// The prep path halts too: "bandar" is lowercase, so needAlphaStart rejects
	// it as the first token of a bare run. Nothing is extracted at all, which is
	// the right outcome — "Kuala Lumpur" is blocklisted anyway.
	if len(got) != 0 {
		t.Errorf("expected no candidates; got %q", names(got))
	}
}

// A capitalised head still matches, so the rule does not over-reject.
func TestPlacesInAcceptsCapitalisedHead(t *testing.T) {
	got := localeMS.placesIn("Beliau tinggal di Bandar Baru Bangi.", SourceBody)

	c, ok := hasName(got, "Bandar Baru Bangi")
	if !ok {
		t.Fatalf("capitalised head did not match; got %q", names(got))
	}
	if c.Prec != PrecTown {
		t.Errorf("Prec = %d, want PrecTown", c.Prec)
	}
}

func TestPlacesInPrepPath(t *testing.T) {
	got := localeMS.placesIn("Lelaki itu ditahan di Kluang, seorang lagi bebas.", SourceBody)

	c, ok := hasName(got, "Kluang")
	if !ok {
		t.Fatalf("no candidate %q; got %q", "Kluang", names(got))
	}
	if c.Prec != PrecNone {
		t.Errorf("Prec = %d, want PrecNone (no head noun to rank it)", c.Prec)
	}
}

// A bare run must not cross a comma — there is no head noun to license it.
func TestPlacesInPrepPathStopsAtComma(t *testing.T) {
	got := localeMS.placesIn("Kejadian di Kluang, Seorang lelaki maut.", SourceBody)

	if _, ok := hasName(got, "Kluang, Seorang"); ok {
		t.Errorf("bare run crossed a comma; got %q", names(got))
	}
	if _, ok := hasName(got, "Kluang"); !ok {
		t.Errorf("expected %q; got %q", "Kluang", names(got))
	}
}

// A run must not stitch two sentences together.
func TestPlacesInDoesNotCrossSentences(t *testing.T) {
	got := localeMS.placesIn("Polis ke Taman. Melati ditahan.", SourceBody)

	for _, c := range got {
		if strings.Contains(c.Name, "Melati") {
			t.Errorf("run crossed a full stop: %+v", c)
		}
	}
}

func TestPlacesInNoPlace(t *testing.T) {
	const s = "Menurut Ketua Polis Timur Laut Abdul Rozak Muhammad, pengasuh direman."

	if got := localeMS.placesIn(s, SourceCaption); len(got) != 0 {
		t.Errorf("expected no candidates from a name-heavy sentence; got %q", names(got))
	}
}

// The Seksyen gate is wired through placesIn's emit.
func TestPlacesInSeksyenGate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool // want a Seksyen candidate
	}{
		{"statute citation", "Disiasat mengikut Seksyen 302 Kanun Keseksaan.", false},
		{"statute connective", "Didakwa di bawah Seksyen 39B Akta Dadah Berbahaya.", false},
		{"place with cue", "Kejadian di Seksyen 7 Shah Alam.", true},
		{"low number, no cue", "Beliau tinggal di Seksyen 9 bersama keluarga.", true},
		{"high number, no cue", "Perbicaraan Seksyen 376 diteruskan esok.", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := localeMS.placesIn(tc.s, SourceBody)

			var found bool
			for _, c := range got {
				if strings.HasPrefix(strings.ToLower(c.Name), "seksyen") {
					found = true
				}
			}
			if found != tc.want {
				t.Errorf("Seksyen candidate = %v, want %v (got %q)", found, tc.want, names(got))
			}
		})
	}
}

// localeAny has no Heads and no Preps, so it must degrade without panicking.
func TestPlacesInNilLocaleMaps(t *testing.T) {
	got := localeAny.placesIn("Kejadian di Taman Titiwangsa, Kluang.", SourceBody)
	if len(got) != 0 {
		t.Errorf("localeAny produced %q, want none (nil Heads and Preps)", names(got))
	}
}

// ExtractPlaces ──────────────────────────────────────────────────────────────

func TestExtractPlacesKeepsMostPrecise(t *testing.T) {
	src := PlaceSources{
		Caption: "Kejadian di Taman Titiwangsa, Kluang.",
		Body:    "Kejadian di Taman Titiwangsa, Kluang semalam.",
	}

	got := localeMS.ExtractPlaces(src)

	c, ok := hasName(got, "Taman Titiwangsa, Kluang")
	if !ok {
		t.Fatalf("missing ranked candidate; got %q", names(got))
	}
	if c.Prec != PrecStreet {
		t.Errorf("Prec = %d, want PrecStreet", c.Prec)
	}
	// The caption is added first, so it wins the tie on equal precision.
	if c.Source != SourceCaption {
		t.Errorf("Source = %q, want %q (caption is ranked first)", c.Source, SourceCaption)
	}

	// The same name must appear exactly once.
	var seen int
	for _, g := range got {
		if g.Name == "Taman Titiwangsa, Kluang" {
			seen++
		}
	}
	if seen != 1 {
		t.Errorf("name appears %d times, want 1", seen)
	}
}

// A later source with higher precision replaces an earlier low-precision hit.
func TestExtractPlacesUpgradesPrecision(t *testing.T) {
	src := PlaceSources{
		Title: "Lelaki maut di Stadium Darul Makmur",              // prep path → PrecNone
		Body:  "Kejadian berlaku di Stadium Darul Makmur semalam", // head path → PrecVenue
	}

	got := localeMS.ExtractPlaces(src)

	c, ok := hasName(got, "Stadium Darul Makmur")
	if !ok {
		t.Fatalf("missing candidate; got %q", names(got))
	}
	if c.Prec != PrecVenue {
		t.Errorf("Prec = %d, want PrecVenue (later, more precise reading must win)", c.Prec)
	}
}

func TestExtractPlacesBlocklist(t *testing.T) {
	src := PlaceSources{
		Dateline: "PETALING JAYA",
		Body:     "Polis di Bukit Aman mengesahkan kejadian di Kluang.",
	}

	got := localeMS.ExtractPlaces(src)

	for _, banned := range []string{"PETALING JAYA", "Bukit Aman"} {
		if c, ok := hasName(got, banned); ok {
			t.Errorf("blocklisted name survived: %+v", c)
		}
	}
	if _, ok := hasName(got, "Kluang"); !ok {
		t.Errorf("expected %q to survive; got %q", "Kluang", names(got))
	}
}

// The blocklist is matched on a normalised key, so casing and spacing in the
// extracted name must not let a banned name through.
func TestExtractPlacesBlocklistIsCaseInsensitive(t *testing.T) {
	got := localeMS.ExtractPlaces(PlaceSources{Dateline: "Petaling  Jaya"})
	if len(got) != 0 {
		t.Errorf("blocklist missed a differently-cased name: %q", names(got))
	}
}

func TestExtractPlacesDatelineSurvives(t *testing.T) {
	got := localeMS.ExtractPlaces(PlaceSources{Dateline: "GEORGE TOWN"})

	c, ok := hasName(got, "GEORGE TOWN")
	if !ok {
		t.Fatalf("dateline dropped; got %q", names(got))
	}
	if c.Prec != PrecNone {
		t.Errorf("Prec = %d, want PrecNone", c.Prec)
	}
	if c.Source != SourceDateline {
		t.Errorf("Source = %q, want %q", c.Source, SourceDateline)
	}
}

func TestExtractPlacesEmpty(t *testing.T) {
	if got := localeMS.ExtractPlaces(PlaceSources{}); len(got) != 0 {
		t.Errorf("empty sources produced %q", names(got))
	}
}

// End-to-end on the real shape a feed item arrives in.
func TestExtractPlacesEndToEnd(t *testing.T) {
	const html = `<figure><img src="x" alt="kluang">` +
		`<figcaption>Kejadian berlaku di rumah teres di Taman Titiwangsa, Kluang. (Gambar Bernama)</figcaption>` +
		`</figure>` +
		`<p>GEORGE TOWN: Seorang lelaki maut selepas rumahnya terbakar.</p>` +
		`<p>Mangsa dibawa ke Hospital Pulau Pinang untuk bedah siasat.</p>`

	got := localeMS.ExtractPlaces(SourcesFrom("Lelaki maut rumah terbakar di Kluang", html))

	want := map[string]Precision{
		"Taman Titiwangsa, Kluang": PrecStreet,
		"Hospital Pulau Pinang":    PrecVenue,
		"GEORGE TOWN":              PrecNone,
	}
	for name, prec := range want {
		c, ok := hasName(got, name)
		if !ok {
			t.Errorf("missing %q; got %q", name, names(got))
			continue
		}
		if c.Prec != prec {
			t.Errorf("%q Prec = %d, want %d", name, c.Prec, prec)
		}
	}
}

// Locale ─────────────────────────────────────────────────────────────────────

func TestCompileNormalisesKeys(t *testing.T) {
	l := Locale{
		Heads:     map[string]Precision{"Balai  Polis": PrecVenue, "TAMAN": PrecStreet},
		Blocklist: map[string]bool{"Bukit  AMAN": true},
	}.compile()

	if l.Heads["balai polis"] != PrecVenue {
		t.Errorf("Heads key not normalised: %v", l.Heads)
	}
	if l.Heads["taman"] != PrecStreet {
		t.Errorf("Heads key not lowercased: %v", l.Heads)
	}
	if !l.Blocklist["bukit aman"] {
		t.Errorf("Blocklist key not normalised: %v", l.Blocklist)
	}
}

func TestCompileDoesNotMutateInput(t *testing.T) {
	heads := map[string]Precision{"Balai  Polis": PrecVenue}
	block := map[string]bool{"Bukit  AMAN": true}

	Locale{Heads: heads, Blocklist: block}.compile()

	if _, ok := heads["Balai  Polis"]; !ok {
		t.Errorf("compile mutated the caller's Heads map: %v", heads)
	}
	if _, ok := block["Bukit  AMAN"]; !ok {
		t.Errorf("compile mutated the caller's Blocklist map: %v", block)
	}
}

func TestCompileMaxHeadTokens(t *testing.T) {
	l := Locale{Heads: map[string]Precision{
		"taman":            PrecStreet,
		"balai polis":      PrecVenue,
		"pusat  bandar  x": PrecTown,
	}}.compile()

	if l.maxHeadTokens != 3 {
		t.Errorf("maxHeadTokens = %d, want 3", l.maxHeadTokens)
	}
}

func TestHeadAtPrefersLongestMatch(t *testing.T) {
	const s = "Ditahan di Balai Polis Sentul"
	toks := tokenize(s)

	// "Balai" is token 2.
	prec, n, ok := localeMS.headAt(s, toks, 2)
	if !ok {
		t.Fatalf("headAt did not match at %q", toks[2].text(s))
	}
	if n != 2 {
		t.Errorf("head span = %d, want 2 (%q must beat a bare %q)", n, "balai polis", "balai")
	}
	if prec != PrecVenue {
		t.Errorf("Prec = %d, want PrecVenue", prec)
	}
}

func TestHeadAtToleratesExtraWhitespace(t *testing.T) {
	const s = "Ditahan di Balai  Polis Sentul"
	toks := tokenize(s)

	if _, n, ok := localeMS.headAt(s, toks, 2); !ok || n != 2 {
		t.Errorf("headAt(double space) = (n=%d, ok=%v), want (2, true)", n, ok)
	}
}

func TestHeadAtNilHeads(t *testing.T) {
	const s = "Taman Melati"
	if _, _, ok := localeAny.headAt(s, tokenize(s), 0); ok {
		t.Error("localeAny matched a head noun with a nil Heads map")
	}
}

func TestLocaleFor(t *testing.T) {
	tests := []struct {
		name     string
		lat, lng float64
		want     string
	}{
		{"Kuala Lumpur", 3.139, 101.687, "ms-MY"},
		{"Kuching, Sarawak", 1.553, 110.359, "ms-MY"},
		{"Hobart, Tasmania", -42.88, 147.32, "any"},
		{"London", 51.5, -0.12, "any"},
		{"null island", 0, 0, "any"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := LocaleFor(tc.lat, tc.lng).Code; got != tc.want {
				t.Errorf("LocaleFor(%v, %v) = %q, want %q", tc.lat, tc.lng, got, tc.want)
			}
		})
	}
}
