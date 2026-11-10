package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"byakugan/internal/news"
	"byakugan/internal/osm"
	"byakugan/internal/store"
)

const newsConcurrency = 4

// NewsLane searches the news for each target as an exact phrase, qualified by
// the state ("Jalan Petaling" "Kuala Lumpur") and an OR group of crime words. Google matches the quoted name against the whole article, so a hit
// means the story names that place. Headlines must also carry a crime word.
//
// A story found through several targets is kept once, pinned at its most
// precise target. ok is false when every search failed (upstream down), so the
// caller can skip recording the sweep and let the next scan retry.
func NewsLane(ctx context.Context, nc *news.Client, r Region, targets []Target, areas []osm.Area) (finds []store.Find, ok bool) {
	quals := qualifierStates(areas)
	if r.QualifyByCity {
		quals = qualifierCities(areas)
	}
	local := localNames(areas, targets)

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		sem       = make(chan struct{}, newsConcurrency)
		best      = map[string]store.Find{}
		succeeded int
	)

	type job struct {
		t    Target
		lang Lang
	}
	var jobs []job
	for _, t := range targets {
		for _, l := range r.Langs {
			jobs = append(jobs, job{t, l})
		}
	}

	for _, jb := range jobs {
		wg.Add(1)
		go func(t Target, l Lang) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			qual := quals
			if !r.QualifyByCity && len(strings.Fields(t.Name)) >= 3 {
				qual = nil // "Jalan Taman Setiawangsa" is specific enough on its own
			}
			q := buildQuery(t.Name, qual, l.QueryTerms, l.Exclude)
			articles, err := nc.SearchIn(ctx, q, l.Edition)
			if err != nil {
				log.Printf("[scan] news %q: %v", t.Name, err)
				return
			}
			mu.Lock()
			succeeded++
			mu.Unlock()

			kept := 0
			for _, a := range articles {
				title := strings.ToLower(a.Title)
				if !hasAny(title, l.TitleTerms) || hasAny(title, l.RejectTerms) {
					continue
				}
				if !pinnable(a.Title, t) || (r.QualifyByCity && elsewhere(a.Title, local)) {
					continue
				}
				f := toFind(a, t, q)

				mu.Lock()
				if prev, seen := best[f.SourceID]; !seen || f.Precision > prev.Precision {
					best[f.SourceID] = f
				}
				mu.Unlock()
				kept++
			}
			log.Printf("[scan] news %s %q: %d results, %d crime", l.Edition.HL, t.Name, len(articles), kept)
		}(jb.t, jb.lang)
	}
	wg.Wait()

	finds = make([]store.Find, 0, len(best))
	for _, f := range best {
		finds = append(finds, f)
	}
	return finds, succeeded > 0 || len(jobs) == 0
}

// qualifierCities: the most local city-level areas (admin_level 6–8, at most
// two), named the way news names them ("City of Brisbane" → "Brisbane").
func qualifierCities(areas []osm.Area) []string {
	var out []string
	for _, a := range areas { // most local first
		if a.Level < 6 || a.Level > 8 {
			continue
		}
		out = append(out, cityName(a.Name))
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 0 {
		return qualifierStates(areas)
	}
	return out
}

func cityName(s string) string {
	for _, p := range []string{"City of ", "Town of ", "Shire of ", "Stadt ", "Ville de "} {
		s = strings.TrimPrefix(s, p)
	}
	return strings.TrimSpace(s)
}

// localNames are the area and neighbourhood names that count as "here".
func localNames(areas []osm.Area, targets []Target) []string {
	var out []string
	for _, a := range areas {
		out = append(out, strings.ToLower(cityName(a.Name)))
	}
	for _, t := range targets {
		if t.Kind == "area" {
			out = append(out, strings.ToLower(t.Name))
		}
	}
	return out
}

// policeSuburbRE: police media headlines end with the suburb
// ("Armed robbery, Broadbeach", "Stolen vehicle, Brisbane City").
var policeSuburbRE = regexp.MustCompile(`,\s*([A-Z][A-Za-z'\- ]{2,40})$`)

// elsewhere reports a headline that names its suburb at the end, when that
// suburb isn't one of the local names. "Queen Street" exists in every
// Queensland town; "Armed robbery, Goodna" is about Goodna's.
func elsewhere(title string, local []string) bool {
	m := policeSuburbRE.FindStringSubmatch(strings.TrimSpace(title))
	if m == nil || len(local) == 0 {
		return false
	}
	suburb := strings.ToLower(m[1])
	if len(strings.Fields(suburb)) > 3 {
		return false // a clause, not a suburb
	}
	for _, l := range local {
		if strings.Contains(suburb, l) || strings.Contains(l, suburb) {
			return false
		}
	}
	return true
}

// klangValley: the KL / Selangor / Putrajaya borders run through neighbourhoods
// (Setiawangsa's Caltex sits in Selangor by OSM, in "Kuala Lumpur" by every
// news report), so a point in any of them is qualified by both names.
var klangValley = map[string]bool{"Kuala Lumpur": true, "Selangor": true, "Putrajaya": true}

// qualifierStates keeps only state-level areas. Councils and districts
// ("Majlis Perbandaran Ampang Jaya") are names news never uses.
func qualifierStates(areas []osm.Area) []string {
	var out []string
	for _, a := range areas {
		if a.Level != 4 {
			continue
		}
		if klangValley[a.Name] {
			return []string{"Kuala Lumpur", "Selangor"}
		}
		out = append(out, a.Name)
	}
	return out
}

// "Jalan Petaling" ("Kuala Lumpur" OR "Selangor") (polis OR tular OR …) -"Balai Polis Trafik"
func buildQuery(name string, areas, terms, exclude []string) string {
	phrase := func(s string) string { return `"` + strings.ReplaceAll(s, `"`, "") + `"` }

	var b strings.Builder
	b.WriteString(phrase(name))

	switch len(areas) {
	case 0:
	case 1:
		b.WriteString(" " + phrase(areas[0]))
	default:
		quoted := make([]string, len(areas))
		for i, a := range areas {
			quoted[i] = phrase(a)
		}
		fmt.Fprintf(&b, " (%s)", strings.Join(quoted, " OR "))
	}

	if len(terms) > 0 {
		fmt.Fprintf(&b, " (%s)", strings.Join(terms, " OR "))
	}
	for _, x := range exclude {
		b.WriteString(" -" + phrase(x))
	}
	return b.String()
}

func toFind(a news.Article, t Target, query string) store.Find {
	raw, _ := json.Marshal(map[string]any{"article": a, "query": query, "target": t.Name, "kind": t.Kind})

	f := store.Find{
		SourceKind: "news",
		SourceID:   dedupKey(a),
		Lat:        t.Lat,
		Lng:        t.Lng,
		Precision:  int16(t.Precision),
		PlaceName:  t.Name,
		Title:      a.Title,
		Body:       a.Snippet,
		URL:        a.Link,
		Outlet:     a.Outlet,
		Raw:        raw,
	}
	if !a.Published.IsZero() {
		p := a.Published
		f.PublishedAt = &p
	}
	return f
}

// dedupKey identifies one story across queries and lanes. Google's CBMi links
// differ per query for the same article, so the key is built from what doesn't
// change: outlet host + normalised headline + publish date.
func dedupKey(a news.Article) string {
	host := a.Outlet
	if u, err := url.Parse(a.OutletURL); err == nil && u.Host != "" {
		host = strings.TrimPrefix(u.Host, "www.")
	}
	date := ""
	if !a.Published.IsZero() {
		date = a.Published.UTC().Format("2006-01-02")
	}
	sum := sha256.Sum256([]byte(host + "|" + normalizeTitle(a.Title) + "|" + date))
	return hex.EncodeToString(sum[:])
}

func normalizeTitle(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// longPostRunes: Google News also indexes outlets' Facebook posts, whose "title"
// is the whole caption. Those mention several places, so a long one has to name
// the target in its own text to be pinned there.
const longPostRunes = 120

// pinnable reports whether a headline may be pinned at target t.
//   - A long social-post caption must name the target itself (a Kuala Selangor
//     crash caption matched "Jalan Alor" somewhere in its hashtags).
//   - A headline naming a different street on a street target is about that
//     street ("Operasi … di Jalan Raja Laut" found via "Jalan Raja Chulan").
func pinnable(title string, t Target) bool {
	norm := " " + normalizeName(title) + " "
	name := normalizeName(t.Name)
	if strings.Contains(norm, " "+name+" ") {
		return true
	}
	if len([]rune(title)) > longPostRunes {
		return false
	}
	if t.Kind != "street" {
		return true
	}
	return !namesOtherStreet(norm, name)
}

// streetHeads start a street name in Malay headlines.
var streetHeads = map[string]bool{"jalan": true, "jln": true, "lorong": true, "lebuh": true, "persiaran": true}

// "jalan raya" (road, as in kemalangan jalan raya) and friends aren't street names
var genericRoadWords = map[string]bool{"raya": true, "utama": true, "tol": true, "pintas": true, "belakang": true, "sempit": true, "kampung": true}

// namesOtherStreet: the headline names a street whose words diverge from the
// target's ("jalan raja laut" vs "jalan raja chulan"). A headline that stops
// early ("di Jalan Raja") doesn't count as a conflict.
func namesOtherStreet(norm, target string) bool {
	tw := strings.Fields(target)
	words := strings.Fields(norm)
	for i, w := range words {
		if !streetHeads[w] || (i+1 < len(words) && genericRoadWords[words[i+1]]) {
			continue
		}
		for k := 1; k < len(tw) && i+k < len(words); k++ {
			if words[i+k] != tw[k] {
				return true
			}
		}
	}
	return false
}

// normalizeName lowercases and drops punctuation, so "Jalan P. Ramlee" and
// "Jalan P Ramlee" compare equal.
func normalizeName(s string) string { return normalizeTitle(s) }

func hasAny(s string, terms []string) bool {
	for _, t := range terms {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
