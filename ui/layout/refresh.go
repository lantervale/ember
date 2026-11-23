package layout

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Refresh brings a laid-out tree up to date with what can change without a layout: the
// kept scroll offsets, clamped and copied into Box.Scroll, and the sticky boxes they move,
// with the fixed ones a scrolled viewport holds in place.
// It is for a frame in which no node is dirty and the viewport is the same, visits only
// the tree's Live boxes, and reports whether any of them moved.
func Refresh(t *box.Tree, _ *dom.Node) bool {
	if t == nil || t.Root == nil {
		return false
	}
	moved := false
	for i, b := range t.Live {
		refreshBox(b)
		if now := [2]geom.Vec2{b.Scroll, b.Sticky}; now != t.Applied[i] {
			t.Applied[i] = now
			moved = true
		}
	}
	return moved
}

// collectLive refreshes every box below b, as a fresh layout leaves them, and gathers
// the live ones and their offsets into t.
func collectLive(b *box.Box, t *box.Tree) {
	_, past := scrolledPast(b)
	moves := isSticky(b) || held(b) != nil || past
	if scrollsItself(b) || moves {
		refreshBox(b)
	}
	if scrollsItself(b) && b.Style != nil && isScrollContainer(b.Style) || moves {
		t.Live = append(t.Live, b)
		t.Applied = append(t.Applied, [2]geom.Vec2{b.Scroll, b.Sticky})
	}
	for _, c := range b.Children {
		collectLive(c, t)
	}
}

func isSticky(b *box.Box) bool { return b.Style != nil && b.Style.Position == style.Sticky }

func isFixed(b *box.Box) bool {
	return b.Kind == box.Element && b.Style != nil && b.Style.Position == style.FixedPosition
}

// scrollsItself reports whether b's node's scroll offset is b's own: an
// element's, but for the root's under a viewport that scrolls, where it is
// the viewport's.
func scrollsItself(b *box.Box) bool {
	if b.Node == nil {
		return false
	}
	return b.Kind == box.Viewport || b.Kind == box.Element && (b.Parent == nil || b.Parent.Kind != box.Viewport)
}

// held is the scrolling viewport a fixed box stays put in, nil for any
// other box: one in no such viewport, inside a fixed box, which holds it
// already, or inside a filtered one, its containing block.
func held(b *box.Box) *box.Box {
	if !isFixed(b) {
		return nil
	}
	top := b.Parent
	for ; top != nil && top.Parent != nil; top = top.Parent {
		if isFixed(top) || top.Kind == box.Element && top.Style != nil && top.Style.Filtered() {
			return nil
		}
	}
	if top == nil || top.Kind != box.Viewport {
		return nil
	}
	return top
}

// scrolledPast is how far the elements that scroll between b, absolutely or fixed
// positioned, and its containing block scrolled, which b stays put in as Chrome's, and
// whether there is such an element. The root's scroll is the viewport's, which held
// answers for.
func scrolledPast(b *box.Box) (geom.Vec2, bool) {
	if b.Kind != box.Element || b.Style == nil ||
		b.Style.Position != style.Absolute && b.Style.Position != style.FixedPosition {
		return geom.Vec2{}, false
	}
	var by geom.Vec2
	past := false
	for a := b.Parent; a != nil && a.Parent != nil && a.Kind == box.Element; a = a.Parent {
		st := a.Style
		if st == nil {
			continue
		}
		if st.Filtered() || isFixed(a) || b.Style.Position == style.Absolute && st.Position != style.Static {
			break
		}
		if isScrollContainer(st) {
			by, past = by.Add(a.Scroll), true
		}
	}
	return by, past
}

// refreshBox brings b's scroll offset up to date, and how far it moved as
// a sticky box, as a fixed one in a viewport that scrolled, or as one the elements it
// is placed past scrolled.
func refreshBox(b *box.Box) {
	if scrollsItself(b) {
		b.Scroll = clampScroll(b)
		dom.Data(b.Node).Scroll = b.Scroll
	}
	moved, ok := geom.Vec2{}, false
	if isSticky(b) {
		moved, ok = stickyOffset(b), true
	} else if v := held(b); v != nil {
		moved, ok = v.Scroll, true
	}
	if by, past := scrolledPast(b); past {
		moved, ok = moved.Add(by), true
	}
	if ok {
		b.Pos = b.Pos.Sub(b.Sticky).Add(moved)
		b.Sticky = moved
	}
}

// scrollport is a box's padding box less its scrollbar gutter, in its own space.
func scrollport(b *box.Box) geom.Rect {
	g := b.Scrollbar
	return geom.Rect{
		X: b.Border.Left, Y: b.Border.Top,
		W: b.Size.X - b.Border.Horizontal() - g.X,
		H: b.Size.Y - b.Border.Vertical() - g.Y,
	}
}

// clampScroll is a scroll container's kept offset within its scrollable overflow beyond
// the scrollport; anything else does not scroll.
func clampScroll(b *box.Box) geom.Vec2 {
	if b.Style == nil || !isScrollContainer(b.Style) || b.Style.Display == style.DisplayNone {
		return geom.Vec2{}
	}
	port := scrollport(b)
	s := dom.Data(b.Node).Scroll
	return geom.Vec2{
		X: min(max(s.X, 0), max(b.Content.X-port.W, 0)),
		Y: min(max(s.Y, 0), max(b.Content.Y-port.H, 0)),
	}
}

// flowAt is a box's border box top-left in the root's space as the flow placed it, before
// scrolling and sticky offsets.
func flowAt(b *box.Box) geom.Vec2 {
	var p geom.Vec2
	for x := b; x != nil; x = x.Parent {
		p = p.Add(x.Pos.Sub(x.Sticky))
	}
	return p
}

// stickyOffset is how far a sticky box moves from its place in the flow to keep its insets
// from the edges of its nearest scroll container's scrollport (the root's box when there
// is none), without leaving its parent's content box; a table cell's, the table's, its
// row and row group being no containing block.
func stickyOffset(b *box.Box) geom.Vec2 {
	parent := b.Parent
	if parent == nil {
		return geom.Vec2{}
	}
	sc := parent
	for sc.Parent != nil && (sc.Style == nil || !isScrollContainer(sc.Style)) {
		sc = sc.Parent
	}
	scAt := flowAt(sc)
	p := scrollport(sc)
	port := rect{l: scAt.X + p.X + sc.Scroll.X, t: scAt.Y + p.Y + sc.Scroll.Y}
	port.r, port.b = port.l+p.W, port.t+p.H

	if b.Style != nil && b.Style.TablePart == style.TableCell {
		for parent.Parent != nil && (parent.Style == nil || !parent.Style.Table) {
			parent = parent.Parent
		}
	}
	pAt := flowAt(parent)
	cb := rect{
		l: pAt.X + parent.Border.Left + parent.Padding.Left,
		t: pAt.Y + parent.Border.Top + parent.Padding.Top,
		r: pAt.X + parent.Size.X - parent.Border.Right - parent.Padding.Right,
		b: pAt.Y + parent.Size.Y - parent.Border.Bottom - parent.Padding.Bottom,
	}
	if parent == sc && sc.Parent != nil {
		cb.r = pAt.X + p.X + max(sc.Content.X, p.W) - parent.Padding.Right
		cb.b = pAt.Y + p.Y + max(sc.Content.Y, p.H) - parent.Padding.Bottom
	}
	at := flowAt(b)
	el := rect{l: at.X, t: at.Y, r: at.X + b.Size.X, b: at.Y + b.Size.Y}
	m := b.Margin
	inset := b.Style.Inset
	axis := func(start, end, cbStart, cbEnd, marginStart, marginEnd, portStart,
		portEnd, portLen float32, insetStart, insetEnd style.Length) float32 {
		d := float32(0)
		if v := resolve(insetStart, portLen); some(v) && start < portStart+v {
			d = min(portStart+v-start, max(cbEnd-marginEnd-end, 0))
		}
		if v := resolve(insetEnd, portLen); some(v) && end+d > portEnd-v {
			d = max(portEnd-v-end, min(cbStart+marginStart-start, 0))
		}
		return d
	}
	return geom.Vec2{
		X: axis(el.l, el.r, cb.l, cb.r, m.Left, m.Right, port.l, port.r, p.W, inset.Left, inset.Right),
		Y: axis(el.t, el.b, cb.t, cb.b, m.Top, m.Bottom, port.t, port.b, p.H, inset.Top, inset.Bottom),
	}
}
