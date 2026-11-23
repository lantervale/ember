package paint

import (
	"unicode/utf8"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/style"
)

// verticalText paints the fragments of a run of vertical lines: each one as the
// horizontal line it is in its frame, turned a quarter clockwise onto the page, its
// glyphs' tops to the right, or anticlockwise for sideways-lr, their tops to the left.
// The characters standing upright (text-orientation) are drawn unturned, each centred
// on the line in the room it takes along it.
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
		for _, seg := range p.orientedParts(line, st) {
			one := *e.b
			one.Lines = []box.Fragment{seg.Fragment}
			if !seg.upright {
				p.s.SetTransform(saved.Mul(turn))
				p.horizontalText(entry{b: &one, clip: e.clip, pseudo: e.pseudo}, alpha)
				continue
			}
			// the centre of the character's room, on the line's central baseline
			m := p.fonts.Metrics(st.Font)
			mid := turn.Xform(geom.V2(seg.Rect.X+seg.Rect.W/2, line.Baseline-(m.Ascent-m.Descent)/2))
			w := p.fonts.Advance(st.Font, seg.Text) + st.Spacing(seg.Text)
			one.Lines[0].Rect = geom.Rect{X: mid.X - w/2, Y: mid.Y - seg.Rect.W/2, W: w, H: seg.Rect.W}
			one.Lines[0].Baseline = mid.Y + (m.Ascent-m.Descent)/2
			p.s.SetTransform(saved)
			p.horizontalText(entry{b: &one, clip: e.clip, pseudo: e.pseudo}, alpha)
		}
	}
}

// orientedPart is a part of a vertical line's fragment, in the line's frame: its glyphs
// turned with it, or one character standing upright.
type orientedPart struct {
	box.Fragment
	upright bool
}

// orientedParts cuts f, a fragment of a vertical line of style st in the line's frame,
// into its turned parts and its upright characters, each over the room it takes, as
// layout measured them.
func (p *painter) orientedParts(f box.Fragment, st *style.Computed) []orientedPart {
	if !st.Stands() {
		return []orientedPart{{Fragment: f}}
	}
	var out []orientedPart
	x := f.Rect.X
	add := func(s string, upright bool) {
		if s == "" {
			return
		}
		w := p.fonts.Advance(st.Font, s)
		if upright {
			m := p.fonts.Metrics(st.Font)
			w = m.Ascent + m.Descent
		}
		w += st.Spacing(s)
		part := f
		part.Text, part.Rect.X, part.Rect.W = s, x, w
		out = append(out, orientedPart{Fragment: part, upright: upright})
		x += w
	}
	from := 0
	for i, r := range f.Text {
		if style.Upright(r, st.Orientation) {
			add(f.Text[from:i], false)
			n := i + utf8.RuneLen(r)
			add(f.Text[i:n], true)
			from = n
		}
	}
	add(f.Text[from:], false)
	return out
}
