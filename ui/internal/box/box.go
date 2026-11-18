// Package box is the box tree: what layout produces from the styled node
// tree and paint, hit testing and Node.Box read.
package box

import (
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

// Kind is what generated a box.
type Kind uint8

const (
	// Element is an element's principal box.
	Element Kind = iota
	// AnonymousBlock wraps inline content beside blocks.
	AnonymousBlock
	// TextRun is a text node's box; its Lines hold the text.
	TextRun
	// Marker is a summary's disclosure triangle, an inline box at the start
	// of its first line as wide as Chrome's ::marker; Node is the summary.
	Marker
	// Viewport is the viewport of a Document whose viewport scrolls, above
	// the root element's box: a scroll container as big as the viewport,
	// its Style the root's Viewport. Node is the root element, which
	// keeps its scroll offset.
	Viewport
)

// Edges is a resolved margin, border or padding, in points.
type Edges struct{ Top, Right, Bottom, Left float32 }

func (e Edges) Horizontal() float32 { return e.Left + e.Right }
func (e Edges) Vertical() float32   { return e.Top + e.Bottom }

// Fragment is a piece of one line of text, in the box's space.
type Fragment struct {
	Text string
	// Rect is the line box's part the text covers; Baseline is its
	// baseline's y.
	Rect     geom.Rect
	Baseline float32
	// Start is where Text starts in its node's text as laid out, in
	// characters: the text translated and cased, its white space not yet
	// collapsed, so a selection finds a fragment's characters in the node.
	Start int32
	// Ellipsis is set when Text was cut and ends with "…".
	Ellipsis bool
}

// Box is one box. Positions are relative to the parent box's border box,
// before scrolling, as Taffy's are.
type Box struct {
	Kind Kind
	// Node is the element or text node, nil for anonymous boxes.
	Node *dom.Node
	// Style is Node's computed style. An anonymous box's is Initial()
	// with its parent's Text, so nothing paints the parent's background
	// or border twice.
	Style    *style.Computed
	Parent   *Box
	Children []*Box
	// Pos is the border box's top-left in the parent's border box; Size
	// is the border box's size.
	Pos     geom.Vec2
	Size    geom.Vec2
	Margin  Edges
	Border  Edges
	Padding Edges
	// Content is the size of the scrollable overflow, and Scroll the
	// node's dom Scroll offset clamped to it by layout.
	Content geom.Vec2
	Scroll  geom.Vec2
	// Scrollbar is the room the box's scrollbars take along its right and bottom
	// padding edges: overflow: scroll's, and auto's once the content overflows.
	Scrollbar geom.Vec2
	// Sticky is how far position: sticky moved the box from its place in the
	// flow, or how far a fixed box moved to stay put in a scrolled Viewport;
	// Pos includes it.
	Sticky geom.Vec2
	// Escapes is set on an absolutely or fixed positioned box and on each
	// box above it up to its containing block: what it draws lies where
	// that containing block placed it, outside these boxes' own overflow.
	Escapes bool
	// Ink bounds what the box and the boxes under it draw and can be hit
	// at, in the box's space: their border boxes, shadows, outlines and
	// text. Loose is set when no rectangle bounds them where paint tests
	// it: the box, or one under it, is placed from elsewhere, moves
	// without a layout, is transformed, or is one paint looks for wherever
	// it lies. Paint works both out (Tree.Inked) and skips a box that is
	// not Loose where its Ink is out of view or misses the pointer.
	Ink   geom.Rect
	Loose bool
	// Lines is a TextRun's text, laid out.
	Lines []Fragment
	// Fragments is an inline element's border box on each line it spans,
	// in the box's space: the first holds its left edge and the last its
	// right, as Chrome's getClientRects. With Broken they are a block's
	// parts in the columns that cut it, the first holding its top edge and
	// the last its bottom; Pos and Size are then their union's, as Chrome's
	// getBoundingClientRect.
	Fragments []geom.Rect
	Broken    bool
	// Columns is a multi-column container's columns, nil for any other box.
	Columns *Columns
	// Baseline is the first baseline's y, for alignment.
	Baseline float32
	// Wrap is the room a table's captions take inside its box, above and below
	// its grid: its background, border and shadows frame the box less it, as
	// Chrome's table wrapper leaves the captions out.
	Wrap Edges
}

// Columns are a multi-column container's columns, side by side from its
// content box's start: Count of them hold content, each Width wide and
// Height high, Gap apart.
type Columns struct {
	Count              int
	Width, Gap, Height float32
}

// Append adds children to b.
func (b *Box) Append(children ...*Box) {
	for _, c := range children {
		c.Parent = b
	}
	b.Children = append(b.Children, children...)
}

// Rect is the border box in the root's space, scrolling applied.
func (b *Box) Rect() geom.Rect {
	p := b.Pos
	for a := b.Parent; a != nil; a = a.Parent {
		p = p.Add(a.Pos).Sub(a.Scroll)
	}
	return geom.Rect{X: p.X, Y: p.Y, W: b.Size.X, H: b.Size.Y}
}

// ContentRect is the content box, relative to the border box.
func (b *Box) ContentRect() geom.Rect {
	return geom.Rect{
		X: b.Border.Left + b.Padding.Left,
		Y: b.Border.Top + b.Padding.Top,
		W: b.Size.X - b.Border.Horizontal() - b.Padding.Horizontal(),
		H: b.Size.Y - b.Border.Vertical() - b.Padding.Vertical(),
	}
}

// Tree is a laid-out tree and each element's principal box. Root is the
// root element's box, or the Viewport box holding it.
type Tree struct {
	Root *Box
	// Of is each node's box. A retained layout updates one map in place, so it holds the
	// latest layout's alone: Written are the nodes that layout gave a box, or whose kept
	// box isn't where their Placed points, and Dropped those it took out of it.
	Of      map[*dom.Node]*Box
	Written []*dom.Node
	Dropped []*dom.Node
	// Live are the scroll containers, the sticky boxes and the fixed ones a
	// Viewport holds, parents first: what can change
	// without a layout; Applied is each one's scroll and sticky offsets as the last layout
	// or refresh left them.
	Live    []*Box
	Applied [][2]geom.Vec2
	// Inked is set once paint has worked out every box's Ink and Loose
	// for the tree as it stands. A layout's tree starts without; whatever
	// else changes a box's size, its place or the style it is shown with,
	// but a scroll, clears it.
	Inked bool
}

// Metrics are a font's vertical metrics at a size, in points: CSS's
// normal line height is Ascent+Descent+LineGap, and XHeight is how tall
// a lowercase x stands, unrounded.
type Metrics struct{ Ascent, Descent, LineGap, XHeight float32 }

// Fonts measures text for layout, which knows no faces: the root package
// implements it over twod's.
type Fonts interface {
	Metrics(f style.Font) Metrics
	// Advance is the width of text on one line, letter spacing apart.
	Advance(f style.Font, text string) float32
}

// FallbackFonts is Fonts that draws some characters from other faces:
// FallbackMetrics are the largest metrics of the faces besides f's that
// draw text, false when f's own face draws it all. Chrome stretches a
// line to fit them, not the inline box holding them.
type FallbackFonts interface {
	FallbackMetrics(f style.Font, text string) (Metrics, bool)
}

// Images gives layout the natural size of a replaced element's picture,
// such as an img's src, in points; ok false when there is none.
type Images interface {
	ImageSize(url string) (size geom.Vec2, ok bool)
}

// Intrinsic is what a picture asks to be sized by. One of pixels asks
// for its size; one drawn from shapes may ask for a width, a height and
// proportions, any of them or none, and takes the rest from where it is
// shown.
type Intrinsic struct {
	// Width and Height are in points, 0 where the picture asks for none.
	Width, Height float32
	// Ratio is its width over its height, 0 when its sides don't hold
	// each other.
	Ratio float32
}

// Sized is the intrinsic size of a picture of size pixels.
func Sized(size geom.Vec2) Intrinsic {
	i := Intrinsic{Width: size.X, Height: size.Y}
	if size.X > 0 && size.Y > 0 {
		i.Ratio = size.X / size.Y
	}
	return i
}

// Proportions are the picture's as a width and a height, its own where
// it asks for both, zero when it has none.
func (i Intrinsic) Proportions() geom.Vec2 {
	switch {
	case i.Ratio <= 0:
		return geom.Vec2{}
	case i.Width > 0 && i.Height > 0:
		return geom.V2(i.Width, i.Height)
	}
	return geom.V2(i.Ratio, 1)
}

// Natural is the size the picture takes where nothing else sizes it but
// an area it is shown in, CSS's default sizing: its own, a side it
// doesn't ask for from the other through its proportions, else the
// area's; with proportions alone, the largest size of them the area
// holds.
func (i Intrinsic) Natural(area geom.Vec2) geom.Vec2 {
	switch {
	case i.Width > 0 && i.Height > 0:
		return geom.V2(i.Width, i.Height)
	case i.Width > 0 && i.Ratio > 0:
		return geom.V2(i.Width, i.Width/i.Ratio)
	case i.Height > 0 && i.Ratio > 0:
		return geom.V2(i.Height*i.Ratio, i.Height)
	case i.Width > 0:
		return geom.V2(i.Width, area.Y)
	case i.Height > 0:
		return geom.V2(area.X, i.Height)
	case i.Ratio > 0:
		if area.X <= area.Y*i.Ratio {
			return geom.V2(area.X, area.X/i.Ratio)
		}
		return geom.V2(area.Y*i.Ratio, area.Y)
	}
	return area
}

// IntrinsicImages is Images some of whose pictures don't ask for a size
// of pixels: ok is false for a picture that does, sized by ImageSize.
type IntrinsicImages interface {
	ImageIntrinsic(url string) (in Intrinsic, ok bool)
}
