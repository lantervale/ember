// Package colourfont reads a font file's colour glyphs and draws them,
// in pure Go: COLR versions 0 and 1 (layers, gradients, transforms and
// composites) in the colours of a CPAL palette, the PNG bitmaps of sbix
// (Apple's emoji) and of CBDT and CBLC (Noto's), and the documents of an
// SVG table through the reader importing ui/svg registers (RegisterSVG),
// without which they are left to the font's other tables or outlines.
package colourfont

import (
	"encoding/binary"
	"image"
	"image/color"
	"sync"

	"golang.org/x/image/font/sfnt"
)

// Kind is how a glyph is coloured.
type Kind uint8

const (
	// None is a glyph drawn by its outline alone.
	None Kind = iota
	// Layers is a COLR glyph.
	Layers
	// Bitmap is an sbix or CBDT glyph.
	Bitmap
	// SVG is a glyph of the SVG table.
	SVG
)

// Font is a font file's colour glyphs.
type Font struct {
	outlines *sfnt.Font
	upem     float32
	// head's bounding box of every glyph, in font units, y up.
	xMin, yMin, xMax, yMax int16
	colr                   table
	palettes               []palette
	sbix                   table
	cblc, cbdt             table
	svg                    table
	numGlyphs              int
	mu                     sync.Mutex
	buf                    sfnt.Buffer
}

// palette is a CPAL palette and the backgrounds it is for.
type palette struct {
	colors []color.NRGBA
	types  uint32
}

// Parse reads the colour tables of the font file ttf, whose outlines
// sfnt read as outlines; nil when it has none.
func Parse(ttf []byte, outlines *sfnt.Font) *Font {
	tables := directory(ttf)
	f := &Font{outlines: outlines, colr: tables["COLR"], sbix: tables["sbix"], cblc: tables["CBLC"],
		cbdt: tables["CBDT"], svg: tables["SVG "]}
	head := tables["head"]
	f.upem = float32(head.u16(18))
	if f.upem == 0 {
		f.upem = 1000
	}
	f.xMin, f.yMin, f.xMax, f.yMax = head.i16(36), head.i16(38), head.i16(40), head.i16(42)
	f.numGlyphs = int(tables["maxp"].u16(4))
	f.palettes = readCPAL(tables["CPAL"])
	if len(f.colr) < 14 {
		f.colr = nil
	}
	if len(f.cblc) < 8 || len(f.cbdt) < 4 {
		f.cblc, f.cbdt = nil, nil
	}
	if len(f.sbix) < 8 {
		f.sbix = nil
	}
	if len(f.svg) < 10 {
		f.svg = nil
	}
	if f.colr == nil && f.sbix == nil && f.cblc == nil && f.svg == nil {
		return nil
	}
	return f
}

// directory is the tables of the font whose directory starts ttf, by tag.
func directory(ttf []byte) map[string]table {
	t := table(ttf)
	out := map[string]table{}
	n := int(t.u16(4))
	for i := range n {
		rec := 12 + 16*i
		off, size := int(t.u32(rec+8)), int(t.u32(rec+12))
		if off < 0 || size < 0 || off+size > len(ttf) || off+size < off {
			continue
		}
		out[string(t.bytes(rec, 4))] = t[off : off+size]
	}
	return out
}

// Kind is how glyph gid is coloured, the first of COLR, the bitmaps and
// the SVG table that has it, as Chrome prefers them; the SVG table only
// once a reader is registered.
func (f *Font) Kind(gid uint16) Kind {
	if f == nil {
		return None
	}
	switch {
	case f.colrHas(gid):
		return Layers
	case f.bitmapHas(gid):
		return Bitmap
	case f.svgDoc(gid) != nil:
		return SVG
	}
	return None
}

// Palettes is how many palettes the font has.
func (f *Font) Palettes() int { return len(f.palettes) }

// PaletteFor is the first palette for a dark background, or a light one
// when dark is false; 0 when none says.
func (f *Font) PaletteFor(dark bool) int {
	flag := uint32(1)
	if dark {
		flag = 2
	}
	for i, p := range f.palettes {
		if p.types&flag != 0 {
			return i
		}
	}
	return 0
}

// Override is a palette entry's colour in place of the font's.
type Override struct {
	Index int
	Color color.NRGBA
}

// Request is how a glyph is drawn.
type Request struct {
	// PPEM is the pixels an em is drawn at.
	PPEM float32
	// Foreground is the text's colour, which palette entry 0xFFFF and
	// an SVG document's currentColor stand for.
	Foreground color.NRGBA
	// Palette is the palette drawn with, Overrides its entries changed.
	Palette   int
	Overrides []Override
	// NoSVG leaves the SVG table's glyphs to their outlines, as Chrome.
	NoSVG bool
}

// Draw draws glyph gid in colour as req has it: the picture, alpha
// premultiplied, and where its top-left is in pixels from the glyph's
// origin, y down; ok is false for a glyph drawn by its outline.
func (f *Font) Draw(gid uint16, req Request) (*image.RGBA, image.Point, bool) {
	if f == nil || !(req.PPEM > 0) || req.PPEM > maxPPEM {
		return nil, image.Point{}, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case f.colrHas(gid):
		return f.drawCOLR(gid, req)
	case f.bitmapHas(gid):
		return f.drawBitmap(gid, req)
	case !req.NoSVG && f.svgDoc(gid) != nil:
		return f.drawSVG(gid, req)
	}
	return nil, image.Point{}, false
}

// maxPPEM is the largest em drawn: a bigger one is no text's.
const maxPPEM = 4096

// colour is entry i of the palette req draws with, alpha times alpha.
func (f *Font) colour(i uint16, alpha float32, req Request) [4]float32 {
	var c color.NRGBA
	switch i {
	case 0xFFFF:
		c = req.Foreground
	default:
		for _, o := range req.Overrides {
			if o.Index == int(i) {
				c = o.Color
				return premultiplied(c, alpha)
			}
		}
		p := req.Palette
		if p < 0 || p >= len(f.palettes) {
			p = 0
		}
		if p < len(f.palettes) && int(i) < len(f.palettes[p].colors) {
			c = f.palettes[p].colors[i]
		}
	}
	return premultiplied(c, alpha)
}

func premultiplied(c color.NRGBA, alpha float32) [4]float32 {
	a := float32(c.A) / 255 * min(max(alpha, 0), 1)
	return [4]float32{float32(c.R) / 255 * a, float32(c.G) / 255 * a, float32(c.B) / 255 * a, a}
}

func readCPAL(t table) []palette {
	if len(t) < 12 {
		return nil
	}
	version := t.u16(0)
	entries, n, records := int(t.u16(2)), int(t.u16(4)), int(t.u16(6))
	at := int(t.u32(8))
	var types table
	if version >= 1 && len(t) >= 12+2*n+4 {
		if off := int(t.u32(12 + 2*n)); off > 0 && off < len(t) {
			types = t[off:]
		}
	}
	out := make([]palette, 0, n)
	for p := range n {
		first := int(t.u16(12 + 2*p))
		pal := palette{colors: make([]color.NRGBA, entries)}
		for e := range entries {
			if first+e >= records {
				break
			}
			rec := at + 4*(first+e)
			pal.colors[e] = color.NRGBA{R: t.u8(rec + 2), G: t.u8(rec + 1), B: t.u8(rec), A: t.u8(rec + 3)}
		}
		if types != nil {
			pal.types = types.u32(4 * p)
		}
		out = append(out, pal)
	}
	return out
}

// table is a font table, read big-endian; a read past its end is 0.
type table []byte

func (t table) in(at, n int) bool { return at >= 0 && n >= 0 && at+n <= len(t) && at+n >= at }

func (t table) u8(at int) uint8 {
	if !t.in(at, 1) {
		return 0
	}
	return t[at]
}

func (t table) u16(at int) uint16 {
	if !t.in(at, 2) {
		return 0
	}
	return binary.BigEndian.Uint16(t[at:])
}

func (t table) i16(at int) int16 { return int16(t.u16(at)) } //nolint:gosec // a signed field

func (t table) u24(at int) uint32 {
	if !t.in(at, 3) {
		return 0
	}
	return uint32(t[at])<<16 | uint32(t[at+1])<<8 | uint32(t[at+2])
}

func (t table) u32(at int) uint32 {
	if !t.in(at, 4) {
		return 0
	}
	return binary.BigEndian.Uint32(t[at:])
}

func (t table) bytes(at, n int) []byte {
	if !t.in(at, n) {
		return nil
	}
	return t[at : at+n]
}

// f2dot14 is the 2.14 number at at.
func (t table) f2dot14(at int) float32 { return float32(t.i16(at)) / 16384 }

// fixed is the 16.16 number at at.
func (t table) fixed(at int) float32 { return float32(int32(t.u32(at))) / 65536 } //nolint:gosec // a signed field
