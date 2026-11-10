package scan

import (
	"strings"
	"time"

	"byakugan/internal/store"
)

const (
	// clusterWindow: follow-ups (remand, charges) land within days of the event.
	clusterWindow = 10 * 24 * time.Hour
	// clusterReachM: one incident can be found through neighbouring targets
	// ("Jalan Sultan Ismail" and "Bukit Bintang"); the shared-words test is what
	// keeps different incidents on the same street apart.
	clusterReachM = 600.0
	// sharedWordsMin distinctive words in common makes two headlines one story;
	// within closeWindow, one is enough (same place, same few days, "tembak").
	sharedWordsMin = 2
	closeWindow    = 3 * 24 * time.Hour
	headlineWords  = 15
	// CorroboratedOutlets distinct outlets reporting it upgrades the status.
	CorroboratedOutlets = 3
)

// clusterStop are words every cop story shares; they say nothing about which story.
var clusterStop = map[string]bool{
	"polis": true, "pdrm": true, "ditahan": true, "tahan": true, "lelaki": true,
	"wanita": true, "suspek": true, "warga": true, "asing": true, "video": true,
	"tular": true, "viral": true, "buru": true, "cari": true, "siasat": true,
	"dakwa": true, "didakwa": true, "kuala": true, "lumpur": true, "selangor": true,
	"jalan": true, "taman": true, "dalam": true, "selepas": true, "kepada": true,
	"untuk": true, "pihak": true, "berkata": true, "kata": true, "seorang": true,
	"mgflash": true, "malaysia": true, "gazette": true, "malaysiagazette": true,
	"police": true, "after": true, "with": true, "from": true,
	// English filler and words every story shares, so English headlines don't
	// glue on them ("arrested" etc. fold onto "tahan" via the lexicon)
	"cops": true, "over": true, "into": true, "near": true, "says": true, "said": true,
	"were": true, "been": true, "have": true, "their": true, "they": true, "this": true,
	"that": true, "when": true, "while": true, "before": true, "amid": true, "following": true,
	"arrested": true, "nabbed": true, "detained": true, "held": true, "caught": true,
	"suspect": true, "suspects": true, "woman": true, "women": true, "foreign": true,
	"foreigner": true, "foreigners": true, "local": true, "charged": true,
	"lumpur's": true, "klang": true, "valley": true,
}

// Cluster groups finds that report the same event: dated within the window of
// each other, pinned within reach, and sharing distinctive headline words.
// Grouping is transitive (A~B and B~C puts all three together). Each group is
// returned as indexes into finds; singletons are groups too.
func Cluster(finds []store.Find) [][]int {
	n := len(finds)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}

	words := make([][]string, n)
	for i, f := range finds {
		words[i] = distinctWords(f.Title, f.PlaceName)
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if sameEvent(finds[i], finds[j], words[i], words[j]) {
				parent[find(i)] = find(j)
			}
		}
	}

	groups := map[int][]int{}
	var order []int
	for i := range finds {
		root := find(i)
		if _, seen := groups[root]; !seen {
			order = append(order, root)
		}
		groups[root] = append(groups[root], i)
	}
	out := make([][]int, 0, len(order))
	for _, root := range order {
		out = append(out, groups[root])
	}
	return out
}

func sameEvent(a, b store.Find, wa, wb []string) bool {
	if a.PublishedAt == nil || b.PublishedAt == nil {
		return false // can't place it in time; leave it alone rather than guess
	}
	gap := a.PublishedAt.Sub(*b.PublishedAt)
	if gap < 0 {
		gap = -gap
	}
	if gap > clusterWindow {
		return false
	}
	if distanceM(a.Lat, a.Lng, b.Lat, b.Lng) > clusterReachM {
		return false
	}
	need := sharedWordsMin
	if gap <= closeWindow {
		need = 1
	}
	shared := 0
	for _, x := range wa {
		for _, y := range wb {
			if wordsMatch(x, y) {
				if strings.HasPrefix(x, strongAnchorPrefix) {
					return true // same RM amount or Op codename, same place, same days
				}
				if shared++; shared >= need {
					return true
				}
				break
			}
		}
	}
	return false
}

// wordsMatch: equal, or one is a prefix of the other with the shorter at least
// 5 letters, so "tembak"/"tembakan" and "langgar"/"langgarnya" line up while
// "maut" only matches "maut".
func wordsMatch(a, b string) bool {
	if a == b {
		return true
	}
	if strings.HasPrefix(a, anchorPrefix) || strings.HasPrefix(b, anchorPrefix) {
		return false // anchors only match exactly
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	return len([]rune(a)) >= 5 && strings.HasPrefix(b, a)
}

// distinctWords: headline words of 4+ letters, minus stop words and the words
// of the place it was pinned to (every Jalan Sultan Ismail story shares those).
func distinctWords(title, place string) []string {
	placeWords := map[string]bool{}
	for _, w := range strings.Fields(normalizeTitle(place)) {
		placeWords[w] = true
	}
	// Facebook captions indexed as "titles" run on for paragraphs; only the
	// headline part (before the first hashtag, first headlineWords words) counts,
	// or one stray word in a long caption merges unrelated stories.
	if i := strings.Index(title, "#"); i > 0 {
		title = title[:i]
	}
	head := strings.Fields(title)
	if len(head) > headlineWords {
		head = head[:headlineWords]
	}
	fields := strings.Fields(normalizeTitle(strings.Join(head, " ")))
	if len(fields) > headlineWords {
		fields = fields[:headlineWords]
	}

	seen := map[string]bool{}
	var out []string
	for _, w := range fields {
		if placeWords[w] || isNumber(w) {
			continue
		}
		w = canonical(w) // English and affixed Malay → one Malay word (lexicon.go)
		if len([]rune(w)) < 4 || clusterStop[w] || placeWords[w] || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	for _, a := range anchors(strings.Join(head, " ")) {
		if !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	return out
}

// unprefix drops the Malay passive/accidental prefixes ("ditembak" → "tembak",
// "terbunuh" → "bunuh"); suffixes are handled by wordsMatch's prefix test.
func unprefix(w string) string {
	for _, p := range []string{"di", "ter"} {
		if rest, ok := strings.CutPrefix(w, p); ok && len([]rune(rest)) >= 4 {
			return rest
		}
	}
	return w
}

func isNumber(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// IncidentPlan is what one cluster should look like in the incidents table.
type IncidentPlan struct {
	Incident  store.Incident // ID set when an existing incident is reused
	MemberIDs []string
	Orphans   []string // incidents whose members all moved here (to delete if empty)
}

// PlanIncidents turns clusters into incident rows. A cluster reuses the
// incident most of its members already belong to, so ids stay stable across
// scans; any other incident its members came from becomes an orphan.
func PlanIncidents(finds []store.Find, groups [][]int) []IncidentPlan {
	plans := make([]IncidentPlan, 0, len(groups))
	for _, g := range groups {
		var (
			p       IncidentPlan
			votes   = map[string]int{}
			outlets = map[string]bool{}
			pin     = finds[g[0]]
			first   *time.Time
			last    *time.Time
			title   = finds[g[0]].Title
		)
		for _, i := range g {
			f := finds[i]
			p.MemberIDs = append(p.MemberIDs, f.ID)
			if f.IncidentID != nil {
				votes[*f.IncidentID]++
			}
			if f.Outlet != "" {
				outlets[strings.ToLower(f.Outlet)] = true
			}
			if f.Precision > pin.Precision {
				pin = f
			}
			if f.PublishedAt != nil {
				if first == nil || f.PublishedAt.Before(*first) {
					first, title = f.PublishedAt, f.Title // the earliest report names the event
				}
				if last == nil || f.PublishedAt.After(*last) {
					last = f.PublishedAt
				}
			}
		}

		best := ""
		for id, n := range votes {
			if best == "" || n > votes[best] || (n == votes[best] && id < best) {
				best = id
			}
		}
		for id := range votes {
			if id != best {
				p.Orphans = append(p.Orphans, id)
			}
		}

		status := "reported"
		if len(outlets) >= CorroboratedOutlets {
			status = "corroborated"
		}

		p.Incident = store.Incident{
			ID:             best,
			Title:          title,
			PlaceName:      pin.PlaceName,
			Lat:            pin.Lat,
			Lng:            pin.Lng,
			Precision:      pin.Precision,
			FirstPublished: first,
			LastPublished:  last,
			ArticleCount:   len(g),
			OutletCount:    len(outlets),
			Status:         status,
		}
		plans = append(plans, p)
	}
	return plans
}
