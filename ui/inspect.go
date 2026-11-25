package ui

import (
	"math"
	"strconv"
	"strings"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/dom"
)

// MatchedRules is what the cascade found for the element n at the last
// frame, the strongest first: the rules of the sheets that match it,
// ember's own among them, its style attribute and its presentational
// attributes, each with its sheet, where it opens there, the @media
// rules and the layer it stands in, and its declarations, those a
// stronger one overrides marked. It is for tools that show a screen's
// styles; it works the cascade of n out again, so ask it for the
// element looked at, never for a tree every frame. Empty for a text
// node and for an element not yet styled.
func (d *Document) MatchedRules(n *Node) []css.Match {
	d.take()
	return d.trans.Matched(n)
}

// ComputedStyle is every property of the element n after the cascade,
// by name, its custom properties last: the value its winning
// declaration gives it with var() replaced and lengths in px, its
// parent's where it inherits, the initial value otherwise. Keywords,
// colours and percentages stay as written: what layout made of them is
// in BoxesOf. As MatchedRules, it is for tools, and costs the cascade
// of n and of its ancestors. A change of n or of an element above it
// since the last frame, a class on the root redefining a custom
// property, is seen at once, as a browser's getComputedStyle sees it.
func (d *Document) ComputedStyle(n *Node) []css.Property {
	d.restyleFor(n)
	return d.trans.Computed(n)
}

// ComputedValue is one property of ComputedStyle, false for a name that
// is no longhand of the subset or custom property of n. A property that
// does not inherit costs the cascade of n alone.
//
// grid-template-columns and grid-template-rows of a grid container laid
// out are its tracks as the last layout sized them, implicit ones too, in
// px with the names its track list gives their lines, as Chrome's
// getComputedStyle resolves them.
func (d *Document) ComputedValue(n *Node, name string) (string, bool) {
	d.restyleFor(n)
	if name == "grid-template-columns" || name == "grid-template-rows" {
		if v, ok := d.resolvedTracks(n, name == "grid-template-columns"); ok {
			return v, true
		}
	}
	return d.trans.ComputedValue(n, name)
}

// resolvedTracks is the used value of a laid out grid container's columns or rows; false
// for any other element, and for a subgrid's axis.
func (d *Document) resolvedTracks(n *Node, columns bool) (string, bool) {
	if d.tree == nil {
		return "", false
	}
	b := d.tree.Of[n]
	if b == nil || b.Grid == nil || b.Style == nil {
		return "", false
	}
	axis := b.Grid.Rows
	if columns {
		if b.Style.Subgrid.Width {
			return "", false
		}
		axis = b.Grid.Columns
	} else if b.Style.Subgrid.Height {
		return "", false
	}
	if len(axis.Sizes) == 0 {
		return "none", true
	}
	var parts []string
	names := func(line int) {
		if i := line - axis.Implicit; i >= 0 && i < len(axis.Names) && len(axis.Names[i]) > 0 {
			parts = append(parts, "["+strings.Join(axis.Names[i], " ")+"]")
		}
	}
	for i, size := range axis.Sizes {
		names(i)
		parts = append(parts, trackPixels(size))
	}
	names(len(axis.Sizes))
	return strings.Join(parts, " "), true
}

// trackPixels writes a track's size as Chrome does: in its 1/64 px layout units, to six
// significant digits.
func trackPixels(size float32) string {
	v := math.Floor(float64(size)*64+0.01) / 64
	s := strconv.FormatFloat(v, 'g', 6, 64)
	if strings.Contains(s, "e") {
		s = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return s + "px"
}

// restyleFor works the styles out again before the frame does when n,
// or an element above it, which n inherits from, changed since they last
// were.
func (d *Document) restyleFor(n *Node) {
	d.take()
	for a := n; a != nil; a = a.Parent() {
		if dom.Data(a).Dirty&(dom.DirtyStyle|dom.DirtyTree|dom.DirtyState) != 0 {
			d.style()
			return
		}
	}
}

// Boxes are the four boxes of an element, each inside the one before.
type Boxes struct {
	Margin, Border, Padding, Content geom.Rect
}

// BoxesOf is n's margin, border, padding and content boxes in the space
// BoxOf answers in, Content being BoxOf's own; false while n isn't laid
// out or isn't shown.
func (d *Document) BoxesOf(n *Node) (Boxes, bool) {
	content, ok := d.BoxOf(n)
	if !ok {
		return Boxes{}, false
	}
	b := d.tree.Of[n]
	border := n.Box()
	padding := geom.Rect{X: border.X + b.Border.Left, Y: border.Y + b.Border.Top,
		W: border.W - b.Border.Horizontal(), H: border.H - b.Border.Vertical()}
	margin := geom.Rect{X: border.X - b.Margin.Left, Y: border.Y - b.Margin.Top,
		W: border.W + b.Margin.Horizontal(), H: border.H + b.Margin.Vertical()}
	return Boxes{Margin: d.toWindow(margin), Border: d.toWindow(border), Padding: d.toWindow(padding),
		Content: content}, true
}
