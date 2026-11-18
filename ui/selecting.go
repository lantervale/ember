package ui

import (
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/internal/textedit"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/paint"
)

// The units a drag extends a selection by.
const (
	byCharacter = iota + 1
	byWord
	byBlock
)

// startsSelection reports whether a press on the hovered chain may begin
// a selection, as in Chrome: on an element that takes one, not on a link,
// a button, a control or anything else a click activates or a drag moves,
// which keep the pointer and leave the selection as it is.
func (d *Document) startsSelection() bool {
	n := firstOf(d.hovered)
	if n == nil || d.tree == nil || !selects(dom.Data(n).Style) {
		return false
	}
	return d.target(d.hovered) == nil && d.drag == nil
}

// pressSelection begins a selection where a left press at falls: at the
// place nearest it in the text of the element under the pointer, or of
// the nearest element round it holding some. A second press of a run
// takes the word there and a third the block, as Chrome; extend, Shift's,
// moves the focus of the selection there instead.
func (d *Document) pressSelection(at geom.Vec2, extend bool) {
	if !d.startsSelection() {
		return
	}
	var p Position
	ok := false
	for b := d.tree.Of[firstOf(d.hovered)]; b != nil && !ok; b = b.Parent {
		p, ok = d.positionNear(at, b)
	}
	if !ok {
		return
	}
	d.clearOtherSelections()
	s := &d.sel
	if s.clicks > 0 && s.clicks < byBlock && d.clock-s.clickAt <= doubleClick && at.Sub(s.clickWhere).Len() <= clickSlop {
		s.clicks++
	} else {
		s.clicks = 1
	}
	s.clickAt, s.clickWhere, s.held, s.unit = d.clock, at, true, s.clicks
	switch {
	case extend && s.anchor.Node != nil:
		s.unit = byCharacter
		d.point(s.anchor, p)
	case s.unit == byCharacter:
		d.point(p, p)
	default:
		s.base[0], s.base[1] = d.unitAt(p)
		d.point(s.base[0], s.base[1])
	}
}

// unitAt is the word or the block at p, as the drag's unit has it.
func (d *Document) unitAt(p Position) (Position, Position) {
	if d.sel.unit == byBlock {
		return d.blockAt(p)
	}
	return d.wordAt(p)
}

// dragSelection takes the held selection's focus to the place nearest
// at, anywhere in the text: by characters, or by words or blocks after a
// double or a triple click, what that click took staying selected. Past
// an edge of a container that scrolls the text, it scrolls it.
func (d *Document) dragSelection(at geom.Vec2) {
	s := &d.sel
	d.scrollSelection(at)
	p, ok := d.positionNear(at, nil)
	if !ok || s.anchor.Node == nil {
		return
	}
	if s.unit == byCharacter {
		d.point(s.anchor, p)
		return
	}
	from, to := d.unitAt(p)
	if before(from, s.base[0]) {
		d.point(s.base[1], from)
		return
	}
	if before(to, s.base[1]) {
		to = s.base[1]
	}
	d.point(s.base[0], to)
}

// selectionScroll is how fast a drag past a container's edge scrolls it,
// in points a second for each point past the edge, and the least speed.
const (
	selectionScroll    = 12
	selectionScrollMin = 80
)

// scrollSelection scrolls the innermost container round the selection's
// anchor that the pointer is above or below, and that can still scroll
// that way.
func (d *Document) scrollSelection(at geom.Vec2) {
	s := &d.sel
	dt := min(max(d.clock-s.scrolledAt, 0), 0.05)
	s.scrolledAt = d.clock
	if s.anchor.Node == nil || d.tree == nil {
		return
	}
	for _, b := range scrollContainers(d.tree.Of[s.anchor.Node]) {
		bars := paint.BarsOf(b, origin(b))
		var past float32
		switch {
		case at.Y < bars.View.Y:
			past = at.Y - bars.View.Y
		case at.Y > bars.View.Y+bars.View.H:
			past = at.Y - bars.View.Y - bars.View.H
		default:
			continue
		}
		if !canScroll(bars, geom.V2(0, past)) {
			continue
		}
		speed := past * selectionScroll
		if past > 0 {
			speed = max(speed, selectionScrollMin)
		} else {
			speed = min(speed, -selectionScrollMin)
		}
		off := bars.Scroll
		off.Y = min(max(off.Y+speed*dt, 0), max(bars.Content.Y-bars.View.H, 0))
		d.setScroll(b, off)
		return
	}
}

// overText reports whether at is on the text of hit, the box under the
// pointer, where a press would begin a selection: the text cursor shows
// there when cursor is auto, as in Chrome.
func (d *Document) overText(hit *box.Box, at geom.Vec2) bool {
	if hit == nil || !d.startsSelection() {
		return false
	}
	for _, c := range hit.Children {
		if c.Kind != box.TextRun {
			continue
		}
		r := c.Rect()
		for _, f := range c.Lines {
			if (geom.Rect{X: r.X + f.Rect.X, Y: r.Y + f.Rect.Y, W: f.Rect.W, H: f.Rect.H}).Contains(at) {
				return true
			}
		}
	}
	return false
}

// selectionCursor shows the text cursor over text a press would select.
func (d *Document) selectionCursor(c *twod.Canvas, hit *box.Box, at geom.Vec2) {
	if n := firstOf(d.hovered); n != nil && d.overText(hit, at) && dom.Data(n).Style.Cursor == "" {
		c.ShowCursor(platform.CursorText)
	}
}

// selectionKeys answers the keys of a selection while no field has the
// focus, as Chrome: the platform's copy copies it as plain text, select
// all takes all the Document's text that can be selected, and Shift with
// a moving key moves its focus. Another Document's field holding the
// keyboard keeps them. It reports whether a key moved the selection.
func (d *Document) selectionKeys(c *twod.Canvas, in *input.State, taken *keySet) bool {
	f := inputfeed.Of(in)
	if in.KeyboardCaptured() || !d.takes() || !f.Changed() {
		return false
	}
	m := textedit.Mods{
		Shift:   f.RawDown(int(input.KeyShift)),
		Control: f.RawDown(int(input.KeyControl)),
		Alt:     f.RawDown(int(input.KeyAlt)),
		Super:   f.RawDown(int(input.KeySuper)),
	}
	moved := false
	for _, k := range textedit.Keys {
		if !f.RawPressed(int(k)) && !f.RawRepeated(int(k)) {
			continue
		}
		a, extend := textedit.Resolve(d.platform, k, m)
		took := false
		switch {
		case a == textedit.Copy:
			if s := d.Selection(); !s.Empty() && s.Text != "" {
				if w := c.Window(); w != nil {
					// the clipboard takes plain spaces for the no-break ones, as Chrome's
					w.SetClipboard(strings.ReplaceAll(s.Text, "\u00a0", " "))
				}
				took = true
			}
		case a == textedit.SelectAll:
			took = d.SelectAll()
		case extend && d.sel.anchor != d.sel.focus:
			took = d.moveSelection(a)
			moved = moved || took
		}
		if took {
			taken.add(k)
			f.CaptureKeys(int(k))
		}
	}
	return moved
}

// moveSelection moves the selection's focus as a does, reporting whether
// a is a move it knows: a character or a word left or right, a line up or
// down, to its line's start or end, to the text's.
func (d *Document) moveSelection(a textedit.Action) bool {
	s := &d.sel
	cur, ok := d.pieceOf(s.focus)
	if !ok {
		return false
	}
	to := s.focus
	switch a {
	case textedit.Left, textedit.Right:
		to = d.step(s.focus, a == textedit.Right)
	case textedit.WordLeft, textedit.WordRight:
		to = d.stepWord(s.focus, a == textedit.WordRight)
	case textedit.Up, textedit.Down:
		to = d.stepLine(cur, a == textedit.Down)
	case textedit.LineStart, textedit.LineEnd:
		to = d.lineEdge(cur, a == textedit.LineEnd)
	case textedit.Start, textedit.End:
		first, last, found := d.textIn(d.html)
		if !found {
			return false
		}
		to = first
		if a == textedit.End {
			to = last
		}
	case textedit.None, textedit.PageUp, textedit.PageDown, textedit.SelectAll, textedit.Backspace, textedit.Delete,
		textedit.DeleteWord, textedit.DeleteToLineStart, textedit.Copy, textedit.Cut, textedit.Paste, textedit.Undo,
		textedit.Redo, textedit.Tab, textedit.Newline, textedit.Submit:
		return false
	}
	d.point(s.anchor, to)
	if p, shown := d.pieceOf(s.focus); shown {
		d.revealRect(p.b, p.r)
	}
	return true
}

// pieceOf is the fragment p stands in or beside, x where p is along it.
func (d *Document) pieceOf(p Position) (piece, bool) {
	if p.Node == nil || d.tree == nil {
		return piece{}, false
	}
	b := d.tree.Of[p.Node]
	m := d.mapOf(p.Node)
	if b == nil || m == nil || len(b.Lines) == 0 {
		return piece{}, false
	}
	k := 0
	for k < len(b.Lines)-1 && int(m.frags[k].offs[len(m.frags[k].offs)-1]) < p.Offset {
		k++
	}
	r, f := b.Rect(), b.Lines[k]
	out := piece{n: p.Node, b: b, k: k, r: geom.Rect{X: r.X + f.Rect.X, Y: r.Y + f.Rect.Y, W: f.Rect.W, H: f.Rect.H}}
	out.x = out.r.X + d.edges(m, b, k)[lower(m.frags[k].offs, p.Offset)]
	return out, true
}

// step is the place one character after p, or before it, in the text
// that takes a selection.
func (d *Document) step(p Position, forward bool) Position {
	m := d.mapOf(p.Node)
	if forward && p.Offset < len(m.src) {
		return Position{p.Node, p.Offset + 1}
	}
	if !forward && p.Offset > 0 {
		return Position{p.Node, p.Offset - 1}
	}
	if n := d.neighbour(p.Node, forward); n != nil {
		if forward {
			return Position{n, min(1, len(d.mapOf(n).src))}
		}
		return Position{n, max(len(d.mapOf(n).src)-1, 0)}
	}
	return p
}

// neighbour is the text node after n, or before it, that takes a
// selection.
func (d *Document) neighbour(n *Node, forward bool) *Node {
	runs := d.textRuns()
	step := 1
	if !forward {
		step = -1
	}
	for i := indexOf(runs, n) + step; i >= 0 && i < len(runs); i += step {
		if selects(d.tree.Of[runs[i]].Style) {
			return runs[i]
		}
	}
	return nil
}

func indexOf(runs []*Node, n *Node) int {
	for i, r := range runs {
		if r == n {
			return i
		}
	}
	return -1
}

// stepWord is the end of the word after p, or the start of the one
// before it.
func (d *Document) stepWord(p Position, forward bool) Position {
	for range 4 {
		q := d.step(p, forward)
		if q == p {
			return p
		}
		from, to := d.wordAt(Position{q.Node, q.Offset})
		if !forward {
			from, to = d.wordAt(Position{q.Node, max(q.Offset, 0)})
		}
		src := d.mapOf(q.Node).src
		blank := from.Offset < len(src) && isBlank(src[from.Offset])
		p = to
		if !forward {
			p = from
		}
		if !blank {
			return p
		}
	}
	return p
}

// stepLine is the place nearest cur's on the line under it, or over it;
// the text's end or start when there is none.
func (d *Document) stepLine(cur piece, down bool) Position {
	var best piece
	found := false
	d.pieces(geom.V2(cur.x, cur.r.Y), nil, func(p piece) {
		gap := p.r.Y - (cur.r.Y + cur.r.H)
		if !down {
			gap = cur.r.Y - (p.r.Y + p.r.H)
		}
		if gap < -0.5 {
			return
		}
		was := best.r.Y - (cur.r.Y + cur.r.H)
		if !down {
			was = cur.r.Y - (best.r.Y + best.r.H)
		}
		if !found || gap < was-0.5 ||
			abs(gap-was) <= 0.5 && away(cur.x, p.r.X, p.r.X+p.r.W) < away(cur.x, best.r.X, best.r.X+best.r.W) {
			best, found = p, true
		}
	})
	if !found {
		first, last, _ := d.textIn(d.html)
		if down {
			return last
		}
		return first
	}
	return d.positionIn(best, cur.x)
}

// lineEdge is the start of cur's line, or its end.
func (d *Document) lineEdge(cur piece, end bool) Position {
	best := cur
	d.pieces(geom.V2(cur.x, cur.r.Y), nil, func(p piece) {
		if abs(p.r.Y-cur.r.Y) > 0.5 || container(p.b) != container(cur.b) {
			return
		}
		if end && p.r.X+p.r.W > best.r.X+best.r.W || !end && p.r.X < best.r.X {
			best = p
		}
	})
	offs := d.mapOf(best.n).frags[best.k].offs
	if end {
		return Position{best.n, int(offs[len(offs)-1])}
	}
	return Position{best.n, int(offs[0])}
}

// revealRect scrolls the containers holding b by the least that shows r,
// in the Document's space.
func (d *Document) revealRect(b *box.Box, r geom.Rect) {
	for _, s := range scrollContainers(b) {
		bars := paint.BarsOf(s, origin(s))
		v := bars.View
		off := bars.Scroll
		off.Y += nearest(r.Y, r.H, v.Y, v.H)
		off.Y = min(max(off.Y, 0), max(bars.Content.Y-v.H, 0))
		if off != bars.Scroll {
			d.setScroll(s, off)
			r.Y -= off.Y - bars.Scroll.Y
		}
	}
}
