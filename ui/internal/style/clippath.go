package style

// ShapeKind is the basic shape of a clip-path.
type ShapeKind uint8

const (
	// ShapeNone is no shape: the reference box alone, or none when the
	// clip-path names no box either.
	ShapeNone ShapeKind = iota
	ShapeInset
	ShapeCircle
	ShapeEllipse
	ShapePolygon
	ShapePath
)

// ShapeBox is the reference box a clip-path's shape is placed in.
type ShapeBox uint8

const (
	// ShapeBoxUnset is a box not written: the border box.
	ShapeBoxUnset ShapeBox = iota
	ShapeBorderBox
	ShapePaddingBox
	ShapeContentBox
	ShapeMarginBox
	// ShapeFillBox is the content box of a CSS box, and ShapeStrokeBox
	// and ShapeViewBox its border box.
	ShapeFillBox
	ShapeStrokeBox
	ShapeViewBox
)

// RadiusSide is how a circle() or ellipse() radius is measured.
type RadiusSide uint8

const (
	// RadiusClosest is the default: as far as the side of the reference
	// box closest to the centre.
	RadiusClosest RadiusSide = iota
	RadiusFarthest
	// RadiusLength is a length, a percentage of the box's (normalised)
	// diagonal for a circle, of its width or height for an ellipse.
	RadiusLength
)

// ShapeRadius is one radius of a circle() or an ellipse().
type ShapeRadius struct {
	Side   RadiusSide
	Length Length
}

// ClipPath is clip-path: a basic shape placed in a reference box, or
// the box alone. The zero ClipPath is none.
type ClipPath struct {
	Shape ShapeKind
	Box   ShapeBox
	// Inset and Round are inset()'s sides and corners, and those of
	// xywh() and rect(), which compute to it.
	Inset Sides[Length]
	Round Corners[Size[Length]]
	// Radius and At are circle()'s, its radius the first, and
	// ellipse()'s; At is the centre, from the box's top-left, written
	// when AtSet.
	Radius [2]ShapeRadius
	At     [2]Length
	AtSet  bool
	// Points are polygon()'s, each x then y.
	Points [][2]Length
	// Path is path()'s data, every command absolute.
	Path    string
	EvenOdd bool
}

// None reports whether c clips nothing.
func (c *ClipPath) None() bool { return c.Shape == ShapeNone && c.Box == ShapeBoxUnset }

// Equal reports whether c and o are the same clip-path.
func (c *ClipPath) Equal(o *ClipPath) bool {
	return c.Shape == o.Shape && c.Box == o.Box && c.Inset == o.Inset && c.Round == o.Round &&
		c.Radius == o.Radius && c.At == o.At && c.AtSet == o.AtSet && sameSlice(c.Points, o.Points) &&
		c.Path == o.Path && c.EvenOdd == o.EvenOdd
}
