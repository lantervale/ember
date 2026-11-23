package fonts

import (
	"math"
	"slices"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/internal/varfont"
	"github.com/lantervale/ember/ui/internal/style"
)

// Intercepts are where the glyphs of text, drawn from 0 on its baseline
// in f with letter points after each character and word after each word
// separator, cross the band from top to bottom below the baseline (y
// down): one stretch of x for each glyph whose outline crosses it, from
// its leftmost crossing to its rightmost, as Skia's getTextIntercepts,
// which Chrome skips an underline's ink by. The glyphs are those drawn,
// shaped with f's features, its face's own small capitals among them.
// The glyph of a character whose ink Chrome leaves whole (SkipsInk) has
// none. A variable font's glyphs are those of the instance f draws.
func (s *Set) Intercepts(f style.Font, text string, letter, word, top, bottom float32, all bool) [][2]float32 {
	f = s.Caps(f, text)
	if runs := f.SmallCaps(text); runs != nil {
		var out [][2]float32
		x := float32(0)
		for _, r := range runs {
			for _, in := range s.Intercepts(r.Font, r.Text, letter, word, top, bottom, all) {
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
	in := intercepts{s: s, text: text, letter: letter, word: word, top: top, bottom: bottom, size: f.Size, all: all,
		of: c, at: c.Instance(f)}
	script := shape.Script(text)
	var run []shape.Glyph
	flush := func() {
		in.run(c, script, f.Features, run)
		run = run[:0]
	}
	for i, r := range text {
		if sc, ok := shape.ScriptOf(r); ok && sc != script {
			flush()
			script = sc
		}
		g, err := c.font.GlyphIndex(&s.buf, r)
		if err != nil {
			continue
		}
		if g == 0 {
			if face, fg := s.fallbackGlyph(c, r); fg != 0 {
				flush()
				//nolint:gosec // a glyph index and a text offset
				in.run(face, script, "", []shape.Glyph{{ID: uint16(fg), Cluster: int32(i), Char: r}})
				continue
			}
			if a, ok := s.spaceAdvance(c, r, f.Size); ok {
				flush()
				in.x += a + letter
				continue
			}
		}
		//nolint:gosec // a glyph index and a text offset
		run = append(run, shape.Glyph{ID: uint16(g), Cluster: int32(i), Char: r})
	}
	flush()
	return in.out
}

// intercepts gathers Intercepts' stretches, run by run of one face.
type intercepts struct {
	s                                  *Set
	text                               string
	letter, word, top, bottom, size, x float32
	all                                bool
	// at is the instance of the glyphs of of, a variable font, nil for
	// its default; the fallback face's are its default's.
	of  *Face
	at  varfont.Coords
	out [][2]float32
}

// run adds the stretches of run, glyphs of c, shaped with features in
// script where c's layout does anything to them, and moves past it.
func (in *intercepts) run(c *Face, script shape.Tag, features string, run []shape.Glyph) {
	if len(run) == 0 {
		return
	}
	s := in.s
	ppem := fixed.I(int(c.upem))
	var at varfont.Coords
	vf := c.Variable()
	if c == in.of {
		at = in.at
	}
	glyphs := slices.Clone(run)
	if p := c.plan(script, features, at); p.Touches(run) {
		glyphs = p.Shape(glyphs)
	} else {
		for i := range glyphs {
			if at != nil {
				glyphs[i].Advance = int32(math.Round(float64(vf.Advance(glyphs[i].ID, at))))
			} else if a, err := c.font.GlyphAdvance(&s.buf, sfnt.GlyphIndex(glyphs[i].ID), ppem, font.HintingNone); err == nil {
				glyphs[i].Advance = int32(a >> 6)
			}
		}
	}
	scale := in.size / c.upem
	last := run[len(run)-1]
	end := int(last.Cluster) + utf8.RuneLen(last.Char)
	for i, g := range glyphs {
		r, _ := utf8.DecodeRuneInString(in.text[g.Cluster:])
		if SkipsInk(r, in.all) {
			dy := float32(g.YOffset) * scale
			if segs, err := c.outline(&s.buf, g.ID, at); err == nil {
				if lo, hi, ok := crossing(segs, scale, in.top+dy, in.bottom+dy); ok {
					dx := in.x + float32(g.XOffset)*scale
					in.out = append(in.out, [2]float32{dx + lo, dx + hi})
				}
			}
		}
		in.x += float32(g.Advance) * scale
		next := end
		if i+1 < len(glyphs) {
			next = int(glyphs[i+1].Cluster)
		}
		if next > int(g.Cluster) {
			in.x += spacing(in.text[g.Cluster:next], in.letter, in.word)
		}
	}
}

// outline is glyph g's outline at instance at of c's variable font, or of
// its default for nil, y down in 26.6 font units as sfnt loads it.
func (c *Face) outline(buf *sfnt.Buffer, g uint16, at varfont.Coords) (sfnt.Segments, error) {
	ppem := fixed.I(int(c.upem))
	if at == nil {
		return c.font.LoadGlyph(buf, sfnt.GlyphIndex(g), ppem, nil)
	}
	points, ends, err := c.Variable().Outline(g, at)
	if err != nil {
		return nil, err
	}
	pt := func(x, y float32) fixed.Point26_6 {
		return fixed.Point26_6{X: fixed.Int26_6(math.Round(float64(x) * 64)), Y: fixed.Int26_6(math.Round(float64(-y) * 64))}
	}
	var segs sfnt.Segments
	varfont.Contours(points, ends, func(x, y float32) {
		segs = append(segs, sfnt.Segment{Op: sfnt.SegmentOpMoveTo, Args: [3]fixed.Point26_6{pt(x, y)}})
	}, func(x, y float32) {
		segs = append(segs, sfnt.Segment{Op: sfnt.SegmentOpLineTo, Args: [3]fixed.Point26_6{pt(x, y)}})
	}, func(cx, cy, x, y float32) {
		segs = append(segs, sfnt.Segment{Op: sfnt.SegmentOpQuadTo, Args: [3]fixed.Point26_6{pt(cx, cy), pt(x, y)}})
	})
	return segs, nil
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
