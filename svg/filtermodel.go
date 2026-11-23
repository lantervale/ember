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
	// Turbulence paints Perlin noise, as Filter Effects' reference code
	// makes it, from Noise.
	Turbulence
	// ConvolveMatrix makes each pixel of its input's around it weighed by
	// Convolve's kernel.
	ConvolveMatrix
	// DisplacementMap moves each pixel of its first input by the colour
	// of its second: by Scale times Channels' values less a half.
	DisplacementMap
	// DiffuseLighting lights its input's alpha as a surface, by Lighting,
	// lit evenly whatever the eye.
	DiffuseLighting
	// SpecularLighting lights it as a shining surface, seen from above.
	SpecularLighting
	// FeImage draws Items, an image or an element, in its region.
	FeImage
)

// EdgeMode is what a kernel reads past its input's edges.
type EdgeMode uint8

const (
	// EdgeDuplicate reads the nearest pixel on the edge.
	EdgeDuplicate EdgeMode = iota
	// EdgeWrap reads the pixel on the other side.
	EdgeWrap
	// EdgeNone reads clear.
	EdgeNone
)

// Noise is a Turbulence's: its frequencies along x and y in the filter's
// space, its octaves, its seed, and whether it is fractal noise, where
// the default sums the noise's magnitudes, and stitched to tile its
// region.
type Noise struct {
	BaseFrequency Point
	Octaves       int
	Seed          float64
	Fractal       bool
	Stitch        bool
}

// Convolve is a ConvolveMatrix's kernel, Order[0] wide and Order[1] high,
// row after row as the file gives it, centred on Target; the weighed sum
// is divided by Divisor, then Bias added.
type Convolve struct {
	Kernel        []float64
	Order, Target [2]int
	Divisor, Bias float64
	Edge          EdgeMode
	// PreserveAlpha keeps each pixel's alpha and weighs its colour alone.
	PreserveAlpha bool
}

// Point3 is a place in a filter's space with a height over the picture.
type Point3 struct{ X, Y, Z float64 }

// LightKind is where a light shines from.
type LightKind uint8

const (
	// DistantLight shines from far off, along Azimuth and Elevation.
	DistantLight LightKind = iota
	// PointLight shines from At every way.
	PointLight
	// SpotLight shines from At towards PointsAt, in a cone.
	SpotLight
)

// Light is a lighting primitive's light source.
type Light struct {
	Kind LightKind
	// Azimuth and Elevation, in degrees, are a distant light's direction.
	Azimuth, Elevation float64
	At, PointsAt       Point3
	// Exponent narrows a spot light's beam about its axis.
	Exponent float64
	// Cone, in degrees, is the most a spot light's ray turns from its
	// axis; 0 for none short of a right angle.
	Cone float64
}

// Lighting is how a lighting primitive lights its surface: the height
// its alpha is raised to, how much light it gives back, the specular
// exponent of a shining one, and its light.
type Lighting struct {
	SurfaceScale float64
	Constant     float64
	Exponent     float64
	Light        Light
}

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
	Noise  Noise
	// Convolve is ConvolveMatrix's kernel.
	Convolve Convolve
	// Scale is how far DisplacementMap moves, in the filter's space, by
	// Channels, red to alpha 0 to 3, along x and y.
	Scale    float64
	Channels [2]uint8
	// Lighting is the lighting primitives', Color their light's colour.
	Lighting Lighting
	// Items are what FeImage draws, in the picture's user space.
	Items []Item
}
