package svg

import "image/color"

// Filter is a filter element's effects, the picture of what it filters
// made anew from it by Primitives, the last one's result shown.
type Filter struct {
	// Region is all the filter shows, in the space Transform takes to
	// user space. Filters made of CSS functions have none, and reach as
	// far as their blurs and offsets take the picture.
	Region Box
	// Transform takes the filter's space, the filtered element's user
	// space, where the regions, lengths and offsets are, to the picture's.
	Transform  Matrix
	Primitives []Primitive
}

// Input is what a primitive reads: the picture filtered, its alpha
// alone, or the result of an earlier primitive, by its index.
type Input int

const (
	// SourceGraphic is the picture of what is filtered.
	SourceGraphic Input = -1
	// SourceAlpha is its alpha, black.
	SourceAlpha Input = -2
	// Transparent is nothing: what a filter cannot draw, such as the
	// background, reads as clear.
	Transparent Input = -3
)

// PrimitiveKind is what a filter primitive does.
type PrimitiveKind uint8

const (
	// Clear makes nothing: a primitive this package does not draw.
	Clear PrimitiveKind = iota
	// GaussianBlur blurs its input by StdDev.
	GaussianBlur
	// Offset moves its input by Offset.
	Offset
	// Flood paints Color.
	Flood
	// ColorMatrix multiplies each colour, not premultiplied, by Matrix.
	ColorMatrix
	// ComponentTransfer maps each channel, not premultiplied, through
	// Transfer.
	ComponentTransfer
	// Composite puts its first input with its second by Operator.
	Composite
	// Merge draws its inputs over one another, the first underneath.
	Merge
	// Blend draws its first input over its second through Mode.
	Blend
	// DropShadow draws its input over its shadow: its alpha blurred by
	// StdDev, moved by Offset and painted Color.
	DropShadow
	// Morphology thins its input (erode) or thickens it (Dilate) by Radius.
	Morphology
	// Tile repeats the part of its input within that input's own
	// region.
	Tile
)

// CompositeOperator is how Composite puts two pictures together.
type CompositeOperator uint8

// Porter and Duff's operators, and the arithmetic one of K.
const (
	CompositeOver CompositeOperator = iota
	CompositeIn
	CompositeOut
	CompositeAtop
	CompositeXor
	CompositeLighter
	CompositeArithmetic
)

// TransferKind is the function a channel goes through.
type TransferKind uint8

const (
	// TransferIdentity leaves the channel.
	TransferIdentity TransferKind = iota
	// TransferTable interpolates between the Table's values.
	TransferTable
	// TransferDiscrete steps through them.
	TransferDiscrete
	// TransferLinear is Slope·C + Intercept.
	TransferLinear
	// TransferGamma is Amplitude·C^Exponent + Offset.
	TransferGamma
)

// Transfer is one channel's function in a ComponentTransfer.
type Transfer struct {
	Kind                                         TransferKind
	Table                                        []float64
	Slope, Intercept, Amplitude, Exponent, Shift float64
}

// Primitive is one step of a filter, writing one picture.
type Primitive struct {
	Kind PrimitiveKind
	// In are the pictures it reads: the first and second inputs, or a
	// Merge's in order.
	In []Input
	// Region is where its result may be, in the filter's space.
	Region Box
	// LinearRGB works on colours in linear light, where false works on
	// them as encoded in sRGB.
	LinearRGB bool
	// StdDev is a blur's standard deviation along x and y.
	StdDev Point
	// Offset is how far Offset and DropShadow move.
	Offset Point
	// Color, alpha not premultiplied, is Flood's and DropShadow's.
	Color color.NRGBA
	// Matrix is ColorMatrix's rows for red, green, blue and alpha, each
	// of the four channels then a constant.
	Matrix [20]float64
	// Transfer are ComponentTransfer's functions for red, green, blue
	// and alpha.
	Transfer [4]Transfer
	Operator CompositeOperator
	// K are the arithmetic operator's k1 to k4.
	K    [4]float64
	Mode BlendMode
	// Radius is Morphology's along x and y; Dilate thickens, where the
	// default erodes.
	Radius Point
	Dilate bool
}
