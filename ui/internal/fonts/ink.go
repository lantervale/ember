package fonts

import (
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Intercepts are where the glyphs of text, drawn from 0 on its baseline
// in f with letter points after each character and word after each word
// separator, cross the band from top to bottom below the baseline (y
// down): one stretch of x for each glyph whose outline crosses it, from
// its leftmost crossing to its rightmost, as Skia's getTextIntercepts,
// which Chrome skips an underline's ink by.
func (s *Set) Intercepts(f style.Font, text string, letter, word, top, bottom float32) [][2]float32 {
	if runs := f.SmallCaps(text); runs != nil {
		var out [][2]float32
		x := float32(0)
		for _, r := range runs {
			for _, in := range s.Intercepts(r.Font, r.Text, letter, word, top, bottom) {
				out = append(out, [2]float32{x + in[0], x + in[1]})
			}
			x += s.Advance(r.Font, r.Text) + spacing(r.Text, letter, word)
		}
		return out
	}
	c := s.Face(f)
	if c == nil || c.font == nil {
		return nil
	}
	ppem := fixed.I(int(c.upem))
	scale := f.Size / c.upem
	var out [][2]float32
	x := float32(0)
	prev := sfnt.GlyphIndex(0)
	for _, r := range text {
		face := c
		g, err := c.font.GlyphIndex(&s.buf, r)
		if err != nil {
			continue
		}
		if g == 0 {
			face, g = s.fallbackGlyph(c, r)
		}
		if g == 0 {
			if a, ok := s.spaceAdvance(c, r, f.Size); ok {
				x += a + letter
				prev = 0
				continue
			}
		}
		fscale, fppem := f.Size/face.upem, fixed.I(int(face.upem))
		if prev != 0 && face == c {
			if k, kerr := c.font.Kern(&s.buf, prev, g, ppem, font.HintingNone); kerr == nil {
				x += float32(k) / 64 * scale
			}
		}
		if segs, lerr := face.font.LoadGlyph(&s.buf, g, fppem, nil); lerr == nil {
			if lo, hi, ok := crossing(segs, fscale, top, bottom); ok {
				out = append(out, [2]float32{x + lo, x + hi})
			}
		}
		if a, aerr := face.font.GlyphAdvance(&s.buf, g, fppem, font.HintingNone); aerr == nil {
			x += float32(a) / 64 * fscale
		}
		x += letter
		if style.WordSeparator(r) {
			x += word
		}
		prev = g
		if face != c {
			prev = 0
		}
	}
	return out
}

// fallbackGlyph is r's glyph in the Fallback face when c lacks it and it has it, else
// c's missing glyph.
func (s *Set) fallbackGlyph(c *Face, r rune) (*Face, sfnt.GlyphIndex) {
	if fb := s.Fallback(); fb != nil && fb != c {
		if g, err := fb.font.GlyphIndex(&s.buf, r); err == nil && g != 0 {
			return fb, g
		}
	}
	return c, 0
}

func spacing(text string, letter, word float32) float32 {
	var w float32
	for _, r := range text {
		w += letter
		if style.WordSeparator(r) {
			w += word
		}
	}
	return w
}

// crossing is the stretch of x where the outline segs, in font units
// times scale, cross the band from top to bottom.
func crossing(segs sfnt.Segments, scale, top, bottom float32) (float32, float32, bool) {
	lo, hi, ok := float32(0), float32(0), false
	add := func(a, b geom.Vec2) {
		if a.Y > b.Y {
			a, b = b, a
		}
		if b.Y < top || a.Y > bottom {
			return
		}
		at := func(y float32) float32 {
			if b.Y == a.Y {
				return a.X
			}
			return a.X + (b.X-a.X)*(y-a.Y)/(b.Y-a.Y)
		}
		for _, x := range [2]float32{at(max(a.Y, top)), at(min(b.Y, bottom))} {
			if !ok {
				lo, hi, ok = x, x, true
			}
			lo, hi = min(lo, x), max(hi, x)
		}
	}
	pt := func(p fixed.Point26_6) geom.Vec2 {
		return geom.V2(float32(p.X)/64*scale, float32(p.Y)/64*scale)
	}
	const steps = 8
	var at, start geom.Vec2
	for _, sg := range segs {
		switch sg.Op {
		case sfnt.SegmentOpMoveTo:
			if at != start {
				add(at, start)
			}
			at = pt(sg.Args[0])
			start = at
		case sfnt.SegmentOpLineTo:
			next := pt(sg.Args[0])
			add(at, next)
			at = next
		case sfnt.SegmentOpQuadTo:
			c, end := pt(sg.Args[0]), pt(sg.Args[1])
			for i := 1; i <= steps; i++ {
				t := float32(i) / steps
				next := geom.QuadBezier(at, c, end, t)
				add(geom.QuadBezier(at, c, end, float32(i-1)/steps), next)
			}
			at = end
		case sfnt.SegmentOpCubeTo:
			c1, c2, end := pt(sg.Args[0]), pt(sg.Args[1]), pt(sg.Args[2])
			for i := 1; i <= steps; i++ {
				t := float32(i) / steps
				next := geom.CubicBezier(at, c1, c2, end, t)
				add(geom.CubicBezier(at, c1, c2, end, float32(i-1)/steps), next)
			}
			at = end
		}
	}
	if at != start {
		add(at, start)
	}
	return lo, hi, ok
}
