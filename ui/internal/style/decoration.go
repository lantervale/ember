package style

import "github.com/lantervale/ember/color"

// DecorationLine is text-decoration-line: the lines set, none when zero.
type DecorationLine uint8

const (
	Underline DecorationLine = 1 << iota
	Overline
	LineThrough
)

// DecorationStyle is text-decoration-style.
type DecorationStyle uint8

const (
	DecorationSolid DecorationStyle = iota
	DecorationDouble
	DecorationDotted
	DecorationDashed
)

// TextDecoration is a box's own text-decoration. Thickness is Auto for
// auto and from-font, else a length whose percentage is of the font size
// of the box that draws it.
type TextDecoration struct {
	Line      DecorationLine
	Style     DecorationStyle
	Color     color.Color
	Thickness Length
}

// TextStroke is -webkit-text-stroke: each glyph's outline drawn Width
// points wide, half outside it and half inside, in Color; none when
// Width is 0.
type TextStroke struct {
	Width float32
	Color color.Color
}

// Decoration is a text-decoration drawn on a box's text: its own, or one
// of an ancestor's. Font is the decorating box's, which places the lines
// and gives auto its thickness: the box that set the decoration while the
// text is on its lines, else the block whose lines the text is on.
type Decoration struct {
	TextDecoration
	Font Font
}
