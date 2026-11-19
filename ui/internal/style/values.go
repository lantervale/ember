// Package style is the computed style: what css produces for each node and
// layout and paint read. Every value is computed: em, rem, var() and
// keywords are resolved, and only percentages (as a part of a Length) wait
// for layout's containing block. Lengths are in points (CSS px).
package style

import "unique"

// LengthKind says how a Length sizes.
type LengthKind uint8

const (
	// Auto is the zero value, so an unset Length is auto.
	Auto LengthKind = iota
	// Fixed is Px plus Percent of the reference (calc() of px and %).
	Fixed
	MinContent
	MaxContent
	// FitContent clamps max-content to Px+Percent, fit-content(<length>);
	// with both zero it is plain fit-content.
	FitContent
	// Fr is a grid track's share, Px holding the factor.
	Fr
	// None is max-width/max-height's none.
	None
)

// Length is a computed length-percentage or sizing keyword.
type Length struct {
	Kind    LengthKind
	Px      float32
	Percent float32 // 0–100
	// IsPercent is set when the value had a percentage, 0% included:
	// 0% and 0px differ where the reference size is indefinite.
	IsPercent bool
	// Calc is set for a min(), max() or clamp() over percentages, which
	// no Px and Percent say: Px is then its value against nothing.
	Calc unique.Handle[Calc]
}

// CalcSteps is how many steps a Calc holds.
const CalcSteps = 16

// Calc is a length-percentage with the least or the greatest of others
// in it, as the N steps that work it out on a stack. One value is one
// Handle, so lengths compare as their fields.
type Calc struct {
	Steps [CalcSteps]CalcStep
	N     int
}

// CalcStep is one step of a Calc: CalcPush puts Px and Percent of the
// reference on the stack, CalcMin, CalcMax and CalcSum replace its top N
// values by their least, greatest or sum, and CalcScale multiplies its
// top by Px.
type CalcStep struct {
	Op          CalcOp
	N           int
	Px, Percent float32
}

type CalcOp uint8

const (
	CalcPush CalcOp = iota
	CalcMin
	CalcMax
	CalcSum
	CalcScale
)

// At is the value against ref, the length 100% stands for.
func (c *Calc) At(ref float32) float32 {
	var fixed [CalcSteps]float32
	stack := fixed[:0]
	for _, s := range c.Steps[:c.N] {
		switch s.Op {
		case CalcPush:
			stack = append(stack, s.Px+s.Percent*ref/100)
			continue
		case CalcScale:
			stack[len(stack)-1] *= s.Px
			continue
		case CalcMin, CalcMax, CalcSum:
		}
		of := stack[len(stack)-s.N:]
		v := of[0]
		for _, o := range of[1:] {
			switch s.Op {
			case CalcMin:
				v = min(v, o)
			case CalcMax:
				v = max(v, o)
			case CalcSum:
				v += o
			case CalcPush, CalcScale:
			}
		}
		stack = append(stack[:len(stack)-s.N], v)
	}
	return stack[0]
}

func Px(v float32) Length      { return Length{Kind: Fixed, Px: v} }
func Percent(v float32) Length { return Length{Kind: Fixed, Percent: v} }

// At is a Fixed length against ref, the length 100% stands for.
func (l Length) At(ref float32) float32 {
	if l.Calc != (unique.Handle[Calc]{}) {
		c := l.Calc.Value()
		return c.At(ref)
	}
	return l.Px + l.Percent*ref/100
}

// Resolve is the length against a reference size, ok false when the
// length is not Fixed.
func (l Length) Resolve(ref float32) (float32, bool) {
	if l.Kind != Fixed {
		return 0, false
	}
	return l.At(ref), true
}

// Sides is one value per side, in CSS order.
type Sides[T any] struct{ Top, Right, Bottom, Left T }

// All is the same value on every side.
func All[T any](v T) Sides[T] { return Sides[T]{v, v, v, v} }

// Corners is one value per corner, in CSS order.
type Corners[T any] struct{ TopLeft, TopRight, BottomRight, BottomLeft T }

// Size is a pair of widths and heights, or columns and rows.
type Size[T any] struct{ Width, Height T }
