package style

import (
	"encoding/binary"
	"math"
	"unique"
)

// AnchorFn is an anchor() or anchor-size() in a length: the anchor it
// names, "" for the element's default anchor, and the side or the size of
// it the length stands for.
type AnchorFn struct {
	Size bool
	Name string
	Side AnchorSide
	// Percent is the side as a percentage, Side AnchorPercent.
	Percent float32
}

// AnchorSide is the side an anchor() reads, or the size an anchor-size()
// reads (from AnchorWidth).
type AnchorSide uint8

const (
	AnchorTop AnchorSide = iota
	AnchorRight
	AnchorBottom
	AnchorLeft
	AnchorStart
	AnchorEnd
	AnchorSelfStart
	AnchorSelfEnd
	AnchorInside
	AnchorOutside
	AnchorCenter
	AnchorPercent
	// AnchorAxis is anchor-size() without a size: the axis of the
	// property it is in.
	AnchorAxis
	AnchorWidth
	AnchorHeight
	AnchorBlock
	AnchorInline
	AnchorSelfBlock
	AnchorSelfInline
)

func (fn AnchorFn) append(b []byte) []byte {
	flags := byte(0)
	if fn.Size {
		flags = 1
	}
	b = append(b, flags, byte(fn.Side))
	b = binary.LittleEndian.AppendUint32(b, math.Float32bits(fn.Percent))
	b = binary.LittleEndian.AppendUint16(b, uint16(len(fn.Name))) //nolint:gosec // a name's length
	return append(b, fn.Name...)
}

func readAnchor(code string) (AnchorFn, string) {
	fn := AnchorFn{Size: code[0] == 1, Side: AnchorSide(code[1])}
	fn.Percent = math.Float32frombits(binary.LittleEndian.Uint32([]byte(code[2:6])))
	n := int(binary.LittleEndian.Uint16([]byte(code[6:8])))
	fn.Name = code[8 : 8+n]
	return fn, code[8+n:]
}

// Anchored reports whether l holds an anchor() or an anchor-size().
func (l Length) Anchored() bool {
	if l.Calc == (unique.Handle[Calc]{}) {
		return false
	}
	c := l.Calc.Value()
	return c.Anchored()
}

// AtAnchors is l against ref with its anchor functions resolved by
// anchor, as Calc.AtAnchors; a length without them is At.
func (l Length) AtAnchors(ref float32, anchor func(AnchorFn) (float32, bool)) (float32, bool) {
	if l.Calc == (unique.Handle[Calc]{}) {
		return l.At(ref), true
	}
	c := l.Calc.Value()
	return c.AtAnchors(ref, anchor)
}

// MapAnchors is l with each anchor function what f makes of it.
func (l Length) MapAnchors(f func(AnchorFn) AnchorFn) Length {
	if !l.Anchored() {
		return l
	}
	c := l.Calc.Value()
	l.Calc = unique.Make(c.Mapped(f))
	return l
}

// PositionAnchor is position-anchor: what an element's default anchor is.
type PositionAnchor uint8

const (
	// AnchorNormal is the implicit anchor of a popover its invoker shows,
	// else none.
	AnchorNormal PositionAnchor = iota
	AnchorNone
	// AnchorAuto is the implicit anchor.
	AnchorAuto
	// AnchorNamed is the anchor PositionAnchorName names.
	AnchorNamed
)

// AnchorScope is anchor-scope: which anchor names the element's subtree
// keeps to itself.
type AnchorScope uint8

const (
	ScopeNone AnchorScope = iota
	ScopeAll
	// ScopeNames keeps AnchorScopeNames.
	ScopeNames
)

// AreaSpan is one axis of a position-area: the tracks of the 3×3 grid
// its anchor cuts the containing block into that the area spans, from
// the start (top or left) track 0 to the end one 2.
type AreaSpan struct {
	From, To uint8
}

// PositionArea is position-area: the region of the grid the default
// anchor cuts the containing block into that the element is placed in,
// Set when it is not none. X and Y are the physical spans, from left and
// from top.
type PositionArea struct {
	Set  bool
	X, Y AreaSpan
}

// Flip is a try tactic of position-try-fallbacks.
type Flip uint8

const (
	FlipBlock Flip = 1 << iota
	FlipInline
	FlipStart
)

// PositionTry is one fallback of position-try-fallbacks: an @position-try
// rule's name, the flips applied after it, or a position-area.
type PositionTry struct {
	Name  string
	Flips []Flip
	Area  PositionArea
}

// TryOrder is position-try-order.
type TryOrder uint8

const (
	TryNormal TryOrder = iota
	TryMostWidth
	TryMostHeight
)

// PositionVisibility is position-visibility, a set.
type PositionVisibility uint8

const (
	// VisibleAlways is always: the empty set.
	VisibleAlways PositionVisibility = 0
	// AnchorsVisible hides the element when its default anchor is
	// clipped away or scrolled out of view.
	AnchorsVisible PositionVisibility = 1 << iota
	// NoOverflow hides the element when it overflows its containing block
	// after every fallback was tried.
	NoOverflow
)

// Anchoring is an element's anchor positioning properties.
type Anchoring struct {
	// Names is anchor-name, nil for none; Scope and ScopeNames are
	// anchor-scope.
	Names      []string
	Scope      AnchorScope
	ScopeNames []string
	// Anchor and AnchorName are position-anchor.
	Anchor     PositionAnchor
	AnchorName string
	Area       PositionArea
	// Fallbacks is position-try-fallbacks, Tries the style each gives
	// the element, in order; Order is position-try-order.
	Fallbacks  []PositionTry
	Tries      []*Computed
	Order      TryOrder
	Visibility PositionVisibility
}

func (a *Anchoring) equal(o *Anchoring) bool {
	return sameSlice(a.Names, o.Names) && a.Scope == o.Scope && sameSlice(a.ScopeNames, o.ScopeNames) &&
		a.Anchor == o.Anchor && a.AnchorName == o.AnchorName && a.Area == o.Area &&
		sameSliceFunc(a.Fallbacks, o.Fallbacks, func(x, y *PositionTry) bool {
			return x.Name == y.Name && x.Area == y.Area && sameSlice(x.Flips, y.Flips)
		}) &&
		sameSliceFunc(a.Tries, o.Tries, func(x, y **Computed) bool { return (*x).Equal(*y) }) &&
		a.Order == o.Order && a.Visibility == o.Visibility
}
