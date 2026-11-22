package paint

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// inked works t's boxes' Ink and Loose out unless they stand (Inked).
func inked(t *box.Tree) {
	if !t.Inked {
		blends, backdrops := bound(t.Root)
		t.Grouped = blends || backdrops
		t.Inked = true
	}
}

// bound sets the Ink and Loose of b and of the boxes under it, children
// first: b's own ink, every rectangle paint culls a part of it by, and
// its children's where b's overflow doesn't cut them on both axes, in
// which case nothing under b shows outside it. A box with a loose child
// is loose, so the walk reaches the child. It reports whether an element
// under b mixes, or filters its backdrop, in a group above b; b is a group
// (Group) when it is the one.
func bound(b *box.Box) (bool, bool) {
	var blends, backdrops bool
	st := shown(b)
	r, loose := ownInk(b, st), looseItself(b, st)
	cuts := st.Overflow.Width != style.Visible && st.Overflow.Height != style.Visible &&
		(b.Kind == box.Element || b.Kind == box.Viewport)
	for _, c := range b.Children {
		bl, bd := bound(c)
		blends, backdrops = blends || bl, backdrops || bd
		switch {
		case c.Loose:
			loose = true
		case !cuts:
			r = union(r, geom.Rect{X: c.Ink.X + c.Pos.X, Y: c.Ink.Y + c.Pos.Y, W: c.Ink.W, H: c.Ink.H})
		}
	}
	if b.Kind == box.Element && len(st.Filters) > 0 {
		r = filterInk(r, st.Filters)
	}
	if s, ok := clipShapeOf(b, geom.Vec2{}, st); ok {
		r = intersect(r, s.bounds)
	}
	b.Ink, b.Loose, b.Group = r, loose, false
	if b.Kind != box.Element {
		return blends, backdrops
	}
	if blends && isolates(st) || backdrops && backdropRoot(st) {
		b.Group = true
	}
	blends = blends && !isolates(st) || st.BlendMode != style.BlendNormal
	backdrops = backdrops && !backdropRoot(st) || len(st.BackdropFilters) > 0
	return blends, backdrops
}

// ownInk is what b itself draws and is hit at, in its own space: an
// element's or a marker's border box, fragments and shadows (inkOf) and
// its outline's ring, a text run's lines (textInk).
func ownInk(b *box.Box, st *style.Computed) geom.Rect {
	e := entry{b: b}
	switch b.Kind {
	case box.Element, box.Marker:
		r := inkOf(e, st)
		if outlined(b) {
			r = union(r, grow(r, st.OutlineOffset+st.OutlineWidth))
		}
		return r
	case box.TextRun:
		if len(b.Lines) > 0 {
			return textInk(e, st)
		}
	case box.AnonymousBlock, box.Viewport:
	}
	return geom.Rect{}
}

// looseItself reports a box no Ink of its ancestors can hold: an
// absolutely or fixed positioned one, or one on the way to the block that
// placed it (Escapes), which the clips under that block don't cut
// (placedClip); a sticky one, which a scroll moves; a transformed one,
// drawn in a space the viewport doesn't cull in; one whose overflow cuts
// a single axis, whose content a scroll moves past it; and a select or a
// dialog, whose open list and modal layer paint finds wherever they lie.
func looseItself(b *box.Box, st *style.Computed) bool {
	if b.Escapes || b.Style != nil && b.Style.Position == style.Sticky {
		return true
	}
	if b.Kind != box.Element {
		return false
	}
	if st.Position == style.Absolute || st.Position == style.FixedPosition || len(st.Transform) > 0 ||
		(st.Overflow.Width == style.Visible) != (st.Overflow.Height == style.Visible) {
		return true
	}
	return b.Node != nil && (b.Node.Tag() == dom.Select || b.Node.Tag() == dom.Dialog)
}

// inkSlack is how far a box's Ink is grown where it is tested: it is
// summed from the leaves up and its parts' places from the root down, and
// the two differ by the sums' rounding.
const inkSlack = 1

// sight is what the children of one box can show in or be hit at: the
// view their clip leaves, in the space they are painted in, or a scan's
// point there.
type sight struct {
	// whole walks every box.
	whole bool
	// probing tests point, none when the scan has no point.
	probing, pointless bool
	point              geom.Vec2
	// view is what shows, when bounded.
	view    geom.Rect
	bounded bool
}

// sight is what shows through the clip c under the surface's transform
// now, as culled tests ink against it.
func (p *painter) sight(c *clip) sight {
	if p.whole {
		return sight{whole: true}
	}
	if p.probing {
		s := sight{probing: true, pointless: p.pointless}
		if !p.pointless {
			s.point = p.s.Transform().Inverse().Xform(p.point)
		}
		return s
	}
	view, bounded := p.view(c)
	return sight{view: view, bounded: bounded}
}

// misses reports whether c, its border box's top-left at at, and all it
// holds are out of s: nothing of them would be drawn or hit.
func (s sight) misses(c *box.Box, at geom.Vec2) bool {
	if s.whole || c.Loose {
		return false
	}
	if s.probing && s.pointless {
		return true
	}
	r := grow(geom.Rect{X: at.X + c.Ink.X, Y: at.Y + c.Ink.Y, W: c.Ink.W, H: c.Ink.H}, inkSlack)
	if s.probing {
		return !r.Contains(s.point)
	}
	return s.bounded && !shows(r, s.view)
}
