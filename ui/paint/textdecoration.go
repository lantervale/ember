package paint

import (
	"math"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// decorations paints those of the run's text decorations' lines that are
// among lines, across each fragment, the outermost box's first. The lines
// are placed as Chrome places them: auto is a tenth of the decorating
// box's font size thick; an underline lies a gap of half its thickness
// below the baseline; an overline lies on the top of the text's own
// ascent, and a line through is centred two thirds of the decorating
// font's ascent below that top, so both step where the text's size
// changes under one decoration. Thickness and place are whole device
// pixels, at least one.
//
// Each line casts the text's shadows under it, as Chrome draws a
// decoration through the text's shadow looper, and an underline or an
// overline leaves a gap round each glyph whose ink crosses it unless
// text-decoration-skip-ink is none.
func (p *painter) decorations(e entry, st *style.Computed, lines style.DecorationLine, alpha float32) {
	for i := len(st.TextShadows) - 1; i >= 0; i-- {
		sh := st.TextShadows[i]
		sc := sh.Color
		if sh.CurrentColor {
			sc = st.Color
		}
		for _, t := range taps(sh.Blur) {
			tc := fade(fade(sc, alpha), t.weight)
			if tc.A > 0 {
				p.decorationsIn(e, st, lines, alpha, geom.V2(sh.X+t.at.X, sh.Y+t.at.Y), &tc)
			}
		}
	}
	p.decorationsIn(e, st, lines, alpha, geom.Vec2{}, nil)
}

// decorationsIn is decorations moved by off, in ink where it is set.
func (p *painter) decorationsIn(e entry, st *style.Computed, lines style.DecorationLine, alpha float32, off geom.Vec2,
	ink *color.Color) {
	for i := range st.Decorations {
		d := &st.Decorations[i]
		c := fade(d.Color, alpha)
		if (d.Color.A > 0 || p.glyphsOnly) && ink != nil {
			c = *ink
		}
		if d.Line&lines == 0 || c.A <= 0 {
			continue
		}
		px := p.pixels()
		thick := max(p.thickness(d)*px, 1)
		whole := float32(math.Floor(float64(thick)))
		for _, f := range e.b.Lines {
			if f.Rect.W <= 0 {
				continue
			}
			r := geom.Rect{X: e.at.X + f.Rect.X + off.X, Y: e.at.Y + f.Baseline + off.Y, W: f.Rect.W, H: whole / px}
			if d.Line&lines&style.Underline != 0 {
				dy, double := p.underline(st, d, thick)
				p.skipping(st, f, r, dy, thick/px, func() { p.decorationLine(d.Style, r, dy, double, c) })
			}
			if d.Line&lines&(style.Overline|style.LineThrough) == 0 {
				continue
			}
			top := -p.ascent(st.Font, f)
			if d.Line&lines&style.Overline != 0 {
				p.skipping(st, f, r, top-whole/px, thick/px, func() {
					p.decorationLine(d.Style, r, top-whole/px, -(thick+1)/px, c)
				})
			}
			if d.Line&lines&style.LineThrough != 0 {
				second := float32(math.Floor(float64(thick + 1)))
				p.decorationLine(d.Style, r, top+2*p.ascent(d.Font, f)/3-thick/px/2, second/px, c)
			}
		}
	}
}

// thickness is d's lines' thickness in points: a tenth of its font's
// size for auto, the face's own underline's for from-font where it has
// one, else the length given.
func (p *painter) thickness(d *style.Decoration) float32 {
	switch d.Thickness.Kind {
	case style.Fixed:
		t, _ := d.Thickness.Resolve(d.Font.Size)
		return t
	case style.None:
		if m := p.metrics(d.Font); m.Underlined && m.UnderlineThickness > 0 {
			return m.UnderlineThickness
		}
	case style.Auto, style.MinContent, style.MaxContent, style.FitContent, style.Fr, style.Stretch:
	}
	return d.Font.Size / 10
}

func (p *painter) metrics(f style.Font) box.Metrics {
	if p.fonts == nil {
		return box.Metrics{}
	}
	return p.fonts.Metrics(f)
}

// underline is how far below the baseline d's underline of text in st
// begins, thick device pixels thick, as Chrome places it, in
// whole device pixels: on horizontal lines and sideways ones, half its
// thickness below the baseline, or text-underline-offset below it, or
// that below the face's own underline (from-font), or a pixel past that
// below the em box's bottom (under); on vertical lines, which Chrome sets
// on their central baseline, a pixel and the offset past the em box's
// left edge, or as far past its right one for right and the languages
// that underline there. The second result is the step to a double line's second, away
// from the text.
func (p *painter) underline(st *style.Computed, d *style.Decoration, thick float32) (float32, float32) {
	px := p.pixels()
	m := p.metrics(d.Font)
	given, offset := d.Offset.Resolve(d.Font.Size)
	given = lu(given * px)
	if st.Stands() {
		t := p.metrics(st.Font)
		ascent, height := t.Ascent*px, (t.Ascent+t.Descent)*px
		em := lu(d.Font.Size * px)
		mid := lu(height / 2)
		at := floor(mid+float32(math.Floor(float64(em*64/2)))/64+given) + 1
		if d.Position&style.UnderRight != 0 || d.Position&style.UnderLeft == 0 && d.RightInVertical {
			return (2*mid - at - floor(thick) - ascent) / px, -(thick + 1) / px
		}
		return (at - ascent) / px, (thick + 1) / px
	}
	var at float32
	switch {
	case d.Position&style.UnderUnder != 0 && m.EmAscent > 0:
		size := d.Font.Size * px
		at = floor(lu(size)-lu(m.EmAscent*size)+given) + 1
	case d.Position&style.UnderFromFont != 0 && m.Underlined:
		at = float32(math.Round(float64(m.Underline*px + given)))
	case offset:
		at = float32(math.Round(float64(given)))
	default:
		at = max(float32(math.Ceil(float64(thick/2))), 1)
	}
	return at / px, (thick + 1) / px
}

// lu is v in Chrome's layout units, 64ths, to the nearest.
func lu(v float32) float32 { return float32(math.Round(float64(v)*64)) / 64 }

func floor(v float32) float32 { return float32(math.Floor(float64(v))) }

// inkDilation is the most a gap round a glyph's ink reaches past it,
// Chrome's kDecorationClipMaxDilation.
const inkDilation = 13

// skipping draws a line of a decoration of fragment f through draw, which
// lies dy below the baseline of r and is thick points: whole where st's
// text decorations go through ink, else clipped out round each glyph
// that crosses it, by as much as the line is thick, as Chrome clips it.
func (p *painter) skipping(st *style.Computed, f box.Fragment, r geom.Rect, dy, thick float32, draw func()) {
	ink, ok := p.fonts.(box.InkFonts)
	if st.SkipInk == style.SkipInkNone || !ok {
		draw()
		return
	}
	in := ink.Intercepts(st.Font, f.Text, st.LetterGap(), st.WordGap(), dy, dy+thick,
		st.SkipInk == style.SkipInkAll)
	if len(in) == 0 {
		draw()
		return
	}
	dilate := min(thick, inkDilation)
	reach := 4*thick + st.Font.Size
	from := r.X - reach
	for _, g := range in {
		if to := r.X + g[0] - dilate; to > from {
			p.clipped(geom.Rect{X: from, Y: r.Y + dy - reach, W: to - from, H: 2 * reach}, draw)
		}
		from = max(from, r.X+g[1]+dilate)
	}
	if end := r.X + r.W + reach; end > from {
		p.clipped(geom.Rect{X: from, Y: r.Y + dy - reach, W: end - from, H: 2 * reach}, draw)
	}
}

func (p *painter) clipped(r geom.Rect, draw func()) {
	p.s.PushClip(r)
	draw()
	p.s.PopClip()
}

// ascent is font's, or without Fonts the room the fragment leaves above
// its baseline.
func (p *painter) ascent(font style.Font, f box.Fragment) float32 {
	if p.fonts == nil {
		return f.Baseline - f.Rect.Y
	}
	return p.fonts.Metrics(font).Ascent
}

// decorationLine draws one line of a decoration: r moved down by dy, and
// for a double line a second one double further.
func (p *painter) decorationLine(st style.DecorationStyle, r geom.Rect, dy, double float32, c color.Color) {
	r.Y += dy
	x, w := r.X, r.W
	r = p.snap(r)
	r.X, r.W = x, w
	switch st {
	case style.DecorationSolid:
		p.s.Rect(r, c)
	case style.DecorationDouble:
		p.s.Rect(r, c)
		r.Y += double
		r = p.snap(r)
		r.X, r.W = x, w
		p.s.Rect(r, c)
	case style.DecorationDotted:
		p.decorationDashes(r, 1, 1, c)
	case style.DecorationDashed:
		if r.H*p.pixels() < 3 {
			p.decorationDashes(r, 3, 2, c)
		} else {
			p.decorationDashes(r, 2, 1, c)
		}
	}
}

// pixels is how many device pixels make a point, 1 without Scale.
func (p *painter) pixels() float32 {
	if p.scale <= 0 {
		return 1
	}
	return p.scale
}

// decorationDashes draws r as dashes dash thicknesses long with gaps near
// gap thicknesses, as Chrome lays a text line's: as many as fit with one
// flush at each end, the gaps spread evenly; a dotted line over 3 device
// pixels thick is round dots.
func (p *painter) decorationDashes(r geom.Rect, dash, gap float32, c color.Color) {
	round := dash == 1 && r.H*p.pixels() > 3
	dash, gap = dash*r.H, gap*r.H
	n := int(math.Round(float64((r.W + gap) / (dash + gap))))
	if n < 2 {
		p.s.Rect(r, c)
		return
	}
	step := (r.W - dash) / float32(n-1)
	for i := range n {
		piece := geom.Rect{X: r.X + float32(i)*step, Y: r.Y, W: dash, H: r.H}
		if round {
			p.s.Circle(piece.Center(), r.H/2, c)
			continue
		}
		p.s.Rect(piece, c)
	}
}

// decorationReach is how far a run's decorations may lie from its
// baseline.
func decorationReach(st *style.Computed) float32 {
	var d float32
	for i := range st.Decorations {
		dec := &st.Decorations[i]
		t, _ := dec.Thickness.Resolve(dec.Font.Size)
		o, _ := dec.Offset.Resolve(dec.Font.Size)
		d = max(d, dec.Font.Size+2*t+float32(math.Abs(float64(o))))
	}
	return d
}
