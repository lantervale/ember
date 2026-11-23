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
// auto, None for from-font, else a length whose percentage is of the font
// size of the box that draws it.
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

// UnderlinePosition is text-underline-position: the words set, auto
// when zero.
type UnderlinePosition uint8

const (
	// UnderFromFont lies an underline where the face's post table says.
	UnderFromFont UnderlinePosition = 1 << iota
	// UnderUnder lies it under the em box, clear of every descender.
	UnderUnder
	// UnderLeft and UnderRight put the underline of vertical lines on
	// their left or right.
	UnderLeft
	UnderRight
)

// Decoration is a text-decoration drawn on a box's text: its own, or one
// of an ancestor's. Font is the decorating box's, which places the lines
// and gives auto its thickness: the box that set the decoration while the
// text is on its lines, else the block whose lines the text is on. Its
// underline lies where the decorating box's text-underline-offset and
// text-underline-position say, as Chrome takes them, the offset's
// percentage of Font's size; RightInVertical is set where that box's
// language is Japanese or Korean, whose vertical lines Chrome underlines
// on their right unless the position says left.
type Decoration struct {
	TextDecoration
	Font            Font
	Offset          Length
	Position        UnderlinePosition
	RightInVertical bool
}
