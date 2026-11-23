// Package fx is a filter element's effects as a Document runs them on an
// element's picture, for filter: url(#id): the model the svg package
// reads from an svg element of the Document (ui/svg makes one of its
// svg.Filter), in the element's user space, its points from the corner
// of its border box. ui never imports svg, so the model is its own.
package fx

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
)

// Graph is a filter: Primitives, each writing one picture from the
// element's or earlier ones, the last one's shown, nothing past Region.
type Graph struct {
	Region     geom.Rect
	Primitives []Primitive
}

// Input is what a primitive reads: the element's picture, its alpha
// alone, nothing, or an earlier primitive's result by its index.
type Input int

const (
	SourceGraphic Input = -1
	SourceAlpha   Input = -2
	Transparent   Input = -3
)

// Kind is what a primitive does, as svg's primitives of the same names.
type Kind uint8

const (
	Clear Kind = iota
	GaussianBlur
	Offset
	Flood
	ColorMatrix
	ComponentTransfer
	Composite
	Merge
	Blend
	DropShadow
	Morphology
	Tile
)

// Operator is how Composite puts its first input with its second.
type Operator uint8

const (
	Over Operator = iota
	In
	Out
	Atop
	Xor
	Lighter
	Arithmetic
)

// TransferKind is a channel's function in a ComponentTransfer.
type TransferKind uint8

const (
	Identity TransferKind = iota
	Table
	Discrete
	Linear
	Gamma
)

// Transfer is one channel's function.
type Transfer struct {
	Kind                                         TransferKind
	Table                                        []float32
	Slope, Intercept, Amplitude, Exponent, Shift float32
}

// Primitive is one step of a Graph. Lengths are in the element's points.
type Primitive struct {
	Kind   Kind
	In     []Input
	Region geom.Rect
	// LinearRGB works on colours in linear light, else as sRGB encodes them.
	LinearRGB bool
	StdDev    geom.Vec2
	Offset    geom.Vec2
	// Color is Flood's and DropShadow's, not premultiplied.
	Color color.Color
	// Matrix is ColorMatrix's rows for red, green, blue and alpha, each
	// of the four channels then a constant.
	Matrix   [20]float32
	Transfer [4]Transfer
	Operator Operator
	K        [4]float32
	// Mode is Blend's, the order of twod.Mix.
	Mode   uint8
	Radius geom.Vec2
	Dilate bool
}
