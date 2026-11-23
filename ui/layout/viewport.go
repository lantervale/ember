package layout

import (
	"slices"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// viewportOf is the style of the viewport root is laid out in when it
// scrolls: the root's Viewport, or body's when the viewport took body's
// overflow; nil when the viewport doesn't scroll. Its bars are styled
// by body's rules, else the root's, as Chrome's.
func viewportOf(root *dom.Node) *style.Computed {
	st := styleOf(root)
	view, bars := st.Viewport, st.ViewportBars
	for _, c := range root.Children() {
		if c.IsText() || c.Tag() != dom.Body {
			continue
		}
		b := styleOf(c)
		if !st.OverflowPropagated && b.OverflowPropagated && b.Viewport != nil {
			view = b.Viewport
		}
		if b.ViewportBars != nil {
			bars = b.ViewportBars
		}
		break
	}
	if view == nil || bars == nil {
		return view
	}
	return bars.On(view)
}

// viewport lays the root out in a viewport of size whole that scrolls, st its
// style: in what its scrollbars leave of it, the initial containing block,
// an overflow: auto bar taking its room once the page overflows that way
// and the root laid out again, as fitGutter does a box's. It returns the
// bars' room and the size of what scrolls.
func (t *tree) viewport(n *node, whole size, st *style.Computed) (size, size) {
	always := scrollbarGutter(st)
	var auto, content size
	if t.keep != nil {
		auto = t.keep.viewportGutter
	}
	for pass := 0; ; pass++ {
		port := whole
		port.w = fmax(port.w-always.x-auto.w, 0)
		port.h = fmax(port.h-always.y-auto.h, 0)
		t.root(n, spaces{def(port.w), def(port.h)})
		content = scrollable(n, port)
		if pass == 2 || !fitViewport(st, &auto, content, port) {
			break
		}
	}
	n.hoisted = unique(n.hoisted)
	if t.keep != nil {
		t.keep.viewportGutter = auto
	}
	return size{always.x + auto.w, always.y + auto.h}, content
}

// unique is ns without the nodes that come twice, as laying the root out
// again hoists its boxes again.
func unique(ns []*node) []*node {
	out := ns[:0]
	for _, n := range ns {
		if !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// scrollable is the size of what a viewport showing port of the root n
// scrolls: port, or as far right and down as the root's margin box, what
// overflows the root, and the absolute boxes the initial containing block
// placed reach. A fixed box stays in the viewport and adds nothing.
func scrollable(n *node, port size) size {
	u := n.unrounded
	at := u.location
	r := at.x + u.size.w + fmax(u.margin.r, 0)
	b := at.y + u.size.h + fmax(u.margin.b, 0)
	r = fmax(r, at.x+u.border.l+u.overflow.rect.r)
	b = fmax(b, at.y+u.border.t+u.overflow.rect.b)
	for _, h := range n.hoisted {
		if h.st.Position != style.Absolute || h.st.Display == style.DisplayNone {
			continue
		}
		hu := h.unrounded
		r = fmax(r, at.x+hu.location.x+hu.size.w+fmax(hu.margin.r, 0))
		b = fmax(b, at.y+hu.location.y+hu.size.h+fmax(hu.margin.b, 0))
	}
	return size{fmax(port.w, r), fmax(port.h, b)}
}

// fitViewport reports whether the viewport's overflow: auto bars no longer
// suit content in port, and fixes their room in auto.
func fitViewport(st *style.Computed, auto *size, content, port size) bool {
	if st.ScrollbarWidth == 0 {
		return false
	}
	const slack = 0.01
	changed := false
	fit := func(is bool, room *float32, bar float32, overflows bool) {
		if want := overflows && is; want != (*room != 0) {
			*room = 0
			if want {
				*room = bar
			}
			changed = true
		}
	}
	fit(st.Overflow.Height == style.OverflowAuto, &auto.w, st.ScrollbarWidth, content.h > port.h+slack)
	fit(st.Overflow.Width == style.OverflowAuto, &auto.h, st.ScrollbarHeight, content.w > port.w+slack)
	return changed
}

// viewportBox is the box of the viewport, holding the root's: a scroll
// container of size whole, whose Node is the root element, which keeps its
// scroll offset.
func (t *tree) viewportBox(root *box.Box, st *style.Computed, whole, gutter, content size) *box.Box {
	b := &box.Box{}
	if t.keep != nil {
		if t.keep.viewport == nil {
			t.keep.viewport = b
		}
		b = t.keep.viewport
	}
	*b = box.Box{
		Kind: box.Viewport, Node: root.Node, Style: st, Children: b.Children[:0],
		Size:      geom.Vec2{X: whole.w, Y: whole.h},
		Content:   geom.Vec2{X: content.w, Y: content.h},
		Scrollbar: geom.Vec2{X: gutter.w, Y: gutter.h},
	}
	b.Append(root)
	return b
}
