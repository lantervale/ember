package svg

import (
	"image"
	"image/color"
	"math"
)

// Point is a position in a picture's units.
type Point struct{ X, Y float64 }

func (p Point) add(q Point) Point      { return Point{p.X + q.X, p.Y + q.Y} }
func (p Point) sub(q Point) Point      { return Point{p.X - q.X, p.Y - q.Y} }
func (p Point) mul(s float64) Point    { return Point{p.X * s, p.Y * s} }
func (p Point) dot(q Point) float64    { return p.X*q.X + p.Y*q.Y }
func (p Point) cross(q Point) float64  { return p.X*q.Y - p.Y*q.X }
func (p Point) len() float64           { return math.Hypot(p.X, p.Y) }
func (p Point) perp() Point            { return Point{-p.Y, p.X} }
func (p Point) finite() bool           { return !math.IsNaN(p.X+p.Y) && !math.IsInf(p.X+p.Y, 0) }
func lerp(a, b Point, t float64) Point { return Point{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t} }

// Matrix is an affine transform mapping (x, y) to (A·x + C·y + E,
// B·x + D·y + F), as SVG's matrix(a b c d e f).
type Matrix struct{ A, B, C, D, E, F float64 }

// Identity is the transform that moves nothing.
func Identity() Matrix { return Matrix{A: 1, D: 1} }

func translate(x, y float64) Matrix { return Matrix{1, 0, 0, 1, x, y} }
func scale(x, y float64) Matrix     { return Matrix{x, 0, 0, y, 0, 0} }

// Mul returns the transform applying n first, then m.
func (m Matrix) Mul(n Matrix) Matrix {
	return Matrix{
		m.A*n.A + m.C*n.B, m.B*n.A + m.D*n.B,
		m.A*n.C + m.C*n.D, m.B*n.C + m.D*n.D,
		m.A*n.E + m.C*n.F + m.E, m.B*n.E + m.D*n.F + m.F,
	}
}

// Apply is p moved by m.
func (m Matrix) Apply(p Point) Point {
	return Point{m.A*p.X + m.C*p.Y + m.E, m.B*p.X + m.D*p.Y + m.F}
}

// Inverse is the transform undoing m; ok is false when m flattens the
// plane and has none.
func (m Matrix) Inverse() (Matrix, bool) {
	d := m.det()
	if d == 0 || math.IsNaN(d) || math.IsInf(d, 0) {
		return Identity(), false
	}
	return Matrix{
		m.D / d, -m.B / d, -m.C / d, m.A / d,
		(m.C*m.F - m.D*m.E) / d, (m.B*m.E - m.A*m.F) / d,
	}, true
}

func (m Matrix) det() float64 { return m.A*m.D - m.B*m.C }

// stretch is the most m lengthens a line.
func (m Matrix) stretch() float64 {
	a, b := m.A*m.A+m.B*m.B+m.C*m.C+m.D*m.D, m.det()
	return math.Sqrt((a + math.Sqrt(math.Max(a*a-4*b*b, 0))) / 2)
}

func (m Matrix) finite() bool {
	s := m.A + m.B + m.C + m.D + m.E + m.F
	return !math.IsNaN(s) && !math.IsInf(s, 0)
}

// Box is a rectangle: its top-left corner and its size.
type Box struct{ X, Y, W, H float64 }

// Empty reports a box with no area.
func (b Box) Empty() bool { return !(b.W > 0 && b.H > 0) }

// Verb is one step of a Path.
type Verb uint8

const (
	// MoveTo starts a subpath at one point.
	MoveTo Verb = iota
	// LineTo draws a line to one point.
	LineTo
	// QuadTo draws a quadratic Bézier curve: its control point, then its end.
	QuadTo
	// CubicTo draws a cubic Bézier curve: its two control points, then its end.
	CubicTo
	// Close draws a line back to the subpath's start and takes no point.
	Close
)

// Path is an outline: the verbs in order, and the points they take one
// after another. Arcs and the basic shapes are already curves in it.
type Path struct {
	Verbs  []Verb
	Points []Point
}

// FillRule says which parts of a path that crosses itself are inside.
type FillRule uint8

const (
	// NonZero fills where the path winds round a point a number of times
	// other than zero.
	NonZero FillRule = iota
	// EvenOdd fills where a ray from a point crosses the path an odd
	// number of times.
	EvenOdd
)

// Cap is the shape of a stroke's ends.
type Cap uint8

const (
	// ButtCap ends the stroke at the path's end.
	ButtCap Cap = iota
	// RoundCap ends it with a half disc.
	RoundCap
	// SquareCap ends it half the stroke's width further.
	SquareCap
)

// Join is the shape of a stroke's corners.
type Join uint8

const (
	// MiterJoin extends the edges to a point, up to the miter limit.
	MiterJoin Join = iota
	// RoundJoin rounds the corner.
	RoundJoin
	// BevelJoin cuts it.
	BevelJoin
)

// PaintKind is what a Paint paints with.
type PaintKind uint8

const (
	// NoPaint paints nothing.
	NoPaint PaintKind = iota
	// Solid paints one colour.
	Solid
	// GradientPaint paints the Paint's Gradient.
	GradientPaint
	// PatternPaint paints the Paint's Pattern.
	PatternPaint
)

// Paint is what fills a shape or its stroke.
type Paint struct {
	Kind PaintKind
	// Color is a Solid paint's colour, alpha not premultiplied.
	Color color.NRGBA
	// Gradient is a GradientPaint's gradient.
	Gradient *Gradient
	// Pattern is a PatternPaint's tile.
	Pattern *Pattern
	// Opacity, 0–1, multiplies the paint's alpha: fill-opacity or
	// stroke-opacity, and the element's own opacity where it needs no
	// layer.
	Opacity float64
}

// Spread is what a gradient paints outside its vector or circle.
type Spread uint8

const (
	// Pad goes on with the first and last stops' colours.
	Pad Spread = iota
	// Reflect runs the gradient back and forth.
	Reflect
	// Repeat starts it over.
	Repeat
)

// Stop is a colour at a place along a gradient.
type Stop struct {
	// Offset is 0–1, never below the stop before.
	Offset float64
	// Color has the stop's opacity in its alpha, not premultiplied.
	Color color.NRGBA
}

// Gradient is a linear or radial gradient, already given the shape it
// paints: Transform takes the gradient's own space, where Start, End and
// the radii are, to the shape's Path, bounding-box units and
// gradientTransform resolved. Colours are interpolated between stops
// alpha not premultiplied, in encoded sRGB.
type Gradient struct {
	Radial bool
	// Start and End are a linear gradient's vector; of a radial one, the
	// focal point and the centre.
	Start, End Point
	// Radius is a radial gradient's, at End; FocalRadius the radius at
	// Start where the gradient begins.
	Radius, FocalRadius float64
	Stops               []Stop
	Spread              Spread
	Transform           Matrix
}

// Shape is one path with its paints.
type Shape struct {
	Path Path
	// Transform takes the Path to the picture's user space, the one its
	// view box is in.
	Transform Matrix
	Fill      Paint
	FillRule  FillRule
	Stroke    Paint
	// StrokeWidth, Dashes and DashOffset are in the Path's units.
	StrokeWidth float64
	Cap         Cap
	Join        Join
	MiterLimit  float64
	// Dashes are the lengths of dashes and gaps in turn, an even count;
	// none for a solid line.
	Dashes     []float64
	DashOffset float64
	// StrokeFirst paints the stroke under the fill (paint-order).
	StrokeFirst bool
	// NonScalingStroke strokes the path where it shows, its width and
	// dashes in CSS pixels whatever Transform and the view do
	// (vector-effect: non-scaling-stroke).
	NonScalingStroke bool
}

// Pattern is a tile repeated over a shape: its Items drawn in the
// pattern's own space, where Tile is, and cut to it.
type Pattern struct {
	Items []Item
	Tile  Box
	// Transform takes the pattern's space to the shape's Path, units and
	// patternTransform resolved.
	Transform Matrix
}

// Group is items drawn together, apart from what is under them: run
// through its Filters, cut by its Clip, shown through its Mask, then
// through one opacity, and blended with what is under them.
type Group struct {
	Opacity float64
	Items   []Item
	// Filters, when set, make the picture of the items anew, one after
	// another.
	Filters []*Filter
	// Clip, when set, keeps only what it covers.
	Clip *Clip
	// Mask, when set, shows the items as much as it is bright, or opaque.
	Mask *Mask
	// Blend is how the group mixes with what is under it.
	Blend BlendMode
}

// BlendMode is how a picture's colours mix with those it is drawn over:
// mix-blend-mode, and feBlend's mode.
type BlendMode uint8

// The blend modes of CSS compositing.
const (
	BlendNormal BlendMode = iota
	BlendMultiply
	BlendScreen
	BlendOverlay
	BlendDarken
	BlendLighten
	BlendColorDodge
	BlendColorBurn
	BlendHardLight
	BlendSoftLight
	BlendDifference
	BlendExclusion
	BlendHue
	BlendSaturation
	BlendColor
	BlendLuminosity
	// BlendPlusLighter adds the colours, each up to white.
	BlendPlusLighter
)

// Clip is a clipping path: what any of its shapes' fills covers, each
// under its own FillRule, and of that what Clip covers in turn when set.
// Only the shapes' Path, Transform and FillRule count.
type Clip struct {
	Shapes []*Shape
	Clip   *Clip
}

// Mask is items drawn apart, whose luminance (their alpha, for an Alpha
// mask) shows what it masks, and nothing outside its Region.
type Mask struct {
	Items  []Item
	Alpha  bool
	Region *Clip
}

// Image is a picture drawn in a rectangle: Pix, alpha premultiplied, its
// pixels a unit each in the space Transform takes to user space.
type Image struct {
	Pix       *image.RGBA
	Transform Matrix
	Opacity   float64
	// Pixelated draws each pixel as a square, where the default smooths
	// between them.
	Pixelated bool
}

// Item is a Shape, a Group or an Image: one of them is set.
type Item struct {
	Shape *Shape
	Group *Group
	Image *Image
}
