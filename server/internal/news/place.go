package news

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Precision int16

const (
	PrecNone Precision = iota
	PrecState
	PrecCity
	PrecTown
	PrecStreet
	PrecVenue
)

const (
	SourceCaption  = "caption"
	SourceTitle    = "title"
	SourceDateline = "dateline"
	SourceBody     = "body"
)

const maxNameTokens = 4
const maxPhraseTokens = 4

type PlaceCandidate struct {
	Name   string
	Prec   Precision
	Source string
}

type runOpts struct {
	limit          int
	stopOnComma    bool
	needAlphaStart bool
}

func isNameToken(s string, t token) bool {
	r, _ := utf8.DecodeRuneInString(t.text(s))
	return unicode.IsUpper(r) || unicode.IsDigit(r)
}

var figcaptionRE = regexp.MustCompile(`(?is)<figcaption[^>]*>(.*?)</figcaption>`)

var photoCreditRE = regexp.MustCompile(`\s*\((?i:Gambar|Foto|Photo|Pic|Image)[^)]*\)\s*$`)

var datelineRE = regexp.MustCompile(`^\s*([A-Z][A-Z\s.'-]{2,30}):\s*`)

type PlaceSources struct {
	Caption  string
	Title    string
	Dateline string
	Body     string
}

func SourcesFrom(title, contentHTML string) PlaceSources {

	body := stripHTML(figcaptionRE.ReplaceAllString(contentHTML, " "))
	dateline, rest := splitDateline(body)

	return PlaceSources{
		Caption:  ExtractCaption(contentHTML),
		Title:    title,
		Dateline: dateline,
		Body:     rest,
	}
}

func (l *Locale) placesIn(s, source string) []PlaceCandidate {
	toks := tokenize(s)
	var out []PlaceCandidate

	emit := func(from, to int, prec Precision, head string) {
		loc := []int{toks[from].start, toks[to].end}
		if l.Gate != nil && head != "" && !l.Gate(head, s, loc) {
			return
		}
		out = append(out, PlaceCandidate{
			Name:   phrase(s, toks, from, to),
			Prec:   prec,
			Source: source,
		})
	}

	for i := 0; i < len(toks); i++ {
		// headLength is 'Balai Polis' = 2
		// nameLength is 'Balai Polis Segambut' = 1 (Only looks at what's after head)
		prec, headLength, ok := l.headAt(s, toks, i)
		if ok && isNameToken(s, toks[i]) {
			head := normKey(s[toks[i].start:toks[i+headLength-1].end])

			if l.HeadPos != headFinal {
				if nameLength := runForward(s,
					toks,
					i+headLength,
					runOpts{limit: maxPhraseTokens - headLength}); nameLength > 0 {
					emit(i, i+headLength+nameLength-1, prec, head)
				}
			}
			if l.HeadPos != headInitial && i > 0 {
				if k := runBackward(
					s,
					toks,
					i-1); k > 0 {
					emit(i-k, i+headLength-1, prec, head)
				}
			}
			continue
		}

		if l.Preps[strings.ToLower(toks[i].text(s))] {
			if k := runForward(s, toks, i+1, runOpts{
				limit:          maxPhraseTokens,
				stopOnComma:    true,
				needAlphaStart: true}); k > 0 {
				emit(i+1, i+k, PrecNone, "")
			}
		}
	}

	return out
}

func phrase(s string, toks []token, from, to int) string {
	var b strings.Builder
	for j := from; j <= to; j++ {
		if j > from {
			if toks[j].commaBefore {
				b.WriteString(", ")
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(toks[j].text(s))
	}
	return b.String()
}

func ExtractCaption(contentHTML string) string {
	m := figcaptionRE.FindStringSubmatch(contentHTML)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(photoCreditRE.ReplaceAllString(stripHTML(m[1]), ""))
}

func (l *Locale) ExtractPlaces(src PlaceSources) []PlaceCandidate {
	var out []PlaceCandidate
	at := map[string]int{}

	add := func(cands ...PlaceCandidate) {
		for _, c := range cands {
			key := normKey(c.Name)
			if key == "" || l.Blocklist[key] {
				continue
			}
			i, seen := at[key]
			if !seen {
				at[key] = len(out)
				out = append(out, c)
				continue
			}
			if c.Prec > out[i].Prec {
				out[i] = c
			}
		}
	}

	add(l.placesIn(src.Caption, SourceCaption)...)
	add(l.placesIn(src.Title, SourceTitle)...)

	if src.Dateline != "" {
		add(PlaceCandidate{Name: src.Dateline, Prec: PrecNone, Source: SourceDateline})
	}
	add(l.placesIn(src.Body, SourceBody)...)

	return out
}

func splitDateline(body string) (dateline, rest string) {
	m := datelineRE.FindStringSubmatchIndex(body)
	if m == nil {
		return "", body
	}
	return normSpace(body[m[2]:m[3]]), body[m[1]:]
}

func runForward(s string, toks []token, from int, o runOpts) int {
	n, truncated := 0, false
	for j := from; j < len(toks); j++ {
		if !isNameToken(s, toks[j]) || toks[j].breakBefore {
			break
		}
		if o.stopOnComma && toks[j].commaBefore {
			break
		}
		if n == o.limit {
			truncated = true
			break
		}
		if n == 0 && o.needAlphaStart {
			r, _ := utf8.DecodeRuneInString(toks[j].text(s))
			if !unicode.IsUpper(r) {
				break
			}
		}

		n++
	}

	if truncated {
		for k := n - 1; k > 0; k-- {
			if toks[from+k].commaBefore {
				return k
			}
		}
	}

	return n
}

func runBackward(s string, toks []token, to int) int {
	n := 0
	for j := to; j >= 0 && n < maxNameTokens; j-- {
		if !isNameToken(s, toks[j]) {
			break
		}
		n++

		if toks[j].breakBefore {
			break
		}
	}
	return n
}

type token struct {
	start, end  int
	breakBefore bool
	commaBefore bool
}

func isSoftSep(r rune) bool { return r == ',' }

func isSep(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune(".;:!?()\"\u2014\u2013", r)
}

func isBreakRune(r rune) bool {
	return strings.ContainsRune(".;:!?()\n", r)
}

func tokenize(s string) []token {
	var out []token
	var cur token
	in, brk, comma := false, false, false

	for i, r := range s {
		switch {
		case isSep(r) || isSoftSep(r):
			if in {
				cur.end = i
				out = append(out, cur)
				in = false
			}
			if isBreakRune(r) {
				brk = true
			}
			if isSoftSep(r) {
				comma = true
			}
		case !in:
			cur = token{start: i, breakBefore: brk, commaBefore: comma}
			in, brk, comma = true, false, false
		}
	}
	if in {
		cur.end = len(s)
		out = append(out, cur)
	}
	return out
}

func (t token) text(s string) string { return s[t.start:t.end] }
