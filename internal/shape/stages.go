package shape

import (
	"cmp"
	"slices"
	"unicode"
)

//nolint:gochecknoglobals // tags read once
var (
	latn = MakeTag("latn")
	dflt = MakeTag("DFLT")
	kern = MakeTag("kern")
	mymr = MakeTag("mymr")
	thai = MakeTag("thai")
	lao  = MakeTag("lao ")
	hang = MakeTag("hang")
)

// shaper is the HarfBuzz shaper a run is shaped with.
type shaper uint8

const (
	defaultShaper shaper = iota
	arabicShaper
	indicShaper
	khmerShaper
	myanmarShaper
	useShaper
)

// script is a script Script tells apart: its characters, its OpenType
// tag, the newer tag of an Indic script, and the shaper HarfBuzz gives
// its runs.
type script struct {
	table  *unicode.RangeTable
	tag    Tag
	newer  Tag
	shaper shaper
}

// scripts are the scripts Script tells apart, by their OpenType tags.
//
//nolint:gochecknoglobals // a fixed table
var scripts = func() []script {
	out := []script{
		{table: unicode.Latin, tag: latn}, {table: unicode.Greek, tag: MakeTag("grek")},
		{table: unicode.Cyrillic, tag: MakeTag("cyrl")},
		{table: unicode.Hebrew, tag: MakeTag("hebr")}, {table: unicode.Thaana, tag: MakeTag("thaa")},
		{table: unicode.Samaritan, tag: MakeTag("samr")},
		{table: unicode.Thai, tag: thai}, {table: unicode.Lao, tag: lao}, {table: unicode.Hangul, tag: hang},
	}
	for _, s := range [...]struct{ name, tag string }{
		{"Arabic", "arab"}, {"Syriac", "syrc"}, {"Nko", "nko "}, {"Mandaic", "mand"}, {"Manichaean", "mani"},
		{"Psalter_Pahlavi", "phlp"}, {"Adlam", "adlm"}, {"Hanifi_Rohingya", "rohg"}, {"Sogdian", "sogd"},
		{"Mongolian", "mong"}, {"Phags_Pa", "phag"},
	} {
		out = append(out, script{table: unicode.Scripts[s.name], tag: MakeTag(s.tag), shaper: arabicShaper})
	}
	for _, s := range [...]struct{ name, tag, newer string }{
		{"Devanagari", "deva", "dev2"}, {"Bengali", "beng", "bng2"}, {"Gurmukhi", "guru", "gur2"},
		{"Gujarati", "gujr", "gjr2"}, {"Oriya", "orya", "ory2"}, {"Tamil", "taml", "tml2"},
		{"Telugu", "telu", "tel2"}, {"Kannada", "knda", "knd2"}, {"Malayalam", "mlym", "mlm2"},
	} {
		out = append(out, script{table: unicode.Scripts[s.name], tag: MakeTag(s.tag), newer: MakeTag(s.newer),
			shaper: indicShaper})
	}
	out = append(out, script{table: unicode.Khmer, tag: MakeTag("khmr"), shaper: khmerShaper},
		script{table: unicode.Myanmar, tag: mymr, newer: MakeTag("mym2"), shaper: myanmarShaper})
	for _, s := range [...]struct{ name, tag string }{
		{"Tibetan", "tibt"}, {"Sinhala", "sinh"}, {"Buhid", "buhd"}, {"Hanunoo", "hano"}, {"Tagalog", "tglg"},
		{"Tagbanwa", "tagb"}, {"Limbu", "limb"}, {"Tai_Le", "tale"}, {"Buginese", "bugi"}, {"Kharoshthi", "khar"},
		{"Syloti_Nagri", "sylo"}, {"Tifinagh", "tfng"}, {"Balinese", "bali"}, {"Cham", "cham"}, {"Kayah_Li", "kali"},
		{"Lepcha", "lepc"}, {"Rejang", "rjng"}, {"Saurashtra", "saur"}, {"Sundanese", "sund"},
		{"Egyptian_Hieroglyphs", "egyp"}, {"Javanese", "java"}, {"Kaithi", "kthi"}, {"Meetei_Mayek", "mtei"},
		{"Tai_Tham", "lana"}, {"Tai_Viet", "tavt"}, {"Batak", "batk"}, {"Brahmi", "brah"}, {"Chakma", "cakm"},
		{"Miao", "plrd"}, {"Sharada", "shrd"}, {"Takri", "takr"}, {"Duployan", "dupl"}, {"Grantha", "gran"},
		{"Khojki", "khoj"}, {"Khudawadi", "sind"}, {"Mahajani", "mahj"}, {"Modi", "modi"},
		{"Pahawh_Hmong", "hmng"}, {"Siddham", "sidd"}, {"Tirhuta", "tirh"}, {"Ahom", "ahom"}, {"Multani", "mult"},
		{"Bhaiksuki", "bhks"}, {"Marchen", "marc"}, {"Newa", "newa"}, {"Masaram_Gondi", "gonm"},
		{"Soyombo", "soyo"}, {"Zanabazar_Square", "zanb"}, {"Dogra", "dogr"}, {"Gunjala_Gondi", "gong"},
		{"Makasar", "maka"}, {"Medefaidrin", "medf"}, {"Old_Sogdian", "sogo"}, {"Elymaic", "elym"},
		{"Nandinagari", "nand"}, {"Nyiakeng_Puachue_Hmong", "hmnp"}, {"Wancho", "wcho"}, {"Chorasmian", "chrs"},
		{"Dives_Akuru", "diak"}, {"Khitan_Small_Script", "kits"}, {"Yezidi", "yezi"}, {"Cypro_Minoan", "cpmn"},
		{"Old_Uyghur", "ougr"}, {"Tangsa", "tnsa"}, {"Toto", "toto"}, {"Vithkuqi", "vith"}, {"Kawi", "kawi"},
		{"Nag_Mundari", "nagm"}, {"Garay", "gara"}, {"Gurung_Khema", "gukh"}, {"Kirat_Rai", "krai"},
		{"Ol_Onal", "onao"}, {"Sunuwar", "sunu"}, {"Todhri", "todr"}, {"Tulu_Tigalari", "tutg"},
		{"Beria_Erfe", "berf"}, {"Sidetic", "sidt"}, {"Tai_Yo", "tayo"}, {"Tolong_Siki", "tols"},
	} {
		if t := unicode.Scripts[s.name]; t != nil {
			out = append(out, script{table: t, tag: MakeTag(s.tag), shaper: useShaper})
		}
	}
	return out
}()

// rtlScripts are the scripts written right to left.
//
//nolint:gochecknoglobals // a fixed list
var rtlScripts = tags("arab", "hebr", "syrc", "thaa", "nko ", "mand", "mani", "phlp", "adlm", "rohg", "sogd",
	"samr")

func tags(names ...string) []Tag {
	out := make([]Tag, len(names))
	for i, n := range names {
		out[i] = MakeTag(n)
	}
	return out
}

// Script is the OpenType script of text: its first character's of a
// script (ScriptOf), DFLT for a script it does not tell apart, Latin for
// text of none.
func Script(text string) Tag {
	for _, r := range text {
		if s, ok := ScriptOf(r); ok {
			return s
		}
	}
	return latn
}

// ScriptOf is r's OpenType script, DFLT for one Script does not tell
// apart, and false for a character of none: a combining mark, a digit,
// a space or punctuation, which take the script of the text round them.
func ScriptOf(r rune) (Tag, bool) {
	if !unicode.In(r, unicode.L, unicode.M, unicode.N) || unicode.In(r, unicode.Common, unicode.Inherited) {
		return 0, false
	}
	if s := scriptOf(r); s != nil {
		return s.tag, true
	}
	return dflt, true
}

func scriptOf(r rune) *script {
	for i := range scripts {
		if unicode.Is(scripts[i].table, r) {
			return &scripts[i]
		}
	}
	return nil
}

// scriptByTag is the script of tag, nil for one Script does not tell
// apart.
func scriptByTag(tag Tag) *script {
	for i := range scripts {
		if scripts[i].tag == tag {
			return &scripts[i]
		}
	}
	return nil
}

// RightToLeft reports whether script is written right to left.
func RightToLeft(script Tag) bool { return slices.Contains(rtlScripts, script) }

// candidates are the tags a layout table is looked up by for script, as
// HarfBuzz tries them: an Indic script's third and second versions
// before its first, Myanmar's second before its first.
func candidates(script Tag) []Tag {
	s := scriptByTag(script)
	switch {
	case s == nil || s.newer == 0:
		return []Tag{script}
	case s.shaper == myanmarShaper:
		return []Tag{s.newer, script}
	default:
		return []Tag{s.newer&^0xff | '3', s.newer, script}
	}
}

// shaperFor is the shaper of runs in script for a font whose GSUB's
// language system is chosen's (0 for none), as HarfBuzz categorizes it:
// a font that has only DFLT or Latin features for an Indic, Myanmar or
// USE script is shaped as Latin, and an Indic font of third version
// tags by the USE.
func shaperFor(script Tag, chosen Tag) shaper {
	s := scriptByTag(script)
	if s == nil {
		return defaultShaper
	}
	generic := chosen == dflt || chosen == latn
	switch s.shaper { //nolint:exhaustive // the shapers that do something here
	case indicShaper:
		switch {
		case generic:
			return defaultShaper
		case chosen&0xff == '3':
			return useShaper
		}
	case myanmarShaper:
		if generic || chosen == mymr {
			return defaultShaper
		}
	case useShaper:
		if generic {
			return defaultShaper
		}
	}
	return s.shaper
}

// Feature flags, as HarfBuzz's map builder takes them: a manual joiner
// is matched rather than passed over, and a feature per syllable
// matches only glyphs of the syllable its lookup starts at.
const (
	manualZWJ uint8 = 1 << iota
	manualZWNJ
	perSyllable
	manualJoiners = manualZWJ | manualZWNJ
)

// request is a feature a run of a script asks for, as HarfBuzz's map
// builder collects them: global ones apply to every glyph, the others
// to those a shaper marks; a stage's lookups all apply before the next
// stage's, and its pause after them.
type request struct {
	tag    Tag
	value  uint32
	stage  int
	global bool
	flags  uint8
}

// collector collects the features of a plan stage by stage.
type collector struct {
	out    []request
	stage  int
	pauses map[int]pause
}

// pause is what a shaper does to the run between two stages.
type pause func(p *Plan, b *buffer)

func (c *collector) add(global bool, flags uint8, names ...string) {
	for _, n := range names {
		c.out = append(c.out, request{MakeTag(n), 1, c.stage, global, flags})
	}
}

func (c *collector) on(names ...string) { c.add(true, 0, names...) }

// pause ends the stage, fn running after its lookups.
func (c *collector) pause(fn pause) {
	if fn != nil {
		c.pauses[c.stage] = fn
	}
	c.stage++
}

// requests are the features a run of script asks for, those its shaper
// has on and features after them, merged by tag: the last one's value,
// the first one's flags and stage.
//
//nolint:funlen // HarfBuzz's collection, stage after stage
func requests(script Tag, sh shaper, features []Feature) ([]request, map[int]pause) {
	c := &collector{pauses: map[int]pause{}}
	c.on("rvrn")
	c.pause(nil)
	if RightToLeft(script) {
		c.on("rtla")
		c.add(false, 0, "rtlm")
	} else {
		c.on("ltra", "ltrm")
	}
	c.add(false, 0, "frac", "numr", "dnom")
	c.on("rand", "trak")
	switch sh { //nolint:exhaustive // the shapers that do something here
	case arabicShaper:
		c.on("stch")
		c.pause(nil)
		c.add(true, manualZWJ, "ccmp", "locl")
		c.pause(nil)
		for _, t := range formTags {
			c.add(false, 0, t.String())
			c.pause(nil)
		}
		c.pause(nil)
		c.add(true, manualZWJ, "rlig")
		c.pause(nil)
		c.add(true, manualZWJ, "rclt", "calt")
		c.pause(nil)
		c.on("mset")
	case indicShaper:
		collectIndic(c)
	case khmerShaper:
		collectKhmer(c)
	case myanmarShaper:
		collectMyanmar(c)
	case useShaper:
		collectUSE(c)
	}
	c.on("abvm", "blwm", "ccmp", "locl")
	c.add(true, manualJoiners, "mark", "mkmk")
	c.on("rlig", "calt", "clig", "curs", "dist", "kern", "liga", "rclt")
	for _, f := range features {
		c.out = append(c.out, request{f.Tag, f.Value, c.stage, true, 0})
	}
	switch sh { //nolint:exhaustive // the shapers that do something here
	case indicShaper:
		c.add(true, 0, "liga")
		c.out[len(c.out)-1].value = 0
		c.pause(nil)
	case khmerShaper:
		c.on("clig")
		c.add(true, 0, "liga")
		c.out[len(c.out)-1].value = 0
	}
	slices.SortStableFunc(c.out, func(a, b request) int { return cmp.Compare(a.tag, b.tag) })
	merged := c.out[:0]
	for _, r := range c.out {
		if n := len(merged); n > 0 && merged[n-1].tag == r.tag {
			m := &merged[n-1]
			m.stage = min(m.stage, r.stage)
			if r.global {
				m.global, m.value = true, r.value
			} else {
				m.global, m.value = false, max(m.value, r.value)
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged, c.pauses
}

// planned is a lookup of a plan: the glyphs it applies to are those of
// its mask, and an alternate substitution takes its value-th.
type planned struct {
	lk    *lookup
	value uint32
	mask  uint32
	stage int
	index uint16
	// autoZWJ and autoZWNJ are whether it passes over joiners it does
	// not match, perSyllable whether it matches only within a syllable.
	autoZWJ, autoZWNJ, perSyllable bool
}

// plan is l's lookups of the features asked for that its language
// system ls has, a stage after the other, in their list's order in
// each; GPOS's in one stage.
func (l *layout) plan(ls data, asked []request, masks map[Tag]uint32) []planned {
	stages := map[Tag]request{}
	for _, r := range asked {
		stages[r.tag] = r
	}
	var out []planned
	l.features(ls, func(tag Tag, f data, required bool) {
		r, ok := stages[tag]
		mask := masks[tag]
		switch {
		case required:
			r.value, mask = 1, globalMask
		case !ok || r.value == 0 || mask == 0:
			return
		}
		if l.gpos {
			r.stage = 0
		}
		for k := range int(f.u16(2)) {
			i := f.u16(4 + 2*k)
			if lk := l.lookup(i); lk != nil {
				out = append(out, planned{lk: lk, value: r.value, mask: mask, stage: r.stage, index: i,
					autoZWJ: r.flags&manualZWJ == 0, autoZWNJ: r.flags&manualZWNJ == 0,
					perSyllable: r.flags&perSyllable != 0})
			}
		}
	})
	slices.SortStableFunc(out, func(a, b planned) int {
		if a.stage != b.stage {
			return a.stage - b.stage
		}
		return int(a.index) - int(b.index)
	})
	merged := out[:0]
	for _, p := range out {
		if n := len(merged); n > 0 && merged[n-1].index == p.index && merged[n-1].stage == p.stage {
			m := &merged[n-1]
			m.mask |= p.mask
			m.autoZWJ = m.autoZWJ && p.autoZWJ
			m.autoZWNJ = m.autoZWNJ && p.autoZWNJ
			continue
		}
		merged = append(merged, p)
	}
	return merged
}

// globalMask is the mask every glyph has, that of the global features.
const globalMask = 1

// masks are the masks of the features asked for: the global mask, or a
// bit of its own for each other, which a shaper gives the glyphs it
// applies to; none for a feature turned off.
func masks(asked []request) map[Tag]uint32 {
	out := map[Tag]uint32{}
	bit := uint32(globalMask)
	for _, r := range asked {
		switch {
		case r.value == 0:
		case r.global:
			out[r.tag] = globalMask
		default:
			bit <<= 1
			out[r.tag] = bit
		}
	}
	return out
}
