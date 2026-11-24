package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/paint"
)

// A finger on a touchscreen selects as Chrome on one (decision 0222):
// held touchLongPress without moving touchSlop, it selects the word
// under it, in the page's text or a text field's, the field then
// focused; the page's selection then shows a handle at each end, boxes
// of class ember-selection-handle, and a finger that comes down on one
// takes that end to the text under it, where the handle shows above. A
// tap clears the page's selection and its handles go, as do they when the
// pointer or the keys change the selection.
const (
	touchLongPress = 0.5
	touchSlop      = 15
	// handleSize is a selection handle's width and height.
	handleSize = 20
)

// touchSelecting is the finger selecting: down at from since the clock's
// at, still a tap or a long press until it moves touchSlop; or on the
// handle of the end it moves, at grab from the handle's point.
type touchSelecting struct {
	id     int
	down   bool
	moved  bool
	long   bool
	from   geom.Vec2
	at     float32
	handle int
	grab   geom.Vec2
	// handles are the start's and the end's while the page's touch
	// selection shows them, and shownFor the selection they show.
	handles  [2]*Node
	shownFor [2]Position
}

// touchSelection reads the fingers on a touchscreen for the selection.
func (d *Document) touchSelection(c *twod.Canvas, in *input.State) {
	ts := &d.touchSel
	if ts.handles[0] != nil && (d.sel.anchor != ts.shownFor[0] || d.sel.focus != ts.shownFor[1]) {
		d.hideHandles()
	}
	for _, t := range in.Touches() {
		if t.Source != input.Touchscreen {
			continue
		}
		at := c.Local(t.At).Scale(1 / d.Zoom())
		switch {
		case t.Phase == input.TouchBegan && !ts.down:
			*ts = touchSelecting{id: t.ID, down: true, from: at, at: d.clock, handles: ts.handles, shownFor: ts.shownFor}
			ts.handle = d.handleAt(at)
			if ts.handle > 0 {
				ts.grab = at.Sub(d.handlePoint(ts.handle - 1))
			}
		case t.ID != ts.id || !ts.down:
		case t.Phase == input.TouchLifted:
			ts.down = false
			if !ts.long && !ts.moved && ts.handle == 0 {
				d.ClearSelection()
				d.hideHandles()
			}
		case ts.handle > 0:
			d.moveHandle(at)
		case !ts.moved && at.Sub(ts.from).Len() >= touchSlop:
			ts.moved = true
		case !ts.moved && !ts.long && d.clock-ts.at >= touchLongPress:
			ts.long = true
			d.longPress(ts.from)
		}
	}
	d.placeHandles()
}

// longPress selects the word at at: in the text field there, focused,
// or in the page's text.
func (d *Document) longPress(at geom.Vec2) {
	n := d.elementAt(at)
	for _, e := range chain(n) {
		if t := textOf(d.machine(e)); t != nil {
			if !d.moveFocus(func() bool { return d.focus.Set(e, false) }) {
				return
			}
			if i, ok := d.positionAt(e, t, at); ok {
				t.Editor.Select(clickSpan(t, i, 2))
			}
			return
		}
	}
	if n == nil || !selects(dom.Data(n).Style) || d.target(chain(n)) != nil {
		return
	}
	p, ok := d.placeIn(n, at)
	if !ok {
		return
	}
	d.clearOtherSelections()
	from, to := d.wordAt(p)
	d.point(from, to)
	if d.sel.anchor != d.sel.focus {
		d.showHandles()
	}
}

// elementAt is the element under at.
func (d *Document) elementAt(at geom.Vec2) *Node {
	if d.tree == nil {
		return nil
	}
	hit := paint.HitTest(d.inked(), at, paint.Controls(d.machine), paint.Viewport(d.viewport),
		paint.Scale(d.fonts.Scale()))
	for b := hit; b != nil; b = b.Parent {
		if b.Node != nil {
			return dom.Origin(b.Node)
		}
	}
	return nil
}

func (d *Document) showHandles() {
	ts := &d.touchSel
	for i := range ts.handles {
		if ts.handles[i] == nil {
			side := "start"
			if i == 1 {
				side = "end"
			}
			ts.handles[i] = Div(Class("ember-selection-handle"), Attr("data-side", side), Attr("aria-hidden", "true"))
			d.html.Append(ts.handles[i])
		}
	}
	ts.shownFor = [2]Position{d.sel.anchor, d.sel.focus}
}

func (d *Document) hideHandles() {
	ts := &d.touchSel
	for i, h := range ts.handles {
		if h != nil && h.Parent() != nil {
			h.Parent().Remove(h)
		}
		ts.handles[i] = nil
	}
}

// SelectionHandles are the handles at the start and the end of the
// selection a finger made, nil while none shows.
func (d *Document) SelectionHandles() [2]*Node { return d.touchSel.handles }

// handlePoint is where handle i points, the start's at the foot of the
// selection's first character, the end's at that of its last.
func (d *Document) handlePoint(i int) geom.Vec2 {
	r := d.SelectionRects()
	if len(r) == 0 {
		return geom.Vec2{}
	}
	if i == 0 {
		return geom.V2(r[0].X, r[0].Y+r[0].H)
	}
	last := r[len(r)-1]
	return geom.V2(last.X+last.W, last.Y+last.H)
}

// placeHandles puts the handles at the ends of the selection.
func (d *Document) placeHandles() {
	ts := &d.touchSel
	for i, h := range ts.handles {
		if h == nil {
			continue
		}
		p := d.handlePoint(i)
		if i == 0 {
			p.X -= handleSize
		}
		setStyleVar(h, "left", px(p.X))
		setStyleVar(h, "top", px(p.Y))
	}
}

// handleAt is 1 for the start's handle under at, 2 for the end's, 0 for
// none.
func (d *Document) handleAt(at geom.Vec2) int {
	for i, h := range d.touchSel.handles {
		if h != nil && h.Box().Contains(at) {
			return i + 1
		}
	}
	return 0
}

// moveHandle takes the grabbed handle's end of the selection to the text
// under the finger, the handle's point kept where the finger holds it.
func (d *Document) moveHandle(at geom.Vec2) {
	ts := &d.touchSel
	r := d.SelectionRects()
	if len(r) == 0 {
		return
	}
	line := r[0].H
	if ts.handle == 2 {
		line = r[len(r)-1].H
	}
	p, ok := d.positionNear(at.Sub(ts.grab).Sub(geom.V2(0, line/2)), nil)
	if !ok {
		return
	}
	start, end := d.ordered(d.sel.anchor, d.sel.focus)
	if ts.handle == 1 {
		if !before(p, end) {
			return
		}
		d.point(end, p)
	} else {
		if !before(start, p) {
			return
		}
		d.point(start, p)
	}
	ts.shownFor = [2]Position{d.sel.anchor, d.sel.focus}
}
