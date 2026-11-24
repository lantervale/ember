package fonts

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/ui/internal/style"
)

// Ink is where text's glyphs in f's own face draw along its line, text
// of a vertical line in text-orientation o (a horizontal one's is
// OrientSideways): from the first value, past the pen's start, as long as
// the second, glyph after glyph by their advances, kerning left out; a
// cluster standing upright draws down its room, from its top; false when
// the face draws none of it.
func (s *Set) Ink(f style.Font, text string, o style.TextOrientation) (float32, float32, bool) {
	c := s.Face(f)
	if c == nil || c.font == nil || c.upem <= 0 {
		return 0, 0, false
	}
	scale := f.Size * c.Look.adjust() / c.upem
	ppem := fixed.I(int(c.upem))
	var pen, lo, hi float32
	ok := false
	take := func(a, b float32) {
		if !ok {
			lo, hi, ok = a, b, true
		}
		lo, hi = min(lo, a), max(hi, b)
	}
	style.Oriented(text, o, func(part string, upright bool) {
		if upright {
			room, origin, drawn := s.Upright(f, part)
			for _, r := range part {
				if g, err := c.font.GlyphIndex(&s.buf, r); err == nil && g != 0 {
					if b, _, boundsErr := c.font.GlyphBounds(&s.buf, g, ppem, font.HintingNone); boundsErr == nil &&
						b.Max.Y > b.Min.Y {
						take(pen+origin+float32(b.Min.Y)/64*scale, pen+origin+float32(b.Max.Y)/64*scale)
					}
				}
			}
			if drawn {
				pen += room
			}
			return
		}
		for _, r := range part {
			g, err := c.font.GlyphIndex(&s.buf, r)
			if err != nil || g == 0 {
				continue
			}
			b, adv, err := c.font.GlyphBounds(&s.buf, g, ppem, font.HintingNone)
			if err != nil {
				continue
			}
			if b.Max.X > b.Min.X {
				take(pen+float32(b.Min.X)/64*scale, pen+float32(b.Max.X)/64*scale)
			}
			pen += float32(adv) / 64 * scale
		}
	})
	return lo, hi - lo, ok
}
