package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/paint"
)

// frameLists is how the open select lists of the Documents in iframes
// show whole over the page in the window, as Chrome's (decision 0166):
// such a Document's paint leaves its list out, and the window's Document
// draws it over everything once it has drawn, kept inside the window,
// the pointer over it the framed Document's alone.
type frameLists struct {
	// top is the window's area, in the drawing's space, for the window's
	// Document; in is it in a framed Document's own root space.
	top, in geom.Rect
	// shown are the framed Documents whose lists the window's Document
	// draws at the end of this frame.
	shown []*Document
	// rects are where the lists drawn in the last frame lie, in the
	// drawing's space, and owners whose each was.
	rects  []geom.Rect
	owners []*Document
}

// outermost is the Document in the window d shows in, through its
// iframes: d itself when it is not in one.
func (d *Document) outermost() *Document {
	for d.framed != nil {
		d = d.framed.host
	}
	return d
}

// placeLists notes where d's lists may lie this frame, d laid out at
// size: the window's area round the outermost Document.
func (d *Document) placeLists(c *twod.Canvas, size geom.Vec2) {
	z := d.Zoom()
	if d.framed == nil {
		d.lists.top = boundsUnder(c.Transform(), geom.Rect{W: size.X * z, H: size.Y * z})
		return
	}
	xf := c.Transform().Mul(geom.Transform2D{X: geom.V2(z, 0), Y: geom.V2(0, z)})
	d.lists.in = boundsUnder(xf.Inverse(), d.outermost().lists.top)
}

// listsFrame is what d's paint takes of where its lists lie: nothing in
// a window, the window's area in an iframe, the lists left out.
func (d *Document) listsFrame() *geom.Rect {
	if d.framed == nil {
		return nil
	}
	return &d.lists.in
}

// showLists has the outermost Document draw d's open lists, which d's
// paint left out, over its page once it has drawn.
func (d *Document) showLists() {
	if d.framed != nil && d.replay.live.ListsApart() {
		root := d.outermost()
		root.lists.shown = append(root.lists.shown, d)
	}
}

// drawLists draws over d the lists the Documents in its frames left out
// this frame, and notes where they lie.
func (d *Document) drawLists(c *twod.Canvas) {
	l := &d.lists
	l.rects, l.owners = l.rects[:0], l.owners[:0]
	for _, child := range l.shown {
		n := len(l.rects)
		l.rects = paint.DrawLists(c, &child.replay.live, child.lists.in, child.clock, l.rects, child.options()...)
		for range len(l.rects) - n {
			l.owners = append(l.owners, child)
		}
	}
	clear(l.shown)
	l.shown = l.shown[:0]
}

// overlaid reports whether the pointer lies over the list of another
// Document than d, shown over the window's page: the pointer is then
// that one's alone.
func (d *Document) overlaid(c *twod.Canvas) bool {
	root := d.outermost()
	if len(root.lists.rects) == 0 {
		return false
	}
	at := c.Transform().Xform(c.Mouse())
	for i, r := range root.lists.rects {
		if root.lists.owners[i] != d && r.Contains(at) {
			return true
		}
	}
	return false
}

// boundsUnder is the rectangle round r under xf.
func boundsUnder(xf geom.Transform2D, r geom.Rect) geom.Rect {
	lo := xf.Xform(geom.V2(r.X, r.Y))
	hi := lo
	for _, p := range [3]geom.Vec2{{X: r.X + r.W, Y: r.Y}, {X: r.X, Y: r.Y + r.H}, {X: r.X + r.W, Y: r.Y + r.H}} {
		v := xf.Xform(p)
		lo, hi = geom.V2(min(lo.X, v.X), min(lo.Y, v.Y)), geom.V2(max(hi.X, v.X), max(hi.Y, v.Y))
	}
	return geom.Rect{X: lo.X, Y: lo.Y, W: hi.X - lo.X, H: hi.Y - lo.Y}
}
