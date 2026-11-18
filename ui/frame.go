package ui

import (
	"slices"

	"github.com/lantervale/ember/automation"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/internal/inputfeed"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/paint"
)

// Iframe is an iframe element showing child, a Document of the program's
// own making, in its content box (decision 0103): child lays out at the
// box's size, which its @media rules are answered for, is clipped to it
// and drawn where the element stands in its Document's paint order.
// The element is replaced, 300 by 150 with a 2px inset border unless
// CSS, its width and height attributes or frameborder="0" say otherwise.
// A nil child shows nothing in the box.
//
// The pointer is child's only where the element is what its Document
// finds under it, the wheel scrolls child until it can scroll no
// further, then the Document around it, and the keys go to child while
// the element holds the focus. Tab walks from the Document into child
// and out of it again. ember fetches nothing for it: what child holds
// is the program's to build.
func Iframe(child *Document, args ...Arg) *Node {
	n := element(dom.Iframe, args)
	dom.Data(n).Canvas = &control.Canvas{Draw: frameDraw(n), Live: true}
	dom.Data(n).Frame = child
	return n
}

// Frame is the Document the iframe n shows, nil for none and for what
// is no iframe.
func (d *Document) Frame(n *Node) *Document { return frameOf(n) }

// SetFrame has the iframe n show child from the next frame, nil for
// nothing: the Document it showed is let go of. A Document shows in one
// iframe at a time.
func (d *Document) SetFrame(n *Node, child *Document) {
	if n == nil || n.Tag() != dom.Iframe || frameOf(n) == child {
		return
	}
	if was := frameOf(n); was != nil {
		was.unframe(n)
	}
	dom.Data(n).Frame = child
	d.replay.repaint = true
}

// FramePoint is at, a point of d in the space BoxOf answers in, as a
// point of the Document the iframe n shows, in that one's: what its
// ElementAt takes. It is false where at is not in n's content box, and
// while n is not laid out or shows nothing.
func (d *Document) FramePoint(n *Node, at geom.Vec2) (geom.Vec2, bool) {
	r, ok := d.BoxOf(n)
	if !ok || frameOf(n) == nil || !r.Contains(at) {
		return geom.Vec2{}, false
	}
	return at.Sub(geom.V2(r.X, r.Y)).Scale(1 / d.Zoom()), true
}

// FrameShown reports whether the Document the iframe n shows was drawn
// in d's last frame: not while n is out of view, has no box, or d
// itself is not drawn.
func (d *Document) FrameShown(n *Node) bool {
	child := frameOf(n)
	return child != nil && child.framed != nil && child.framed.n == n && child.drawnOnce && d.drawnOnce &&
		child.drawn == d.drawn
}

func frameOf(n *Node) *Document {
	if n == nil || n.IsText() || n.Tag() != dom.Iframe {
		return nil
	}
	child, _ := dom.Data(n).Frame.(*Document)
	return child
}

// framing is where a Document shows inside another: the iframe n of
// host.
type framing struct {
	host *Document
	n    *Node
	// leaving is a Tab that walked out of the Document, 1 past its last
	// target and -1 before its first, for host to take on; asked is a
	// Focus the program gave an element of it, which host follows.
	leaving int
	asked   bool
	// entered is set when a Tab of host walked in: the Document, drawn
	// later in the same frame, leaves that Tab alone.
	entered bool
}

// frameDraw draws what the iframe n shows, its automation IDs under n's
// own.
func frameDraw(n *Node) func(c *twod.Canvas) {
	return func(c *twod.Canvas) {
		child := frameOf(n)
		if child == nil || child.framed == nil || child.framed.n != n {
			return
		}
		path := scope(n)
		if name := scopeName(n); name != "" {
			path = append(path, automation.Slug(name))
		}
		prev := c.SetScope(nil)
		c.SetScope(append(slices.Clone(prev), path...))
		child.Draw(c)
		c.RestoreScope(prev)
	}
}

// unframe lets go of the Document as the one n shows.
func (d *Document) unframe(n *Node) {
	if d.framed == nil || d.framed.n != n {
		return
	}
	d.framed = nil
	d.hovered, d.pressed = nil, nil
	d.moveFocus(func() bool {
		d.focus.Blur()
		return true
	})
	d.ClearSelection()
	if d.owner.owning {
		d.release()
	}
	d.owner = owner{}
}

// syncFrames ties the Documents d's iframes show to d, lets go of those
// it shows no more, and settles the focus between d and them: a frame a
// Tab left hands the focus on, one whose element the program focused
// takes d's focus with it, and one d's focus left loses its own.
func (d *Document) syncFrames() {
	shown := d.frames[:0:0]
	for _, n := range d.controls {
		child := frameOf(n)
		if child == nil {
			continue
		}
		shown = append(shown, n)
		if child.framed == nil || child.framed.host != d || child.framed.n != n {
			if child.framed != nil {
				child.unframe(child.framed.n)
			}
			child.framed = &framing{host: d, n: n}
		}
	}
	for i, n := range d.frames {
		if child := d.framedAt[i]; !slices.Contains(shown, n) || frameOf(n) != child {
			child.unframe(n)
		}
	}
	d.frames, d.framedAt = shown, d.framedAt[:0]
	for _, n := range shown {
		child := frameOf(n)
		d.framedAt = append(d.framedAt, child)
		f := child.framed
		switch {
		case f.leaving != 0:
			back := f.leaving < 0
			f.leaving = 0
			if d.focused() == n {
				d.tab(back)
			}
		case f.asked:
			f.asked = false
			if d.focused() != n {
				d.moveFocus(func() bool { return d.focus.Set(n, false) })
				d.askedOf()
			}
		case d.focused() != n && child.focused() != nil:
			child.moveFocus(func() bool {
				child.focus.Blur()
				return true
			})
		}
	}
}

// askedOf tells the Document around d, if any, that a Focus was given
// inside it.
func (d *Document) askedOf() {
	if d.framed != nil {
		d.framed.asked = true
	}
}

// tab moves d's focus as a Tab does, into the Document of an iframe it
// lands on, and out of d where d is itself in an iframe and the Tab
// would wrap.
func (d *Document) tab(back bool) {
	if none := d.focused() == nil; d.framed != nil && (!d.focus.Tabs() || none && back || !none && d.focus.AtEdge(back)) {
		d.framed.leaving = 1
		if back {
			d.framed.leaving = -1
		}
		d.moveFocus(func() bool {
			d.focus.Blur()
			return true
		})
		return
	}
	d.moveFocus(func() bool { return d.focus.Tab(back) })
	n := d.focused()
	child := frameOf(n)
	if child == nil {
		return
	}
	d.focus.Set(n, false)
	if child.framed != nil && child.focus.Tabs() {
		child.framed.entered = true
		child.moveFocus(func() bool {
			child.focus.Blur()
			return child.focus.Tab(back)
		})
	}
}

// keyed reports whether the keys are d's: always for a Document of a
// window, and for one in an iframe while that element holds the focus of
// a Document that has them.
func (d *Document) keyed() bool {
	f := d.framed
	return f == nil || f.host.focused() == f.n && f.host.keyed()
}

// shielded reports whether the pointer is kept from d this frame: d is
// in an iframe that is not what its Document finds under the pointer,
// and holds nothing a press began.
func (d *Document) shielded() bool {
	f := d.framed
	if f == nil {
		return false
	}
	if len(d.pressed) > 0 || d.dragging != nil || d.drag != nil || d.held != nil || d.sel.held {
		return false
	}
	return f.host.shield || firstOf(f.host.hovered) != f.n
}

// farAway is where a pointer kept from a Document is.
//
//nolint:gochecknoglobals // a constant point
var farAway = geom.V2(-1e9, -1e9)

// idleInput is an input with no key, no button and the pointer away.
func (d *Document) idleInput() *input.State {
	if d.idle == nil {
		d.idle = &input.State{}
		d.idle.SetMouse(farAway)
		inputfeed.Of(d.idle).BeginFrame()
	}
	return d.idle
}

// editing reports whether n, the focused element, takes typed text: a
// text field, or an iframe whose Document's focus is in one.
func (d *Document) editing(n *Node) bool {
	if textOf(d.machine(n)) != nil {
		return true
	}
	child := frameOf(n)
	return child != nil && child.editing(child.focused())
}

// wheels reports whether d, in an iframe, would scroll by a wheel of w
// where the pointer last was over it: the Document around it then leaves
// the wheel to it.
func (d *Document) wheels(w geom.Vec2) bool {
	if child := frameOf(firstOf(d.hovered)); child != nil && child.wheels(w) {
		return true
	}
	for _, b := range scrollContainers(d.replay.hit) {
		if canScroll(paint.BarsOf(b, origin(b)), w) {
			return true
		}
	}
	return false
}

// clearOtherSelections drops the selection of every Document shown with
// d, around it and in its iframes, as one page has one selection.
func (d *Document) clearOtherSelections() {
	root := d
	for root.framed != nil {
		root = root.framed.host
	}
	root.clearSelections(d)
}

func (d *Document) clearSelections(but *Document) {
	if d != but && d.sel.anchor.Node != nil {
		d.ClearSelection()
	}
	for _, child := range d.framedAt {
		if child != nil {
			child.clearSelections(but)
		}
	}
}
