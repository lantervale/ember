package varfonttest

import (
	"encoding/binary"
	"math"
	"slices"

	"golang.org/x/image/font/sfnt"
)

// What the layout tables Options.Layout adds do, in font units and
// normalized coordinates (wght's after avar: 650 is 0.75, 900 is 1).
//
// GPOS kerns A V by -80 and T o by -120 (pairs of glyphs), and L Y by
// -100 (pairs of classes), each varied by GDEF's store: A V by -160 at
// wght's maximum, +80 at its minimum and -60 at wdth's maximum, T o by
// -120 at wght's maximum, L Y by -120 at wght's maximum and +40 at its
// minimum. Its mark feature puts ^, a mark in GDEF, on x by an anchor
// that moves 140 right at wght's maximum and 180 at wdth's, and 160 up
// at wght's maximum and 40 down at its minimum. From wdth 0.5 (125) up,
// its feature variations kern A V by -400 and nothing else.
//
// GSUB's rvrn and liga do nothing at the default instance. Its feature
// variations, the first that matches winning: from wght 0.5 up, rvrn
// draws $ as S; to wght -0.5 (250) down, rvrn draws $ as s and liga z
// as Z; from wdth 0.5 with wght between -0.4 and 0.4, liga draws z as Z.
const (
	KernAV       = -80
	KernTo       = -120
	KernLY       = -100
	KernAVWide   = -400
	MarkAnchorX  = 300
	MarkAnchorY  = 1000
	BaseAnchorX  = 512
	BaseAnchorY  = 1100
	VariedAt     = 0.5
	ReversedUpTo = -0.5
)

// The deltas of GDEF's store, one row an item, over its regions: wght's
// maximum, wght's minimum, wdth's maximum.
//
//nolint:gochecknoglobals // the font's deltas
var Deltas = [][3]int16{
	{-160, 80, -60}, // A V
	{-120, 0, 0},    // T o
	{140, 0, 180},   // x's anchor across
	{160, -40, 0},   // x's anchor up
	{-120, 40, 0},   // L Y
}

// layout adds GDEF, GSUB and GPOS to tables, for the glyphs of src.
func layout(tables map[string][]byte, src []byte) {
	f, err := sfnt.Parse(src)
	if err != nil {
		panic(err)
	}
	var buf sfnt.Buffer
	gid := func(r rune) uint16 {
		g, gerr := f.GlyphIndex(&buf, r)
		if gerr != nil || g == 0 {
			panic("varfonttest: the source has no glyph for " + string(r))
		}
		return uint16(g)
	}
	tables["GDEF"] = gdef(gid)
	tables["GSUB"] = gsub(gid)
	tables["GPOS"] = gpos(gid)
}

// link is an offset field of a table's head and the table it points to,
// put after the head; nil leaves the offset 0.
type link struct {
	at    int
	wide  bool
	table []byte
}

// table is head followed by the tables its links point to.
func table(head []byte, links ...link) []byte {
	out := slices.Clone(head)
	for _, l := range links {
		if l.table == nil {
			continue
		}
		if l.wide {
			binary.BigEndian.PutUint32(out[l.at:], uint32(len(out))) //nolint:gosec // small
		} else {
			binary.BigEndian.PutUint16(out[l.at:], uint16(len(out))) //nolint:gosec // small
		}
		out = append(out, l.table...)
	}
	return out
}

func u16s(vs ...int) []byte {
	out := make([]byte, 0, 2*len(vs))
	for _, v := range vs {
		out = binary.BigEndian.AppendUint16(out, uint16(int16(v))) //nolint:gosec // 16-bit fields
	}
	return out
}

func coverage(gs ...uint16) []byte {
	slices.Sort(gs)
	out := u16s(1, len(gs))
	for _, g := range gs {
		out = binary.BigEndian.AppendUint16(out, g)
	}
	return out
}

// classes is a class definition of format 1 giving each glyph its class.
func classes(of map[uint16]int) []byte {
	var lo, hi uint16 = 0xffff, 0
	for g := range of {
		lo, hi = min(lo, g), max(hi, g)
	}
	out := u16s(1, int(lo), int(hi-lo)+1)
	for g := lo; g <= hi; g++ {
		out = append(out, u16s(of[g])...)
	}
	return out
}

func device(item int) []byte { return u16s(0, item, 0x8000) }

func f2(v float32) int { return int(math.Round(float64(v) * 16384)) }

func gdef(gid func(rune) uint16) []byte {
	head := slices.Concat(u16s(1, 3), u16s(0, 0, 0, 0, 0), []byte{0, 0, 0, 0})
	return table(head, link{at: 4, table: classes(map[uint16]int{gid('x'): 1, gid('^'): 3})},
		link{at: 14, wide: true, table: store()})
}

// store is GDEF's ItemVariationStore: Deltas over three regions.
func store() []byte {
	regions := [][2][3]float32{{{0, 1, 1}, {}}, {{-1, -1, 0}, {}}, {{}, {0, 1, 1}}}
	list := u16s(2, len(regions))
	for _, r := range regions {
		for _, axis := range r {
			for _, v := range axis {
				list = append(list, u16s(f2(v))...)
			}
		}
	}
	data := u16s(len(Deltas), len(regions), len(regions), 0, 1, 2)
	for _, row := range Deltas {
		for _, d := range row {
			data = append(data, u16s(int(d))...)
		}
	}
	head := slices.Concat(u16s(1), []byte{0, 0, 0, 0}, u16s(1), []byte{0, 0, 0, 0})
	return table(head, link{at: 2, wide: true, table: list}, link{at: 8, wide: true, table: data})
}

// scripts is a script list of DFLT and latn whose default language
// systems have features 0 to n-1.
func scripts(n int) []byte {
	ls := u16s(0, 0xffff, n)
	for i := range n {
		ls = append(ls, u16s(i)...)
	}
	script := table(u16s(0, 0), link{at: 0, table: ls})
	head := slices.Concat(u16s(2), []byte("DFLT"), u16s(0), []byte("latn"), u16s(0))
	return table(head, link{at: 6, table: script}, link{at: 12, table: script})
}

func feature(lookups ...int) []byte { return u16s(append([]int{0, len(lookups)}, lookups...)...) }

func features(tags []string, lookups [][]int) []byte {
	head := u16s(len(tags))
	links := make([]link, len(tags))
	for i, t := range tags {
		head = append(append(head, t...), 0, 0)
		links[i] = link{at: 2 + 6*i + 4, table: feature(lookups[i]...)}
	}
	return table(head, links...)
}

func lookups(kinds []int, subtables [][]byte) []byte {
	head := u16s(len(kinds))
	links := make([]link, len(kinds))
	for i, k := range kinds {
		head = append(head, 0, 0)
		links[i] = link{at: 2 + 2*i, table: table(u16s(k, 0, 1, 0), link{at: 6, table: subtables[i]})}
	}
	return table(head, links...)
}

// condition is a condition of format 1: axis between lo and hi.
type condition struct {
	axis   int
	lo, hi float32
}

// variation is a feature variation: its conditions, and the features it
// gives other lookups, by index.
type variation struct {
	when []condition
	subs map[int][]int
}

func featureVariations(vs []variation) []byte {
	head := slices.Concat(u16s(1, 0), []byte{0, 0, 0, uint8(len(vs))}) //nolint:gosec // a few
	var links []link
	for i, v := range vs {
		head = append(head, make([]byte, 8)...)
		set := u16s(len(v.when))
		var conds []link
		for k, c := range v.when {
			set = append(set, 0, 0, 0, 0)
			conds = append(conds, link{at: 2 + 4*k, wide: true, table: u16s(1, c.axis, f2(c.lo), f2(c.hi))})
		}
		keys := make([]int, 0, len(v.subs))
		for k := range v.subs {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		subst := u16s(1, 0, len(keys))
		var alts []link
		for k, index := range keys {
			subst = append(subst, u16s(index, 0, 0)...)
			alts = append(alts, link{at: 6 + 6*k + 2, wide: true, table: feature(v.subs[index]...)})
		}
		links = append(links, link{at: 8 + 8*i, wide: true, table: table(set, conds...)},
			link{at: 8 + 8*i + 4, wide: true, table: table(subst, alts...)})
	}
	return table(head, links...)
}

// head11 is the head of a GSUB or GPOS of version 1.1.
func head11() []byte { return slices.Concat(u16s(1, 1, 0, 0, 0), []byte{0, 0, 0, 0}) }

func gsub(gid func(rune) uint16) []byte {
	single := func(from, to rune) []byte {
		return table(u16s(2, 0, 1, int(gid(to))), link{at: 2, table: coverage(gid(from))})
	}
	return table(head11(), link{at: 4, table: scripts(2)},
		link{at: 6, table: features([]string{"rvrn", "liga"}, [][]int{nil, nil})},
		link{at: 8, table: lookups([]int{1, 1, 1}, [][]byte{single('$', 'S'), single('$', 's'), single('z', 'Z')})},
		link{at: 10, wide: true, table: featureVariations([]variation{
			{when: []condition{{0, VariedAt, 1}}, subs: map[int][]int{0: {0}}},
			{when: []condition{{0, -1, ReversedUpTo}}, subs: map[int][]int{0: {1}, 1: {2}}},
			{when: []condition{{1, VariedAt, 1}, {0, -0.4, 0.4}}, subs: map[int][]int{1: {2}}},
		})})
}

const (
	xAdvance       = 0x0004
	xAdvanceDevice = 0x0040
)

func gpos(gid func(rune) uint16) []byte {
	// pairSet is a pair set of one pair, its device's offset from the set.
	pairSet := func(second rune, kern, item int) []byte {
		if item < 0 {
			return u16s(1, int(gid(second)), kern)
		}
		return table(u16s(1, int(gid(second)), kern, 0), link{at: 6, table: device(item)})
	}
	pairs := table(u16s(1, 0, xAdvance|xAdvanceDevice, 0, 2, 0, 0), link{at: 2, table: coverage(gid('A'), gid('T'))},
		link{at: 10, table: pairSet('V', KernAV, 0)}, link{at: 12, table: pairSet('o', KernTo, 1)})
	classPairs := table(u16s(2, 0, xAdvance|xAdvanceDevice, 0, 0, 0, 2, 2, 0, 0, 0, 0, 0, 0, KernLY, 0),
		link{at: 2, table: coverage(gid('L'))}, link{at: 8, table: classes(map[uint16]int{gid('L'): 1})},
		link{at: 10, table: classes(map[uint16]int{gid('Y'): 1})}, link{at: 30, table: device(4)})
	wide := table(u16s(1, 0, xAdvance, 0, 1, 0), link{at: 2, table: coverage(gid('A'))},
		link{at: 10, table: u16s(1, int(gid('V')), KernAVWide)})
	baseAnchor := table(u16s(3, BaseAnchorX, BaseAnchorY, 0, 0), link{at: 6, table: device(2)},
		link{at: 8, table: device(3)})
	markBase := table(u16s(1, 0, 0, 1, 0, 0), link{at: 2, table: coverage(gid('^'))},
		link{at: 4, table: coverage(gid('x'))},
		link{at: 8, table: table(u16s(1, 0, 0), link{at: 4, table: u16s(1, MarkAnchorX, MarkAnchorY)})},
		link{at: 10, table: table(u16s(1, 0), link{at: 2, table: baseAnchor})})
	kern := table(u16s(2, 0, 2, 0, 0), link{at: 6, table: pairs}, link{at: 8, table: classPairs})
	head := u16s(3, 0, 0, 0)
	lookupList := table(head, link{at: 2, table: kern}, link{at: 4, table: table(u16s(4, 0, 1, 0),
		link{at: 6, table: markBase})}, link{at: 6, table: table(u16s(2, 0, 1, 0), link{at: 6, table: wide})})
	return table(head11(), link{at: 4, table: scripts(2)},
		link{at: 6, table: features([]string{"kern", "mark"}, [][]int{{0}, {1}})},
		link{at: 8, table: lookupList},
		link{at: 10, wide: true, table: featureVariations([]variation{
			{when: []condition{{1, VariedAt, 1}}, subs: map[int][]int{0: {2}}},
		})})
}
