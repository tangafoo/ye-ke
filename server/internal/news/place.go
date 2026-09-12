package news

import (
	"strings"
	"unicode"
)

const (
	PrecNone   int16 = 0
	PrecState  int16 = 1
	PrecCity   int16 = 2
	PrecTown   int16 = 3
	PrecStreet int16 = 4
	PrecVenue  int16 = 5
)

type token struct {
	start, end  int
	breakBefore bool
}

func isSep(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune(",.;:!?()\"\u2014\u2013", r)
}

func isBreakRune(r rune) bool {
	return strings.ContainsRune(".;:!?\n", r)
}

func tokenize(s string) []token {
	var out []token
	var cur token
	in, brk := false, false

	for i, r := range s {
		switch {
		case isSep(r):
			if in {
				cur.end = i
				out = append(out, cur)
				in = false
			}
			if isBreakRune(r) {
				brk = true
			}
		case !in:
			cur = token{start: i, breakBefore: brk}
			in, brk = true, false
		}
	}
	if in {
		cur.end = len(s)
		out = append(out, cur)
	}
	return out
}

func (t token) text(s string) string { return s[t.start:t.end] }
