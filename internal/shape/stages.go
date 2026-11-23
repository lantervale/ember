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
	arab = MakeTag("arab")
)

// scripts are the scripts Script tells apart, by their OpenType tags.
//
//nolint:gochecknoglobals // a fixed table
var scripts = []struct {
	table *unicode.RangeTable
	tag   Tag
}{
	{unicode.Latin, latn}, {unicode.Greek, MakeTag("grek")}, {unicode.Cyrillic, MakeTag("cyrl")},
	{unicode.Arabic, arab}, {unicode.Hebrew, MakeTag("hebr")}, {unicode.Syriac, MakeTag("syrc")},
	{unicode.Thaana, MakeTag("thaa")}, {unicode.Nko, MakeTag("nko")}, {unicode.Mandaic, MakeTag("mand")},
	{unicode.Manichaean, MakeTag("mani")}, {unicode.Psalter_Pahlavi, MakeTag("phlp")},
	{unicode.Adlam, MakeTag("adlm")}, {unicode.Hanifi_Rohingya, MakeTag("rohg")},
	{unicode.Sogdian, MakeTag("sogd")}, {unicode.Samaritan, MakeTag("samr")},
	{unicode.Mongolian, MakeTag("mong")}, {unicode.Phags_Pa, MakeTag("phag")},
}

// joiningScripts are those HarfBuzz shapes with its Arabic shaper,
// whose letters join; rtlScripts those written right to left.
//
//nolint:gochecknoglobals // fixed lists
var (
	joiningScripts = tags("arab", "syrc", "nko", "mand", "mani", "phlp", "adlm", "rohg", "sogd", "mong", "phag")
	rtlScripts     = tags("arab", "hebr", "syrc", "thaa", "nko", "mand", "mani", "phlp", "adlm", "rohg", "sogd",
		"samr")
)

func tags(names ...string) []Tag {
	out := make([]Tag, len(names))
	for i, n := range names {
		out[i] = MakeTag(n)
	}
	return out
}

// Script is the OpenType script of text: its first letter's, DFLT for a
// script it does not tell apart, Latin for text of none.
func Script(text string) Tag {
	for _, r := range text {
		if !unicode.IsLetter(r) || unicode.In(r, unicode.Common, unicode.Inherited) {
			continue
		}
		for _, s := range scripts {
			if unicode.Is(s.table, r) {
				return s.tag
			}
		}
		return dflt
	}
	return latn
}

// RightToLeft reports whether script is written right to left.
func RightToLeft(script Tag) bool { return slices.Contains(rtlScripts, script) }

// request is a feature a run of a script asks for, as HarfBuzz's map
// builder collects them: global ones apply to every glyph, the others
// to those a shaper marks; a stage's lookups all apply before the next
// stage's.
type request struct {
	tag    Tag
	value  uint32
	stage  int
	global bool
}

// requests are the features a run of script asks for, those its shaper
// has on and features after them, merged by tag: the last one's value,
// the first stage.
func requests(script Tag, features []Feature) []request {
	var out []request
	stage := 0
	on := func(global bool, names ...string) {
		for _, n := range names {
			out = append(out, request{MakeTag(n), 1, stage, global})
		}
	}
	on(true, "rvrn")
	stage++
	if RightToLeft(script) {
		on(true, "rtla")
		on(false, "rtlm")
	} else {
		on(true, "ltra", "ltrm")
	}
	on(false, "frac", "numr", "dnom")
	on(true, "rand", "trak")
	if slices.Contains(joiningScripts, script) {
		on(true, "stch")
		stage++
		on(true, "ccmp", "locl")
		stage++
		for _, t := range formTags {
			on(false, t.String())
			stage++
		}
		stage++
		on(true, "rlig")
		stage++
		on(true, "rclt", "calt")
		stage++
		on(true, "mset")
	}
	on(true, "abvm", "blwm", "ccmp", "locl", "mark", "mkmk", "rlig")
	on(true, "calt", "clig", "curs", "dist", "kern", "liga", "rclt")
	for _, f := range features {
		out = append(out, request{f.Tag, f.Value, stage, true})
	}
	slices.SortStableFunc(out, func(a, b request) int { return cmp.Compare(a.tag, b.tag) })
	merged := out[:0]
	for _, r := range out {
		if n := len(merged); n > 0 && merged[n-1].tag == r.tag {
			m := &merged[n-1]
			m.stage = min(m.stage, r.stage)
			m.global = r.global
			if r.global {
				m.value = r.value
			}
			continue
		}
		merged = append(merged, r)
	}
	return merged
}

// planned is a lookup of a plan: the glyphs it applies to are those of
// its mask, and an alternate substitution takes its value-th.
type planned struct {
	lk    *lookup
	value uint32
	mask  uint32
	stage int
	index uint16
}

// plan is l's lookups of the features asked for that script's language
// system has, a stage after the other, in their list's order in each.
func (l *layout) plan(script Tag, asked []request, masks map[Tag]uint32) []planned {
	stages := map[Tag]request{}
	for _, r := range asked {
		stages[r.tag] = r
	}
	var out []planned
	l.features(script, func(tag Tag, f data, required bool) {
		r, ok := stages[tag]
		mask := masks[tag]
		switch {
		case required:
			r.value, mask = 1, globalMask
		case !ok || r.value == 0 || mask == 0:
			return
		}
		for k := range int(f.u16(2)) {
			i := f.u16(4 + 2*k)
			if lk := l.lookup(i); lk != nil {
				out = append(out, planned{lk: lk, value: r.value, mask: mask, stage: r.stage, index: i})
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
			merged[n-1].mask |= p.mask
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
