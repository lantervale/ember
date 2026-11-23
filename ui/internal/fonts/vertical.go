package fonts

import (
	"encoding/binary"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/ui/internal/style"
)

// verticalFeatures are the features of glyphs standing in vertical lines, HarfBuzz's for
// a line running down: Chrome turns on vert, never vrt2.
const verticalFeatures = "vert"

// vertical is a face's vertical metrics, in its units: vmtx's advance heights and top
// side bearings, and VORG's origins where it has one.
type vertical struct {
	advances, bearings []int16
	vorg               bool
	origin             int16
	origins            map[uint16]int16
}

// noVertical marks a face read and found without vertical metrics.
var noVertical = &vertical{} //nolint:gochecknoglobals // a sentinel

// verticalOf is c's vertical metrics, read once: nil for a face without both vhea and
// vmtx, which Chrome sets by its ascent and descent.
func (c *Face) verticalOf() *vertical {
	if c.vertical == nil {
		c.vertical = readVertical(c.TTF)
	}
	if c.vertical == noVertical {
		return nil
	}
	return c.vertical
}

func readVertical(ttf []byte) *vertical {
	tables := tableData(ttf)
	vhea, vmtx := tables["vhea"], tables["vmtx"]
	if len(vhea) < 36 || len(vmtx) < 4 {
		return noVertical
	}
	be := binary.BigEndian
	long := int(be.Uint16(vhea[34:]))
	if long == 0 || 4*long > len(vmtx) {
		return noVertical
	}
	v := &vertical{}
	for i := range long {
		v.advances = append(v.advances, int16(be.Uint16(vmtx[4*i:])))   //nolint:gosec // vmtx's fields
		v.bearings = append(v.bearings, int16(be.Uint16(vmtx[4*i+2:]))) //nolint:gosec // vmtx's fields
	}
	for at := 4 * long; at+2 <= len(vmtx); at += 2 {
		v.bearings = append(v.bearings, int16(be.Uint16(vmtx[at:]))) //nolint:gosec // vmtx's fields
	}
	if vorg := tables["VORG"]; len(vorg) >= 8 && be.Uint16(vorg) == 1 && be.Uint16(vorg[2:]) == 0 {
		v.vorg, v.origin = true, int16(be.Uint16(vorg[4:])) //nolint:gosec // VORG's fields
		v.origins = map[uint16]int16{}
		for i, at := 0, 8; i < int(be.Uint16(vorg[6:])) && at+4 <= len(vorg); i, at = i+1, at+4 {
			v.origins[be.Uint16(vorg[at:])] = int16(be.Uint16(vorg[at+2:])) //nolint:gosec // VORG's fields
		}
	}
	return v
}

// tableData is each table of ttf, one font file, by its tag.
func tableData(ttf []byte) map[string][]byte {
	be := binary.BigEndian
	out := map[string][]byte{}
	if len(ttf) < 12 {
		return out
	}
	for i := range int(be.Uint16(ttf[4:])) {
		rec := 12 + 16*i
		if rec+16 > len(ttf) {
			break
		}
		off, n := int(be.Uint32(ttf[rec+8:])), int(be.Uint32(ttf[rec+12:]))
		if off > 0 && n > 0 && off+n <= len(ttf) {
			out[string(ttf[rec:rec+4])] = ttf[off : off+n]
		}
	}
	return out
}

// advance is glyph g's advance down a vertical line.
func (v *vertical) advance(g uint16) int16 {
	return v.advances[min(int(g), len(v.advances)-1)]
}

// uprightKey is a grapheme cluster set upright in a face with its features.
type uprightKey struct {
	face     *Face
	features string
	text     string
}

// uprightSet is a cluster set upright: its advance down the line and its glyphs'
// baseline below the top of its room, in its face's units.
type uprightSet struct{ advance, origin float32 }

// Upright is how text, one grapheme cluster standing in a vertical line, is set by the
// vertical metrics (vhea, vmtx, VORG) of the face of f drawing it, in its glyphs for
// vertical lines (vert): how far it takes down the line, and how far below the top of
// that room its glyphs' baseline stands. ok is false when that face has none, where the
// cluster takes the face's ascent and descent.
func (s *Set) Upright(f style.Font, text string) (advance, baseline float32, ok bool) {
	r, _ := utf8.DecodeRuneInString(text)
	var c *Face
	if ch := s.chained(f); ch != nil {
		c, _, _ = ch.glyph(s, r)
	} else if c = s.Face(f); c != nil && !s.hasGlyph(c, r) {
		c = nil
	}
	if c == nil || c.verticalOf() == nil {
		return 0, 0, false
	}
	key := uprightKey{c, f.Features, text}
	set, seen := s.uprights[key]
	if !seen {
		set = s.setUpright(c, f.Features, text)
		if s.uprights == nil || len(s.uprights) >= maxShaped {
			s.uprights = map[uprightKey]uprightSet{}
		}
		s.uprights[key] = set
	}
	scale := f.Size * c.Look.adjust() / c.upem
	return set.advance * scale, set.origin * scale, true
}

// setUpright sets text upright in c, shaped with features and vert: the sum of its
// glyphs' advances, and its first glyph's origin, VORG's or else its top side bearing
// above its top, as Chrome's.
func (s *Set) setUpright(c *Face, features, text string) uprightSet {
	v := c.verticalOf()
	var run []shape.Glyph
	for i, r := range text {
		g, err := c.font.GlyphIndex(&s.buf, r)
		if err != nil {
			continue
		}
		//nolint:gosec // a glyph index and a text offset
		run = append(run, shape.Glyph{ID: uint16(g), Cluster: int32(i), Char: r})
	}
	if len(run) == 0 {
		return uprightSet{}
	}
	if p := c.layout().Plan(shape.Script(text), c.joined(shape.Splice(features, verticalFeatures))); p.Touches(run) {
		run = p.Shape(run)
	}
	var set uprightSet
	for _, g := range run {
		set.advance += float32(v.advance(g.ID))
	}
	first := run[0].ID
	switch o, has := v.origins[first]; {
	case has:
		set.origin = float32(o)
	case v.vorg:
		set.origin = float32(v.origin)
	default:
		ppem := fixed.I(int(c.upem))
		b, _, err := c.font.GlyphBounds(&s.buf, sfnt.GlyphIndex(first), ppem, font.HintingNone)
		if err == nil && int(first) < len(v.bearings) {
			set.origin = float32(v.bearings[first]) - float32(b.Min.Y)/64
		}
	}
	return set
}
