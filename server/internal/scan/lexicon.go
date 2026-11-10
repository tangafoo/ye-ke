package scan

import (
	"regexp"
	"strings"
)

// The Malay–English glue. Malaysian cop stories run in both languages: "Empat
// penjenayah ditembak mati di Jalan Imbi" (Utusan) and "Four robbers shot dead
// in Jalan Imbi" (The Star) are one incident. Clustering compares headline
// words, so both languages are folded onto one canonical (Malay) word first.
//
// lexicon maps a canonical word to every form that should count as it:
// English words and inflections, and Malay affixed forms that unprefix() can't
// reach (meN- changes the root's first letter: tembak → menembak).
var lexicon = map[string][]string{
	// violence
	"tembak":  {"shoot", "shoots", "shot", "shooting", "shootings", "shootout", "gunned", "gunshot", "gunfire", "menembak", "tembakan", "penembak"},
	"pistol":  {"gun", "guns", "firearm", "firearms", "handgun", "revolver"},
	"tikam":   {"stab", "stabs", "stabbed", "stabbing", "menikam", "tikaman"},
	"parang":  {"machete", "machetes"},
	"pisau":   {"knife", "knives"},
	"bunuh":   {"murder", "murders", "murdered", "murderer", "killing", "slain", "homicide", "membunuh", "pembunuhan", "pembunuh"},
	"maut":    {"kill", "kills", "killed", "dead", "death", "deaths", "died", "dies", "fatal", "mati", "meninggal", "terkorban"},
	"serang":  {"attack", "attacked", "assault", "assaulted", "beat", "beaten", "beating", "menyerang", "serangan", "belasah", "pukul", "memukul"},
	"gaduh":   {"fight", "fights", "fighting", "brawl", "brawls", "scuffle", "clash", "bergaduh", "pergaduhan", "kecoh", "rusuhan", "riot"},
	"culik":   {"kidnap", "kidnapped", "kidnapping", "abduct", "abducted", "abduction", "menculik", "penculikan"},
	"rogol":   {"rape", "raped", "rapist", "merogol"},
	"cabul":   {"molest", "molested", "molestation", "mencabul"},
	"letupan": {"explosion", "explosions", "blast", "blasts", "bomb", "bombing", "meletup", "bom"},
	"suar":    {"flare", "flares"},

	// property crime
	"samun":    {"rob", "robs", "robbed", "robbing", "robber", "robbers", "robbery", "robberies", "heist", "menyamun", "samunan", "penyamun", "rompak", "merompak", "rompakan", "perompak"},
	"ragut":    {"snatch", "snatched", "snatcher", "snatchers", "meragut", "peragut"},
	"curi":     {"theft", "thefts", "steal", "steals", "stole", "stolen", "stealing", "thief", "thieves", "mencuri", "pencuri", "kecurian", "curian"},
	"pecah":    {"burglary", "burglar", "burglars", "break-in", "breakin", "pecahrumah"},
	"penipuan": {"scam", "scams", "scammed", "scammer", "scammers", "fraud", "tipu", "menipu", "penipu"},

	// vehicles and the road
	"langgar":   {"crash", "crashes", "crashed", "collision", "collided", "ram", "rammed", "ramming", "rempuh", "merempuh", "melanggar", "nahas", "kemalangan", "accident"},
	"kejar":     {"chase", "chased", "chasing", "pursuit", "pursued", "mengejar", "kejarmengejar"},
	"lori":      {"lorry", "lorries", "truck", "trucks"},
	"kereta":    {"car", "cars", "vehicle"},
	"motosikal": {"motorcycle", "motorcycles", "motorbike", "motorbikes", "motorcyclist", "motorcyclists", "penunggang", "rider", "riders"},
	"pemandu":   {"driver", "drivers", "memandu", "driving"},
	"terbabas":  {"skid", "skidded", "overturned", "terbalik"},

	// police action and the aftermath
	"serbu":     {"raid", "raids", "raided", "menyerbu", "serbuan", "gempur", "menggempur", "ops"},
	"reman":     {"remand", "remanded"},
	"mayat":     {"body", "bodies", "corpse", "remains"},
	"mangsa":    {"victim", "victims"},
	"saksi":     {"witness", "witnesses"},
	"bakar":     {"fire", "blaze", "burnt", "burned", "burning", "arson", "terbakar", "kebakaran", "membakar"},
	"geng":      {"gang", "gangs", "gangster", "gangsters", "triad", "samseng"},
	"dadah":     {"drug", "drugs", "narcotics", "syabu", "meth", "methamphetamine", "heroin", "ketamine", "ganja", "cannabis"},
	"judi":      {"gambling", "gamble", "casino", "berjudi", "perjudian"},
	"pelacuran": {"vice", "prostitution", "prostitute", "prostitutes", "pelacur", "seks", "sex"},
	"protes":    {"protest", "protests", "protester", "protesters", "rally", "demonstration", "himpunan", "berhimpun", "perhimpunan", "bantahan"},

	// people and places
	"imigresen":  {"immigration", "jim"},
	"pati":       {"undocumented", "illegals"},
	"remaja":     {"teen", "teens", "teenager", "teenagers", "teenage"},
	"pelajar":    {"student", "students"},
	"sekeluarga": {"family", "families", "keluarga", "beranak"},
	"restoran":   {"restaurant", "restaurants", "eatery", "kedaimakan"},
	"emas":       {"gold", "goldsmith", "jewellery", "jewelry", "jeweller"},
	"kedai":      {"shop", "shops", "store", "stores", "premis", "premises"},
	"hospital":   {"hkl"},
	"mahkamah":   {"court", "courts", "magistrate", "sessions"},
	"juta":       {"million", "millions"},

	// numbers that headlines spell out
	"dua":   {"two"},
	"tiga":  {"three"},
	"empat": {"four"},
	"lima":  {"five"},
	"enam":  {"six"},
}

// canonWord is lexicon inverted: form → canonical word.
var canonWord = func() map[string]string {
	m := map[string]string{}
	for canon, forms := range lexicon {
		m[canon] = canon
		for _, f := range forms {
			m[normalizeTitle(f)] = canon
		}
	}
	return m
}()

// canonical folds one normalised headline word onto its canonical form.
func canonical(w string) string {
	if c, ok := canonWord[w]; ok {
		return c
	}
	if u := unprefix(w); u != w {
		if c, ok := canonWord[u]; ok {
			return c
		}
		return u
	}
	return w
}

// Anchors are tokens that read the same in any language: money, ages, counts,
// operation codenames. They're compared exactly, and a "strong" anchor (money
// or an Op codename) is enough on its own to call two headlines one story,
// within the cluster's time and distance limits.
const (
	anchorPrefix       = "@"
	strongAnchorPrefix = "@!"
)

var (
	// RM9.7 juta / RM9.7 million / RM250,000 / RM15k
	moneyRE = regexp.MustCompile(`(?i)\bRM\s?(\d+(?:[.,]\d+)*)\s*(juta|million|mil|bilion|billion|ribu|k)?\b`)
	// 14 tahun / 14-year-old / 14 years old
	ageRE = regexp.MustCompile(`(?i)\b(\d{1,2})[\s-]*(?:tahun|year[s]?[\s-]*old)\b`)
	// Op Tapis Khas / Ops Selamat / Operasi Noda
	opRE = regexp.MustCompile(`\bOp(?:s|erasi)?\s+([A-Z][a-z]+)`)
	// counts of 2+ digits ("417 ditahan", "1,101 PATI"), not years
	countRE = regexp.MustCompile(`\b(\d{1,3}(?:,\d{3})+|\d{2,})\b`)
)

func anchors(title string) []string {
	var out []string

	for _, m := range moneyRE.FindAllStringSubmatch(title, -1) {
		amount := strings.ReplaceAll(m[1], ",", "")
		switch strings.ToLower(m[2]) {
		case "juta", "million", "mil":
			amount += "m"
		case "bilion", "billion":
			amount += "b"
		case "ribu", "k":
			amount += "k"
		}
		out = append(out, strongAnchorPrefix+"rm"+amount)
	}
	for _, m := range opRE.FindAllStringSubmatch(title, -1) {
		out = append(out, strongAnchorPrefix+"op"+strings.ToLower(m[1]))
	}
	ages := map[string]bool{}
	for _, m := range ageRE.FindAllStringSubmatch(title, -1) {
		ages[m[1]] = true
		out = append(out, anchorPrefix+"age"+m[1])
	}

	// counts: skip what was already read as money or an age, and years
	stripped := moneyRE.ReplaceAllString(title, " ")
	for _, m := range countRE.FindAllString(stripped, -1) {
		n := strings.ReplaceAll(m, ",", "")
		if ages[n] || (len(n) == 4 && (strings.HasPrefix(n, "19") || strings.HasPrefix(n, "20"))) {
			continue
		}
		out = append(out, anchorPrefix+"n"+n)
	}
	return out
}
