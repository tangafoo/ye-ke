package news

import (
	"regexp"
	"strings"
)

type seksyenKind int

const (
	seksyenUnknown seksyenKind = iota
	seksyenStatute
	seksyenPlace
)

var statuteCues = []string{
	"kanun keseksaan",
	"akta ",
	"apj ",
	"kanun ",
	"seksyen kecil", "penal code", "act ",
}

var placeCues = []string{
	"shah alam",
	"petaling jaya",
	"pj ",
	"mont kiara",
	"segambut",
	"subang",
	"selangor",
	"bangi",
	"klang",
	"johor", "cheras", "yulek", "ampang", "desa park city", "kepong", "mantin", "seremban", "ttdi", "choo cheng khay",
}

var statuteConnectives = []string{
	"mengikut ", "di bawah ", "dibaca bersama ", "bawah ",
}

var seksyenRE = regexp.MustCompile(`(?i)\bseksyen\s+(\d+)\s*(?:\(\d+\))?\s*\(?([A-Za-z])?\)?`)

func classifySeksyen(text string, loc []int) seksyenKind {
	lower := strings.ToLower(text)

	after := lower[min(loc[1], len(lower)):min(loc[1]+60, len(lower))]
	before := lower[max(0, loc[0]-60):loc[0]]

	for _, cue := range statuteCues {
		if strings.Contains(after, cue) {
			return seksyenStatute
		}
	}
	for _, cue := range statuteConnectives {
		if strings.Contains(before, cue) {
			return seksyenStatute
		}
	}
	for _, cue := range placeCues {
		if strings.Contains(before, cue) || strings.Contains(after, cue) {
			return seksyenPlace
		}
	}

	if sub := seksyenRE.FindStringSubmatch(text[loc[0]:loc[1]]); sub != nil && sub[2] != "" {
		return seksyenStatute
	}

	return seksyenUnknown
}
