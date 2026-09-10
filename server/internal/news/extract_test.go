package news

import (
	"testing"
)

func TestExtractOpCode(t *testing.T) {
	tests := []struct {
		name  string
		texts []string
		want  []OpMention
	}{
		{
			name:  "plain headline",
			texts: []string{"Polis tumpaskan sindiket menerusi Op Eumenes di Puncak Alam"},
			want:  []OpMention{{Key: "eumenes", Display: "Op Eumenes", Mentions: 1}},
		},
		{
			// PDRM publishes headlines in ALL CAPS. A case-sensitive prefix
			// would silently skip our highest-quality source.
			name:  "all-caps headline still matches",
			texts: []string{"#WarAgainstDrugs OP EUMENES: MAKMAL DADAH RM2.4 BILION TUMPAS"},
			want:  []OpMention{{Key: "eumenes", Display: "OP EUMENES", Mentions: 1}},
		},
		{
			name:  "Ops variant",
			texts: []string{"Ops Selamat 2026 bermula esok"},
			want:  []OpMention{{Key: "selamat", Display: "Ops Selamat", Mentions: 1}},
		},
		{
			name:  "abbreviated with a full stop",
			texts: []string{"Menerusi Ops. Tapis, 12 individu ditahan"},
			want:  []OpMention{{Key: "tapis", Display: "Ops. Tapis", Mentions: 1}},
		},
		{
			// A stopword must skip that MATCH, not abandon the whole string —
			// "Op Khas" is "special operation", but Eumenes is right behind it.
			name:  "stopword does not swallow a real codename",
			texts: []string{"Op Khas dijalankan bersama Op Eumenes di Selangor"},
			want:  []OpMention{{Key: "eumenes", Display: "Op Eumenes", Mentions: 1}},
		},
		{
			name:  "two operations both returned",
			texts: []string{"Op Tapis dan Op Eumenes dilancarkan serentak di Selangor"},
			want: []OpMention{
				{Key: "tapis", Display: "Op Tapis", Mentions: 1},
				{Key: "eumenes", Display: "Op Eumenes", Mentions: 1},
			},
		},
		{
			name: "mentions accumulate across texts",
			texts: []string{
				"Op Tapis dan Op Eumenes dilancarkan serentak",
				"Menerusi Op Eumenes, makmal diserbu. Op Eumenes melibatkan enam serbuan.",
			},
			want: []OpMention{
				{Key: "tapis", Display: "Op Tapis", Mentions: 1},
				{Key: "eumenes", Display: "Op Eumenes", Mentions: 3},
			},
		},
		{
			// operations holds one row per code, so the display must not depend
			// on which article happened to arrive first.
			name: "mixed-case spelling wins over all-caps",
			texts: []string{
				"OP EUMENES: MAKMAL TUMPAS",
				"Menerusi Op Eumenes, polis menyerbu premis di Puncak Alam.",
			},
			want: []OpMention{{Key: "eumenes", Display: "Op Eumenes", Mentions: 2}},
		},
		{
			name:  "no operation in an ordinary crime headline",
			texts: []string{"Polis tahan lima lelaki di Kampung Baru"},
			want:  nil,
		},
		{
			// The capitalisation of the NAME is the signal that separates a
			// proper noun from ordinary Malay prose. Lowercasing the input
			// first would invent "Op Berjaya" here.
			name:  "lowercase prose is not an operation",
			texts: []string{"beliau berkata op berjaya menahan lima suspek"},
			want:  nil,
		},
		{
			name:  "generic descriptor alone yields nothing",
			texts: []string{"Op Khas dijalankan di seluruh negeri"},
			want:  nil,
		},
		{
			// In ALL CAPS the proper-noun signal is gone, so the stopword list
			// is the only defence left.
			name:  "all-caps function word is stopped",
			texts: []string{"PDRM SAHKAN OP TERSEBUT MASIH DIJALANKAN"},
			want:  nil,
		},
		{
			name:  "no texts at all",
			texts: nil,
			want:  nil,
		},
		{
			name:  "whitespace inside the match is normalised",
			texts: []string{"Menerusi Op   Eumenes\n polis menyerbu"},
			want:  []OpMention{{Key: "eumenes", Display: "Op Eumenes", Mentions: 1}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractOpCode(tc.texts...)

			if len(got) != len(tc.want) {
				t.Fatalf("got %d mentions %+v, want %d %+v", len(got), got, len(tc.want), tc.want)
			}
			// Order is first-appearance, so compare positionally.
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("mention[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestNormSpace(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Op Eumenes", "Op Eumenes"},
		{"Op   Eumenes", "Op Eumenes"},
		{"Op\nEumenes", "Op Eumenes"},
		{"  Op\t Eumenes  ", "Op Eumenes"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := normSpace(tc.in); got != tc.want {
			t.Errorf("normSpace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsAllCaps(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"OP EUMENES", true},
		{"Op Eumenes", false},
		{"op eumenes", false},
		{"OPS. TAPIS", true},
		{"", true}, // vacuously — callers only compare it against a real match
	}
	for _, tc := range tests {
		if got := isAllCaps(tc.in); got != tc.want {
			t.Errorf("isAllCaps(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
