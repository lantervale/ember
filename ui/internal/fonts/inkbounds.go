package fonts

import (
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/ui/internal/style"
)

// Ink is where text's glyphs in f's own face draw across: from left of
// the pen's start to left+width, glyph after glyph by their advances,
// kerning left out. ok is false when the face draws none of it.
func (s *Set) Ink(f style.Font, text string) (left, width float32, ok bool) {
	c := s.Face(f)
	if c == nil || c.font == nil || c.upem <= 0 {
		return 0, 0, false
	}
	scale := f.Size * c.Look.adjust() / c.upem
	ppem := fixed.I(int(c.upem))
	var pen, x0, x1 float32
	for _, r := range text {
		g, err := c.font.GlyphIndex(&s.buf, r)
		if err != nil || g == 0 {
			continue
		}
		b, adv, err := c.font.GlyphBounds(&s.buf, g, ppem, font.HintingNone)
		if err != nil {
			continue
		}
		if b.Max.X > b.Min.X {
			lo, hi := pen+float32(b.Min.X)/64*scale, pen+float32(b.Max.X)/64*scale
			if !ok {
				x0, x1, ok = lo, hi, true
			}
			x0, x1 = min(x0, lo), max(x1, hi)
		}
		pen += float32(adv) / 64 * scale
	}
	return x0, x1 - x0, ok
}
