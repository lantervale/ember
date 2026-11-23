package style

// SnapAxes are the axes scroll-snap-type snaps a scroll container on.
type SnapAxes uint8

const (
	SnapX SnapAxes = 1 << iota
	SnapY
)

// SnapAlign is one axis of scroll-snap-align: where a snap area lines up
// with its container's snapport.
type SnapAlign uint8

const (
	SnapNone SnapAlign = iota
	SnapStart
	SnapEnd
	SnapCenter
)

// Overscroll is one axis of overscroll-behavior.
type Overscroll uint8

const (
	OverscrollAuto Overscroll = iota
	// OverscrollContain and OverscrollNone keep a scroll that reaches the
	// box's edge from chaining to the boxes round it.
	OverscrollContain
	OverscrollNone
)

// Snap is how a box scrolls besides its overflow (decision 0163): the axes
// it snaps its scroll on, mandatory or by proximity, and its snapport's
// scroll-padding as a scroll container; where it lines up, its
// scroll-margin and its scroll-snap-stop as a snap area; scroll-behavior,
// and overscroll-behavior on each axis.
type Snap struct {
	Type      SnapAxes
	Mandatory bool
	// AlignX and AlignY are scroll-snap-align's inline and block values,
	// AlignX's start the right under direction rtl.
	AlignX, AlignY SnapAlign
	Always         bool
	// Padding is scroll-padding, auto where unset, a percentage of the
	// scrollport; Margin is scroll-margin.
	Padding                  Sides[Length]
	Margin                   Sides[Length]
	Smooth                   bool
	OverscrollX, OverscrollY Overscroll
}
