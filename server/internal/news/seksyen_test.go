package news

import (
	"regexp"
	"testing"
)

// seksyenFinder locates every "Seksyen N" in a string so a test can classify a
// chosen occurrence. Deliberately looser than seksyenRE — the tests must be
// able to point at a match without depending on the classifier's own pattern.
var seksyenFinder = regexp.MustCompile(`(?i)\bseksyen\s+\d+`)

func (k seksyenKind) String() string {
	switch k {
	case seksyenStatute:
		return "statute"
	case seksyenPlace:
		return "place"
	default:
		return "unknown"
	}
}

func TestClassifySeksyen(t *testing.T) {
	tests := []struct {
		name string
		text string
		// which occurrence to classify; -1 means the last one
		nth  int
		want seksyenKind
	}{
		// ---- statute, by act name following ----
		{
			name: "act name follows",
			text: "Kes disiasat mengikut Seksyen 39B Akta Dadah Berbahaya 1952",
			want: seksyenStatute,
		},
		{
			name: "penal code in Malay",
			text: "Suspek dihantar mengikut Seksyen 427 Kanun Keseksaan hari ini",
			want: seksyenStatute,
		},
		{
			name: "penal code in English",
			text: "He was charged under Seksyen 302 Penal Code yesterday",
			want: seksyenStatute,
		},

		// ---- statute, by the connective preceding ----
		{
			// The act name sits BEHIND this reference, not ahead of it, so the
			// after-window cue cannot see it.
			name: "chained reference, act name is behind",
			text: "Mereka didakwa mengikut Seksyen 149 Kanun Keseksaan yang dibaca bersama Seksyen 34",
			nth:  -1,
			want: seksyenStatute,
		},
		{
			// Regression guard: a place name inside the 60-char window must not
			// outrank a tight legal collocation. Getting this backwards
			// geocodes a statute citation to a Selangor suburb.
			name: "connective outranks an incidental place name",
			text: "Kejadian di Shah Alam. Didakwa dibaca bersama Seksyen 34 semalam",
			nth:  -1,
			want: seksyenStatute,
		},
		{
			name: "di bawah",
			text: "Lelaki itu ditahan di bawah Seksyen 15 kerana kesalahan dadah",
			want: seksyenStatute,
		},

		// ---- place ----
		{
			name: "township with the town named before",
			text: "Kejadian berlaku di Shah Alam Seksyen 7 malam tadi",
			want: seksyenPlace,
		},
		{
			name: "township with the town named after",
			text: "Rompakan di Seksyen 13, Petaling Jaya dilaporkan pagi ini",
			want: seksyenPlace,
		},
		{
			name: "state name is enough context",
			text: "Banjir kilat melanda Seksyen 24 di Selangor semalam",
			want: seksyenPlace,
		},

		// ---- fail closed ----
		{
			// A bare number with no act, no connective and no town is genuinely
			// ambiguous. A wrong pin is worse than no pin, so this must not
			// guess.
			name: "bare number with no context",
			text: "Seksyen 7 dilaporkan sunyi malam itu",
			want: seksyenUnknown,
		},
		{
			name: "number with no cues either side",
			text: "Menurutnya Seksyen 14 masih dalam siasatan lanjut",
			want: seksyenUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			locs := seksyenFinder.FindAllStringIndex(tc.text, -1)
			if len(locs) == 0 {
				t.Fatalf("test fixture has no %q occurrence: %q", "Seksyen N", tc.text)
			}

			idx := tc.nth
			if idx < 0 {
				idx = len(locs) + idx
			}
			if idx < 0 || idx >= len(locs) {
				t.Fatalf("nth=%d out of range (%d occurrences)", tc.nth, len(locs))
			}

			got := classifySeksyen(tc.text, locs[idx])
			if got != tc.want {
				t.Errorf("classifySeksyen() = %s, want %s\n  text: %q\n  span: %q",
					got, tc.want, tc.text, tc.text[locs[idx][0]:locs[idx][1]])
			}
		})
	}
}

// TestClassifySeksyenWindowIsLocal is the reason the classifier windows its
// context instead of scanning the whole string: crime articles almost always
// cite a statute somewhere, and a citation far away must not reclassify a
// neighbourhood mentioned at the start.
func TestClassifySeksyenWindowIsLocal(t *testing.T) {
	text := "Pergaduhan di Shah Alam Seksyen 7 malam tadi. " +
		"Polis memaklumkan siasatan masih dijalankan dan keterangan saksi sedang direkodkan " +
		"sebelum kes dirujuk. Mereka kemudian didakwa mengikut Kanun Keseksaan di mahkamah."

	locs := seksyenFinder.FindAllStringIndex(text, -1)
	if len(locs) != 1 {
		t.Fatalf("fixture should hold exactly one occurrence, got %d", len(locs))
	}

	if got := classifySeksyen(text, locs[0]); got != seksyenPlace {
		t.Errorf("classifySeksyen() = %s, want place — a distant statute cue leaked into the window", got)
	}
}

func TestClassifySeksyenBoundsAreSafe(t *testing.T) {
	// The windows slice around the match; a match at either end must not panic.
	tests := []struct {
		name string
		text string
	}{
		{"match at the very start", "Seksyen 7"},
		{"match at the very end", "Kejadian di Seksyen 7"},
		{"whole string is the match", "Seksyen 13"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			locs := seksyenFinder.FindAllStringIndex(tc.text, -1)
			if len(locs) == 0 {
				t.Fatalf("no match in %q", tc.text)
			}
			_ = classifySeksyen(tc.text, locs[0]) // must not panic
		})
	}
}
