package ui

import (
	"math"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/input"
	"github.com/lantervale/ember/platform"
	"github.com/lantervale/ember/twod"
	"github.com/lantervale/ember/ui/internal/box"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
	"github.com/lantervale/ember/ui/paint"
)

// resizeDrag is a box being resized by its resizer: where the press
// was, the box's border-box size then, and the least size the drag
// takes it to: its resizer's corner, or its size then when smaller, as
// Chrome's.
type resizeDrag struct {
	n         *Node
	from      geom.Vec2
	size, min geom.Vec2
}

// pressResizer has a press at at on the resizer of hit or the innermost
// box round it that has one take the pointer, to drag that box's size.
// It reports whether one took it.
func (d *Document) pressResizer(hit *box.Box, at geom.Vec2) bool {
	b := resizerAt(hit, at)
	if b == nil {
		return false
	}
	corner := paint.BarsOf(b, origin(b)).ResizerAt
	d.resizing = &resizeDrag{n: b.Node, from: at, size: b.Size,
		min: geom.V2(min(corner.W, b.Size.X), min(corner.H, b.Size.Y))}
	return true
}

// dragResize follows the pointer at with the box held by its resizer, as
// Chrome resizes it: the size at the press plus how far the pointer moved
// since, mirrored sideways for a resizer on the left, no less than its
// floor, written into the element's style attribute as Chrome writes it,
// width and height in whole pixels of the size box-sizing names, and a
// form control's margins beside them; let go, the drag ends.
func (d *Document) dragResize(in *input.State, at geom.Vec2) {
	g := d.resizing
	b := d.tree.Of[g.n]
	if b == nil || b.Style == nil || !b.Style.Resizable() || !in.Down(input.MouseLeft) {
		d.resizing = nil
		return
	}
	move := at.Sub(g.from)
	if b.Style.Dir == style.RTL {
		move.X = -move.X
	}
	to := geom.V2(max(g.size.X+move.X, g.min.X), max(g.size.Y+move.Y, g.min.Y))
	st, axes := b.Style, b.Style.Resize
	inner := func(n float32, border, padding float32) string {
		if st.BoxSizing != style.BorderBox {
			n -= border + padding
		}
		return px(float32(math.Round(float64(n))))
	}
	control := isFormControl(g.n)
	if axes.Width() && to.X != b.Size.X {
		if control {
			setDeclaration(g.n, "margin-left", px(b.Margin.Left))
			setDeclaration(g.n, "margin-right", px(b.Margin.Right))
		}
		setDeclaration(g.n, "width", inner(to.X, b.Border.Left+b.Border.Right, b.Padding.Left+b.Padding.Right))
	}
	if axes.Height() && to.Y != b.Size.Y {
		if control {
			setDeclaration(g.n, "margin-top", px(b.Margin.Top))
			setDeclaration(g.n, "margin-bottom", px(b.Margin.Bottom))
		}
		setDeclaration(g.n, "height", inner(to.Y, b.Border.Top+b.Border.Bottom, b.Padding.Top+b.Padding.Bottom))
	}
}

// isFormControl reports n a form control, whose margins Chrome writes
// beside the size it resizes.
func isFormControl(n *Node) bool {
	t := n.Tag()
	return t == dom.Input || t == dom.Select || t == dom.Textarea || t == dom.Button
}

// resizerAt is hit or the innermost box round it whose resizer lies
// under at.
func resizerAt(hit *box.Box, at geom.Vec2) *box.Box {
	for b := hit; b != nil; b = b.Parent {
		if b.Kind == box.Element && b.Style != nil && b.Style.Resizable() &&
			paint.BarsOf(b, origin(b)).ResizerAt.Contains(at) {
			return b
		}
	}
	return nil
}

// resizerCursor shows the corner's resize shape over a resizer and while
// one is dragged, as Chrome's, reporting whether it did.
func (d *Document) resizerCursor(c *twod.Canvas, at geom.Vec2, hit *box.Box) bool {
	if d.tree == nil || d.resizing == nil && resizerAt(hit, at) == nil {
		return false
	}
	c.ShowCursor(platform.CursorResizeNWSE)
	return true
}
