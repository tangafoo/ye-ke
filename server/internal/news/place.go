package news

import (
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
	limit       int
	stopOnComma bool
}

func isNameToken(s string, t token) bool {
	r, _ := utf8.DecodeRuneInString(t.text(s))
	return unicode.IsUpper(r) || unicode.IsDigit(r)
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
	return strings.ContainsRune(".;:!?\n", r)
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
