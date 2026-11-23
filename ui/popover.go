package ui

import (
	"cmp"
	"slices"
	"strings"

	"github.com/lantervale/ember/ui/control"
	"github.com/lantervale/ember/ui/internal/dom"
)

// popoverKind is what an element's popover attribute makes it.
type popoverKind uint8

const (
	notPopover popoverKind = iota
	autoPopover
	hintPopover
	manualPopover
)

func popoverOf(n *Node) popoverKind {
	if n == nil || n.IsText() {
		return notPopover
	}
	v, ok := n.Attr("popover")
	switch {
	case !ok:
		return notPopover
	case v == "" || strings.EqualFold(v, "auto"):
		return autoPopover
	case strings.EqualFold(v, "hint"):
		return hintPopover
	}
	return manualPopover
}

// popover is an open popover: the element that showed it, its implicit
// anchor, if any.
type popover struct {
	kind   popoverKind
	source *Node
}

// ShowPopover shows the popover n in the top layer, as HTML's
// showPopover({source}), emitting Toggled: source, if any, is its implicit anchor, which
// position-anchor's normal and auto name. Showing an auto popover hides
// the auto and hint popovers open that are none of its ancestors, a hint
// popover the other hints.
func (d *Document) ShowPopover(n, source *Node) {
	kind := popoverOf(n)
	if kind == notPopover || d.PopoverOpen(n) {
		return
	}
	if kind != manualPopover {
		keep := d.popoverAncestor(n, source)
		for i := len(d.popovers) - 1; i >= 0; i-- {
			p := d.popovers[i]
			if p == keep || kind == hintPopover && d.openPopovers[p].kind == autoPopover {
				break
			}
			d.HidePopover(p)
		}
		d.popovers = append(d.popovers, n)
	}
	if d.openPopovers == nil {
		d.openPopovers = map[*Node]*popover{}
	}
	d.openPopovers[n] = &popover{kind: kind, source: source}
	dom.Data(n).TopLayer = control.NextLayer()
	n.SetState(dom.PopoverOpen, true)
	d.events = append(d.events, Toggled{n})
}

// HidePopover hides the popover n, and the auto and hint popovers shown
// after it.
func (d *Document) HidePopover(n *Node) {
	if !d.PopoverOpen(n) {
		return
	}
	if i := slices.Index(d.popovers, n); i >= 0 {
		for len(d.popovers) > i+1 {
			d.HidePopover(d.popovers[len(d.popovers)-1])
		}
		d.popovers = d.popovers[:i]
	}
	delete(d.openPopovers, n)
	dom.Data(n).TopLayer = 0
	n.SetState(dom.PopoverOpen, false)
	d.events = append(d.events, Toggled{n})
}

// TogglePopover shows the popover n when it is hidden, else hides it.
func (d *Document) TogglePopover(n, source *Node) {
	if d.PopoverOpen(n) {
		d.HidePopover(n)
		return
	}
	d.ShowPopover(n, source)
}

// PopoverOpen reports whether the popover n is shown.
func (d *Document) PopoverOpen(n *Node) bool {
	_, ok := d.openPopovers[n]
	return ok
}

// popoverAncestor is the newest open auto or hint popover n, shown by
// source, belongs to: one holding n or source, or one source belongs to.
func (d *Document) popoverAncestor(n, source *Node) *Node {
	for i := len(d.popovers) - 1; i >= 0; i-- {
		p := d.popovers[i]
		if p == n {
			continue
		}
		if holds(p, n) || source != nil && holds(p, source) {
			return p
		}
	}
	return nil
}

// implicitAnchor is the element that showed the open popover n.
func (d *Document) implicitAnchor(n *Node) *Node {
	if p := d.openPopovers[n]; p != nil {
		return p.source
	}
	return nil
}

// popoverClick is a click on n: a button with popovertarget shows, hides
// or toggles, as its popovertargetaction says, the popover of that id,
// itself its implicit anchor.
func (d *Document) popoverClick(n *Node) {
	id, ok := n.Attr("popovertarget")
	if !ok || disabledField(n) {
		return
	}
	target := d.ByID(id)
	if popoverOf(target) == notPopover {
		return
	}
	action, _ := n.Attr("popovertargetaction")
	switch strings.ToLower(action) {
	case "show":
		d.ShowPopover(target, n)
	case "hide":
		d.HidePopover(target)
	default:
		d.TogglePopover(target, n)
	}
}

// lightDismiss hides, on a press at hovered (the element pressed and those
// above it), the auto and hint popovers it is outside of: those after
// the newest one holding it or shown by it.
func (d *Document) lightDismiss(hovered []*Node) {
	if len(d.popovers) == 0 {
		return
	}
	var at *Node
	if len(hovered) > 0 {
		at = hovered[0]
	}
	keep := -1
	for i, p := range d.popovers {
		if at != nil && (holds(p, at) || d.openPopovers[p].source != nil && holds(d.openPopovers[p].source, at)) {
			keep = i
		}
	}
	for len(d.popovers) > keep+1 {
		d.HidePopover(d.popovers[keep+1])
	}
}

// backFromPopover hides the newest auto or hint popover on Escape,
// reporting whether there was one.
func (d *Document) backFromPopover() bool {
	if len(d.popovers) == 0 {
		return false
	}
	d.HidePopover(d.popovers[len(d.popovers)-1])
	return true
}

// OpenPopovers are the popovers shown, in the order they joined the top
// layer, the highest last.
func (d *Document) OpenPopovers() []*Node {
	out := make([]*Node, 0, len(d.openPopovers))
	for n := range d.openPopovers {
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b *Node) int {
		return cmp.Compare(dom.Data(a).TopLayer, dom.Data(b).TopLayer)
	})
	return out
}

// PopoverSource is the element that showed the popover n, its implicit
// anchor, nil for none or when n is not shown.
func (d *Document) PopoverSource(n *Node) *Node { return d.implicitAnchor(n) }
