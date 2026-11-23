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
	mode, parts := shown(e.b).Mode, p.selected(e.b)
	for i := range e.b.Lines {
		f := e.b.Lines[i]
		r := f.Rect
		turn := LineTurn(mode, geom.Rect{X: e.at.X + r.X, Y: e.at.Y + r.Y, W: r.W, H: r.H})
		line := f
		line.Rect = geom.Rect{W: r.H, H: r.W}
		line.Baseline = LineBaseline(mode, f)
		st := shown(e.b)
		if f.Style != nil {
			st = f.Style
		}
		sel, selected := lineSelection(parts, i, LineTurn(mode, r))
		whole := *e.b
		whole.Lines = []box.Fragment{line}
		along := entry{b: &whole, clip: e.clip, pseudo: e.pseudo}
		p.s.SetTransform(saved.Mul(turn))
		p.lineDecorations(along, st, style.Underline|style.Overline, alpha)
		p.lineDecorated, p.lineSelecting = true, true
		p.lineHighlight(st, sel, selected, alpha)
		for _, seg := range p.orientedParts(line, st) {
			one := *e.b
			one.Lines = []box.Fragment{seg.Fragment}
			p.selectPart(seg, sel, selected)
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
			if len(p.lineParts) > 0 {
				p.lineParts[0].Rect = one.Lines[0].Rect
			}
			p.s.SetTransform(saved)
			if k := seg.Rect.W / w; st.Combine && k < 1 {
				// combined text wider than its em is pressed into it
				p.s.SetTransform(saved.Mul(geom.Transform2D{X: geom.V2(k, 0), Y: geom.V2(0, 1),
					Origin: geom.V2(mid.X*(1-k), 0)}))
			}
			p.horizontalText(entry{b: &one, clip: e.clip, pseudo: e.pseudo}, alpha)
		}
		p.lineDecorated, p.lineSelecting = false, false
		p.s.SetTransform(saved.Mul(turn))
		p.lineDecorations(along, st, style.LineThrough, alpha)
	}
}

// selected is the parts of the run b the selection covers.
func (p *painter) selected(b *box.Box) []Selected {
	if p.sel == nil {
		return nil
	}
	return p.sel.Parts(b)
}

// selectPart sets lineParts for seg, a part of a vertical line whose selected part is
// sel where selected: a turned part takes sel, a standing one sel whole where sel
// covers its middle.
func (p *painter) selectPart(seg orientedPart, sel Selected, selected bool) {
	p.lineParts = p.lineParts[:0]
	if mid := seg.Rect.X + seg.Rect.W/2; selected && (!seg.upright || sel.Rect.X <= mid && mid < sel.Rect.X+sel.Rect.W) {
		p.lineParts = append(p.lineParts, Selected{Rect: sel.Rect})
	}
}

// lineHighlight draws the selection's background under sel, the selected part of a
// vertical line in its frame, where selected.
func (p *painter) lineHighlight(st *style.Computed, sel Selected, selected bool, alpha float32) {
	if !selected || !st.Visible {
		return
	}
	if back, _ := p.selectionColors(st, fade(st.Fill(), alpha), alpha); back.A > 0 {
		p.s.Rect(p.snap(sel.Rect), back)
	}
}

// LineTurn is the transform from the frame of a fragment of a vertical line of mode m,
// r on the page, to the page: the horizontal line it is in its frame, x along it from
// its start and y across from the side its glyphs' tops face, turned a quarter
// clockwise, or anticlockwise for sideways-lr.
func LineTurn(m style.WritingMode, r geom.Rect) geom.Transform2D {
	if m.Upward() {
		return geom.Transform2D{X: geom.V2(0, -1), Y: geom.V2(1, 0), Origin: geom.V2(r.X, r.Y+r.H)}
	}
	return geom.Transform2D{X: geom.V2(0, 1), Y: geom.V2(-1, 0), Origin: geom.V2(r.X+r.W, r.Y)}
}

// LineBaseline is where fragment f of a vertical line of mode m has its baseline in
// its line's frame (LineTurn).
func LineBaseline(m style.WritingMode, f box.Fragment) float32 {
	if m.Upward() {
		return f.Baseline - f.Rect.X
	}
	return f.Rect.X + f.Rect.W - f.Baseline
}

// TurnRect is r, a rectangle in the frame of xf's line, on the page.
func TurnRect(xf geom.Transform2D, r geom.Rect) geom.Rect {
	a, b := xf.Xform(geom.V2(r.X, r.Y)), xf.Xform(geom.V2(r.X+r.W, r.Y+r.H))
	return geom.Rect{X: min(a.X, b.X), Y: min(a.Y, b.Y), W: abs(b.X - a.X), H: abs(b.Y - a.Y)}
}

// lineSelection is the part of line i of a vertical run that parts select, in the
// line's frame, turn its transform to the run's space.
func lineSelection(parts []Selected, i int, turn geom.Transform2D) (Selected, bool) {
	for _, s := range parts {
		if s.Line == i {
			return Selected{Line: i, Rect: TurnRect(turn.Inverse(), s.Rect)}, true
		}
	}
	return Selected{}, false
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
	x, at := f.Rect.X, 0
	style.Oriented(f.Text, st.Orientation, func(s string, upright bool) {
		w := p.fonts.Advance(st.Font, s)
		var base float32
		if upright {
			w, base = box.Upright(p.fonts, st.Font, s)
		}
		w += st.Spacing(s)
		part := f
		part.Text, part.Rect.X, part.Rect.W, part.Spacing = s, x, w, partSpacing(f.Spacing, at, len(s))
		room := w
		for _, sp := range part.Spacing {
			room += sp.Space
		}
		if upright && len(part.Spacing) > 0 {
			// the cluster's glyphs moved along the line as text-spacing-trim
			// moves them, over the room it leaves them
			part.Rect.X += part.Spacing[0].Shift
			part.Spacing = nil
		} else {
			part.Rect.W = room
		}
		out = append(out, orientedPart{Fragment: part, upright: upright, baseline: base})
		x += room
		at += len(s)
	})
	return out
}

// partSpacing is the entries of spacing within the n bytes from at of
// its fragment's text, by their bytes in that part.
func partSpacing(spacing []box.Spacing, at, n int) []box.Spacing {
	var out []box.Spacing
	for _, s := range spacing {
		if s.At >= at && s.At < at+n {
			s.At -= at
			out = append(out, s)
		}
	}
	return out
}
