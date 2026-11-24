// Package shape applies a font's OpenType layout to a run of its glyphs,
// as HarfBuzz does for Chrome: GSUB's substitutions (ligatures, small
// capitals, figures, the joined forms of Arabic letters…) and GPOS's
// kerning, marks placed on their bases and cursive attachment, for the
// features a run asks for and those its script's shaper has on, with
// HarfBuzz's Indic, Khmer, Myanmar and Universal Shaping Engine
// reordering the syllables of the scripts they shape. It is pure Go and
// reads the tables itself.
package shape

import (
	"slices"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/internal/varfont"
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
	// varStore is GDEF's store of the variations of GPOS's values, nil
	// for none.
	varStore *varfont.ItemStore
	// uvs is cmap's subtable of variation sequences, nil for none.
	uvs data
	// kernTable is whether src has a kern table and GPOS no kern
	// feature, so kerning comes from the old table, as HarfBuzz falls
	// back to it.
	kernTable bool
	// arabic are the lookups of HarfBuzz's Arabic fallback, made of the
	// presentation forms of cmap when first asked.
	arabic *[7]*lookup

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
	// featVars are its feature variations, nil for none.
	featVars data
	// parsed are its lookups read so far, by index.
	parsed map[uint16]*lookup
}

func newLayout(t data, gpos bool) layout {
	l := layout{gpos: gpos, parsed: map[uint16]*lookup{}}
	if t == nil || t.u16(0) != 1 {
		return l
	}
	l.scripts, l.feats, l.lookups = t.sub(4), t.sub(6), t.sub(8)
	if t.u16(2) >= 1 {
		l.featVars = t.at(int(t.u32(10)))
	}
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
func Parse(src *sfnt.Font, ttf []byte) *Font { return ParseFont(src, ttf, 0) }

// ParseFont is Parse for the index-th font of file, a collection, or
// the font file itself.
func ParseFont(src *sfnt.Font, file []byte, index int) *Font {
	f := &Font{src: src, upem: int32(src.UnitsPerEm()), plans: map[planKey]*Plan{}}
	d := data(file)
	at := 0
	if Tag(d.u32(0)) == MakeTag("ttcf") {
		at = int(d.u32(12 + 4*index))
	}
	tables := map[Tag]data{}
	for i := range int(d.u16(at + 4)) {
		rec := at + 12 + 16*i
		off, n := int(d.u32(rec+8)), int(d.u32(rec+12))
		if off <= 0 || n <= 0 || off+n > len(d) {
			continue
		}
		tables[Tag(d.u32(rec))] = d[off : off+n]
	}
	gdef := tables[MakeTag("GDEF")]
	if slices.Contains(brokenGDEFs, [3]int{len(gdef), len(tables[MakeTag("GSUB")]), len(tables[MakeTag("GPOS")])}) {
		gdef = nil
	}
	if gdef != nil {
		f.glyphClasses = gdef.sub(4)
		f.markClasses = gdef.sub(10)
		if gdef.u32(0) >= 0x00010002 {
			f.markSets = gdef.sub(12)
		}
		if gdef.u32(0) >= 0x00010003 {
			if s, err := varfont.ReadItemStore(gdef, int(gdef.u32(14))); err == nil {
				f.varStore = s
			}
		}
	}
	f.uvs = variationSequences(tables[MakeTag("cmap")])
	f.gsub = newLayout(tables[MakeTag("GSUB")], false)
	f.gpos = newLayout(tables[MakeTag("GPOS")], true)
	f.kernTable = tables[MakeTag("kern")] != nil && !f.gpos.lists(kern)
	return f
}

// brokenGDEFs are the lengths of the GDEF, GSUB and GPOS tables of the
// fonts whose GDEF HarfBuzz leaves out, as it calls spacing glyphs
// marks: versions of Tahoma, Times New Roman Italic and Courier New
// (the Mac's among them), Microsoft Himalaya, Cantarell and Padauk.
//
//nolint:gochecknoglobals // HarfBuzz's list
var brokenGDEFs = [][3]int{
	{442, 2874, 42038}, {430, 2874, 40662}, {442, 2874, 39116}, {430, 2874, 39374}, {490, 3046, 41638},
	{478, 3046, 41902}, {898, 12554, 46470}, {910, 12566, 47732}, {928, 23298, 59332}, {940, 23310, 60732},
	{964, 23836, 60072}, {976, 23832, 61456}, {994, 24474, 60336}, {1006, 24470, 61740}, {1006, 24576, 61346},
	{1018, 24572, 62828}, {1006, 24576, 61352}, {1018, 24572, 62834}, {832, 7324, 47162}, {844, 7302, 45474},
	{180, 13054, 7254}, {192, 12638, 7254}, {192, 12690, 7254}, {188, 248, 3852}, {188, 264, 3426},
	{1058, 47032, 11818}, {1046, 47030, 12600}, {1058, 71796, 16770}, {1046, 71790, 17862}, {1046, 71788, 17112},
	{1058, 71794, 17514}, {1330, 109904, 57938}, {1330, 109904, 58972}, {1004, 59092, 14836}, {588, 5078, 14418},
	{588, 5078, 14238}, {894, 17162, 33960}, {894, 17154, 34472}, {816, 7868, 17052}, {816, 7868, 17138},
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
	extension := lk.kind == ext
	for i := range n {
		sub := t.sub(6 + 2*i)
		if sub == nil {
			continue
		}
		if extension {
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

// skipsClass reports whether a lookup of flag, and markSet where the
// flag says to use one, passes over glyph g of class.
func (f *Font) skipsClass(flag, markSet uint16, g, class uint16) bool {
	switch class {
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

// defaultAdvance is glyph id's advance in font units at the default
// instance.
func (f *Font) defaultAdvance(id uint16) (int32, bool) {
	a, err := f.src.GlyphAdvance(&f.buf, sfnt.GlyphIndex(id), fixed.I(int(f.upem)), font.HintingNone)
	return int32(a >> 6), err == nil
}

// isMark reports whether GDEF classes glyph g as a mark.
func (f *Font) isMark(g uint16) bool { return f.glyphClasses.class(g) == markGlyph }
