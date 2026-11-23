// Package shape applies a font's OpenType layout to a run of its glyphs,
// as HarfBuzz does for Chrome: GSUB's substitutions (ligatures, small
// capitals, figures, the joined forms of Arabic letters…) and GPOS's
// kerning, marks placed on their bases and cursive attachment, for the
// features a run asks for and those its script's shaper has on. It is
// pure Go and reads the tables itself; the reordering of Indic and other
// complex scripts is left out.
package shape

import (
	"sync"

	"golang.org/x/image/font/sfnt"
)

// Tag is a four-letter OpenType tag.
type Tag uint32

// MakeTag is the tag of s's first four bytes, padded with spaces.
func MakeTag(s string) Tag {
	var t Tag
	for i := range 4 {
		c := byte(' ')
		if i < len(s) {
			c = s[i]
		}
		t = t<<8 | Tag(c)
	}
	return t
}

func (t Tag) String() string {
	return string([]byte{byte(t >> 24), byte(t >> 16 & 0xff), byte(t >> 8 & 0xff), byte(t & 0xff)})
}

// Font is a font file's layout tables, with the plans made of them.
type Font struct {
	src          *sfnt.Font
	upem         int32
	gsub, gpos   layout
	glyphClasses data
	markClasses  data
	markSets     data
	// kernTable is whether src has a kern table and GPOS no kern
	// feature, so kerning comes from the old table, as HarfBuzz falls
	// back to it.
	kernTable bool

	// mu guards the plans and the lookups read, and the buffer.
	mu    sync.Mutex
	plans map[planKey]*Plan
	buf   sfnt.Buffer
}

// layout is a GSUB or GPOS table.
type layout struct {
	gpos    bool
	scripts data
	feats   data
	lookups data
	// parsed are its lookups read so far, by index.
	parsed map[uint16]*lookup
}

func newLayout(t data, gpos bool) layout {
	l := layout{gpos: gpos, parsed: map[uint16]*lookup{}}
	if t == nil || t.u16(0) != 1 {
		return l
	}
	l.scripts, l.feats, l.lookups = t.sub(4), t.sub(6), t.sub(8)
	return l
}

func (l *layout) lookupCount() int { return int(l.lookups.u16(0)) }

// lists reports whether l's feature list has feature t, for any script.
func (l *layout) lists(t Tag) bool {
	for i := range int(l.feats.u16(0)) {
		if Tag(l.feats.u32(2+6*i)) == t {
			return true
		}
	}
	return false
}

// Parse reads the layout tables of ttf, src's file: what it lacks or
// cannot read does nothing.
func Parse(src *sfnt.Font, ttf []byte) *Font {
	f := &Font{src: src, upem: int32(src.UnitsPerEm()), plans: map[planKey]*Plan{}}
	d := data(ttf)
	tables := map[Tag]data{}
	for i := range int(d.u16(4)) {
		rec := 12 + 16*i
		off, n := int(d.u32(rec+8)), int(d.u32(rec+12))
		if off <= 0 || n <= 0 || off+n > len(d) {
			continue
		}
		tables[Tag(d.u32(rec))] = d[off : off+n]
	}
	if gdef := tables[MakeTag("GDEF")]; gdef != nil {
		f.glyphClasses = gdef.sub(4)
		f.markClasses = gdef.sub(10)
		if gdef.u32(0) >= 0x00010002 {
			f.markSets = gdef.sub(12)
		}
	}
	f.gsub = newLayout(tables[MakeTag("GSUB")], false)
	f.gpos = newLayout(tables[MakeTag("GPOS")], true)
	f.kernTable = tables[MakeTag("kern")] != nil && !f.gpos.lists(kern)
	return f
}

// Glyph classes GDEF gives.
const (
	baseGlyph     = 1
	ligatureGlyph = 2
	markGlyph     = 3
)

// Lookup flags.
const (
	rightToLeft     = 0x1
	ignoreBase      = 0x2
	ignoreLigatures = 0x4
	ignoreMarks     = 0x8
	useMarkSet      = 0x10
	markAttachType  = 0xFF00
)

// Lookup kinds: GSUB's, and GPOS's that this package applies, its
// context and chained context lookups read as GSUB's, which they share
// the shape of.
const (
	single    = 1
	multiple  = 2
	alternate = 3
	ligature  = 4
	context   = 5
	chain     = 6
	gsubExt   = 7
	reverse   = 8

	adjust       = 101
	pair         = 102
	cursive      = 103
	markBase     = 104
	markLigature = 105
	markMark     = 106
	gposContext  = 7
	gposChain    = 8
	gposExt      = 9
	gposShift    = 100
)

// lookup is a lookup of a GSUB or GPOS table, its extension subtables
// resolved.
type lookup struct {
	kind uint16
	// positions is whether it is GPOS's.
	positions bool
	flag      uint16
	markSet   uint16
	subs      []data
	// first are the glyphs one of its subtables may start at.
	first glyphSet
}

// lookup is l's lookup i, read once; nil when there is none.
func (l *layout) lookup(i uint16) *lookup {
	if lk, ok := l.parsed[i]; ok {
		return lk
	}
	var lk *lookup
	if int(i) < l.lookupCount() {
		lk = readLookup(l.lookups.sub(2+2*int(i)), l.gpos)
	}
	l.parsed[i] = lk
	return lk
}

func readLookup(t data, gpos bool) *lookup {
	if t == nil {
		return nil
	}
	ext := uint16(gsubExt)
	if gpos {
		ext = gposExt
	}
	lk := &lookup{kind: t.u16(0), flag: t.u16(2), positions: gpos}
	n := int(t.u16(4))
	if lk.flag&useMarkSet != 0 {
		lk.markSet = t.u16(6 + 2*n)
	}
	for i := range n {
		sub := t.sub(6 + 2*i)
		if sub == nil {
			continue
		}
		if lk.kind == ext {
			if sub.u16(0) != 1 {
				continue
			}
			lk.kind = sub.u16(2)
			sub = sub.at(int(sub.u32(4)))
			if sub == nil {
				continue
			}
		}
		lk.subs = append(lk.subs, sub)
	}
	switch {
	case gpos && (lk.kind == gposContext || lk.kind == gposChain):
		lk.kind -= gposContext - context
	case gpos:
		lk.kind += gposShift
	}
	for _, s := range lk.subs {
		lk.firstGlyphs(s)
	}
	return lk
}

// firstGlyphs adds the glyphs subtable s may start at to lk.first.
func (lk *lookup) firstGlyphs(s data) {
	cov := s.sub(2)
	switch {
	case lk.kind == context && s.u16(0) == 3:
		cov = s.sub(6)
	case lk.kind == chain && s.u16(0) == 3:
		cov = s.sub(4 + 2*int(s.u16(2)) + 2)
	}
	cov.eachCovered(lk.first.add)
}

// ignores reports whether lk passes over glyph g, as its flag says.
func (f *Font) ignores(lk *lookup, g uint16) bool { return f.skips(lk.flag, lk.markSet, g) }

// skips reports whether a lookup of flag, and markSet where the flag
// says to use one, passes over glyph g.
func (f *Font) skips(flag, markSet uint16, g uint16) bool {
	if flag&(ignoreBase|ignoreLigatures|ignoreMarks|useMarkSet|markAttachType) == 0 {
		return false
	}
	switch f.glyphClasses.class(g) {
	case baseGlyph:
		return flag&ignoreBase != 0
	case ligatureGlyph:
		return flag&ignoreLigatures != 0
	case markGlyph:
		if flag&ignoreMarks != 0 {
			return true
		}
		if flag&useMarkSet != 0 {
			if int(markSet) >= int(f.markSets.u16(2)) {
				return true
			}
			return f.markSets.at(int(f.markSets.u32(4+4*int(markSet)))).coverage(g) < 0
		}
		if t := flag & markAttachType >> 8; t != 0 {
			return f.markClasses.class(g) != t
		}
	}
	return false
}

// isMark reports whether GDEF classes glyph g as a mark.
func (f *Font) isMark(g uint16) bool { return f.glyphClasses.class(g) == markGlyph }
