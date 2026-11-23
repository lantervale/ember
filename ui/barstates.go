package ui

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/paint"
)

// barStates keeps, on the scroll container whose styled bars are under
// the pointer at, the part it is on and the one the left button presses,
// as Chrome does for its parts' :hover and :active: a part pressed stays
// so until the button is let go, and the part under the pointer stays
// the thumb while the thumb is dragged. hit is the box under the pointer.
func (d *Document) barStates(in *input.State, at geom.Vec2, hit *box.Box) {
	if d.tree == nil {
		return
	}
	var on *Node
	part := style.NoBarPart
	for _, b := range scrollContainers(hit) {
		if p := paint.BarsOf(b, origin(b)).PartAt(at); p != style.NoBarPart {
			on, part = b.Node, p
			break
		}
	}
	held := d.barHeld
	if held != nil && dom.Data(held).BarPressed.Part() == style.BarThumb && in.Down(input.MouseLeft) {
		on, part = held, dom.Data(held).BarPressed
	}
	if d.barOn != nil && d.barOn != on {
		d.setBarState(d.barOn, func(p *dom.Pipeline) *style.BarPart { return &p.BarHover }, style.NoBarPart)
	}
	d.barOn = on
	if on != nil {
		d.setBarState(on, func(p *dom.Pipeline) *style.BarPart { return &p.BarHover }, part)
	}
	pressed := func(p *dom.Pipeline) *style.BarPart { return &p.BarPressed }
	switch {
	case in.Pressed(input.MouseLeft) && on != nil:
		if held != nil && held != on {
			d.setBarState(held, pressed, style.NoBarPart)
		}
		d.barHeld = on
		d.setBarState(on, pressed, part)
	case held != nil && !in.Down(input.MouseLeft):
		d.barHeld = nil
		d.setBarState(held, pressed, style.NoBarPart)
	}
}

// setBarState sets the part field of n's pipeline names to part, and has
// css restyle what reads it: n, or for the viewport's bars, whose parts
// body's or the root's rules style, both.
func (d *Document) setBarState(n *Node, field func(*dom.Pipeline) *style.BarPart, part style.BarPart) {
	f := field(dom.Data(n))
	if *f == part {
		return
	}
	*f = part
	dom.Invalidate(n, ":scrollbar")
	if n == d.html && d.body != nil {
		dom.Invalidate(d.body, ":scrollbar")
	}
}
