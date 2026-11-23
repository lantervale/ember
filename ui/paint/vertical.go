package paint

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	textshape "github.com/lantervale/ember/internal/shape"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// verticalText paints the fragments of a run of vertical lines: each one as the
// horizontal line it is in its frame, turned a quarter clockwise onto the page, its
// glyphs' tops to the right, or anticlockwise for sideways-lr, their tops to the left.
// The characters standing upright (text-orientation) are drawn unturned, each centred
// on the line in the room it takes along it, and text-combine-upright's text across
// the em it takes, pressed into it when wider. The text decorations go along the
// whole line, the turned parts' and the upright ones' alike, as Chrome draws them.
func (p *painter) verticalText(e entry, alpha float32) {
	saved := p.s.Transform()
	defer p.s.SetTransform(saved)
	up := shown(e.b).Mode.Upward()
	for i := range e.b.Lines {
		f := e.b.Lines[i]
		r := f.Rect
		line, turn := f, geom.Transform2D{X: geom.V2(0, 1), Y: geom.V2(-1, 0),
			Origin: geom.V2(e.at.X+r.X+r.W, e.at.Y+r.Y)}
		line.Rect = geom.Rect{W: r.H, H: r.W}
		line.Baseline = r.X + r.W - f.Baseline
		if up {
			turn = geom.Transform2D{X: geom.V2(0, -1), Y: geom.V2(1, 0), Origin: geom.V2(e.at.X+r.X, e.at.Y+r.Y+r.H)}
			line.Baseline = f.Baseline - r.X
		}
		st := shown(e.b)
		if f.Style != nil {
			st = f.Style
		}
		whole := *e.b
		whole.Lines = []box.Fragment{line}
		along := entry{b: &whole, clip: e.clip, pseudo: e.pseudo}
		p.s.SetTransform(saved.Mul(turn))
		p.lineDecorations(along, st, style.Underline|style.Overline, alpha)
		p.lineDecorated = true
		for _, seg := range p.orientedParts(line, st) {
			one := *e.b
			one.Lines = []box.Fragment{seg.Fragment}
			if !seg.upright {
				p.s.SetTransform(saved.Mul(turn))
				p.horizontalText(entry{b: &one, clip: e.clip, pseudo: e.pseudo}, alpha)
				continue
			}
			// the glyphs centred on the line's central baseline, their room's top where it
			// starts on it; combined text is set across, without vertical alternates
			vst := st
			if !st.Combine {
				vst = p.standing(st)
				one.Lines[0].Style = vst
			}
			m := p.fonts.Metrics(st.Font)
			mid := turn.Xform(geom.V2(seg.Rect.X+seg.Rect.W/2, line.Baseline-(m.Ascent-m.Descent)/2))
			top := mid.Y - seg.Rect.W/2
			w := p.fonts.Advance(vst.Font, seg.Text) + st.Spacing(seg.Text)
			one.Lines[0].Rect = geom.Rect{X: mid.X - w/2, Y: top, W: w, H: seg.Rect.W}
			one.Lines[0].Baseline = top + seg.baseline
			p.s.SetTransform(saved)
			if k := seg.Rect.W / w; st.Combine && k < 1 {
				// combined text wider than its em is pressed into it
				p.s.SetTransform(saved.Mul(geom.Transform2D{X: geom.V2(k, 0), Y: geom.V2(0, 1),
					Origin: geom.V2(mid.X*(1-k), 0)}))
			}
			p.horizontalText(entry{b: &one, clip: e.clip, pseudo: e.pseudo}, alpha)
		}
		p.lineDecorated = false
		p.s.SetTransform(saved.Mul(turn))
		p.lineDecorations(along, st, style.LineThrough, alpha)
	}
}

// lineDecorations draws the lines of st's text decorations along a vertical line,
// in its turned frame, where textLines would draw them on its text.
func (p *painter) lineDecorations(e entry, st *style.Computed, lines style.DecorationLine, alpha float32) {
	if !st.Visible {
		return
	}
	if p.glyphsOnly {
		if lines&style.Underline != 0 {
			white := color.White
			p.decorationsIn(e, st, style.Underline|style.Overline|style.LineThrough, 1, geom.Vec2{}, &white)
		}
		return
	}
	p.decorations(e, st, lines, alpha)
}

// standing is st with the features of glyphs standing in vertical lines (vert) first
// among its own, made once a paint.
func (p *painter) standing(st *style.Computed) *style.Computed {
	if v, ok := p.standings[st]; ok {
		return v
	}
	if p.standings == nil {
		p.standings = map[*style.Computed]*style.Computed{}
	}
	v := *st
	v.Font.Features = textshape.Splice(st.Font.Features, "vert")
	p.standings[st] = &v
	return &v
}

// orientedPart is a part of a vertical line's fragment, in the line's frame: its glyphs
// turned with it, or one cluster standing upright, its baseline that far below the top of
// its room.
type orientedPart struct {
	box.Fragment
	upright  bool
	baseline float32
}

// orientedParts cuts f, a fragment of a vertical line of style st in the line's frame,
// into its turned parts and its upright clusters, each over the room it takes, as layout
// measured them.
func (p *painter) orientedParts(f box.Fragment, st *style.Computed) []orientedPart {
	if st.Combine && st.Mode.Vertical() {
		f.Rect.W = st.Font.Size
		m := p.fonts.Metrics(st.Font)
		return []orientedPart{{Fragment: f, upright: true, baseline: (st.Font.Size + m.Ascent - m.Descent) / 2}}
	}
	if !st.Stands() {
		return []orientedPart{{Fragment: f}}
	}
	var out []orientedPart
	x := f.Rect.X
	style.Oriented(f.Text, st.Orientation, func(s string, upright bool) {
		w := p.fonts.Advance(st.Font, s)
		var base float32
		if upright {
			w, base = box.Upright(p.fonts, st.Font, s)
		}
		w += st.Spacing(s)
		part := f
		part.Text, part.Rect.X, part.Rect.W = s, x, w
		out = append(out, orientedPart{Fragment: part, upright: upright, baseline: base})
		x += w
	})
	return out
}
