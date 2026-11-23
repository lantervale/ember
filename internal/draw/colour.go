package draw

import (
	stdcolor "image/color"
	"math"

	"golang.org/x/image/font/sfnt"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/colourfont"
	"github.com/lantervale/ember/render"
)

// Palette is which of a colour font's palettes its glyphs are drawn
// in, as font-palette has it: Index, or the first for a light or a dark
// background, its entries changed by Overrides.
type Palette struct {
	Index       int
	Light, Dark bool
	Overrides   []colourfont.Override
}

// colourKey is a colour glyph of a face drawn in a text colour.
type colourKey struct {
	face int
	idx  sfnt.GlyphIndex
	fg   stdcolor.NRGBA
}

// colourGlyph is a colour glyph's picture and where its top-left is
// from the glyph's origin, in the font's pixels, y down; tex is nil for
// one drawn by its outline.
type colourGlyph struct {
	tex  *render.Texture
	at   geom.Vec2
	size geom.Vec2
}

// maxColourGlyphs bounds the colour glyphs a font keeps drawn.
const maxColourGlyphs = 1 << 10

// colourOf is glyph idx of face in colour, drawn in the text colour fg:
// a picture for a COLR, bitmap or SVG glyph, none for another.
func (f *Font) colourOf(face int, idx sfnt.GlyphIndex, fg color.Color) colourGlyph {
	if face < 0 || face >= len(f.colours) || f.colours[face] == nil {
		return colourGlyph{}
	}
	cf := f.colours[face]
	look := f.looks[face]
	kind := cf.Kind(uint16(idx))
	if kind == colourfont.None || kind == colourfont.SVG && look.NoSVG {
		return colourGlyph{}
	}
	s := fg.SRGB()
	byte8 := func(v float32) uint8 { return uint8(math.Round(float64(min(max(v, 0), 1)) * 255)) }
	key := colourKey{face, idx, stdcolor.NRGBA{R: byte8(s[0]), G: byte8(s[1]), B: byte8(s[2]), A: 255}}
	if g, ok := f.colourGlyphs[key]; ok {
		return g
	}
	req := colourfont.Request{PPEM: f.size * oversample * look.scale(), Foreground: key.fg, NoSVG: look.NoSVG}
	if p := look.Palette; p != nil {
		req.Palette, req.Overrides = p.Index, p.Overrides
		if p.Light || p.Dark {
			req.Palette = cf.PaletteFor(p.Dark)
		}
	}
	var g colourGlyph
	if img, at, ok := cf.Draw(uint16(idx), req); ok {
		g = colourGlyph{
			tex:  &render.Texture{Image: img},
			at:   geom.V2(float32(at.X), float32(at.Y)),
			size: geom.V2(float32(img.Rect.Dx()), float32(img.Rect.Dy())),
		}
	}
	if f.colourGlyphs == nil || len(f.colourGlyphs) >= maxColourGlyphs {
		f.colourGlyphs = map[colourKey]colourGlyph{}
	}
	f.colourGlyphs[key] = g
	return g
}

// colourItem is the colour glyph it draws in fg, none for a glyph drawn
// by its outline.
func (f *Font) colourItem(it item, fg color.Color) colourGlyph {
	if len(f.colours) == 0 || it.g.face < 0 {
		return colourGlyph{}
	}
	idx := it.g.idx
	if it.shaped {
		idx = it.idx
	}
	if idx == 0 {
		return colourGlyph{}
	}
	return f.colourOf(it.g.face, idx, fg)
}
