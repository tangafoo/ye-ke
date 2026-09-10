package news

import (
	"regexp"
	"strings"
)

var opRE = regexp.MustCompile(`\b(?i:Ops?)\.?\s+([A-Z][\p{L}]{2,})`)

var opStopwords = map[string]bool{
	"khas":       true,
	"bersepadu":  true,
	"gabungan":   true,
	"besar":      true,
	"bersama":    true,
	"susulan":    true,
	"ini":        true,
	"itu":        true,
	"yang":       true,
	"tersebut":   true,
	"berkenaan":  true,
	"sebenarnya": true,
	"berikutan":  true,
}

type OpMention struct {
	Key      string
	Display  string
	Mentions int
}

func ExtractOpCode(texts ...string) []OpMention {
	var out []OpMention
	at := map[string]int{}

	for _, t := range texts {
		for _, m := range opRE.FindAllStringSubmatchIndex(t, -1) {
			raw, name := t[m[0]:m[1]], t[m[2]:m[3]]
			key := strings.ToLower(name)

			if opStopwords[key] {
				continue
			}

			i, seen := at[key]
			if !seen {
				at[key] = len(out)
				out = append(out, OpMention{Key: key, Display: normSpace(raw)})
				i = len(out) - 1
			}
			if isAllCaps(out[i].Display) && !isAllCaps(raw) {
				out[i].Display = normSpace(raw)
			}
			out[i].Mentions++
		}
	}
	return out
}

func normSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

func isAllCaps(s string) bool { return s == strings.ToUpper(s) }
