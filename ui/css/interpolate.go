package css

import (
	"math"
	"slices"
	"unique"

	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/ui/internal/style"
)

// animKind is how a longhand takes part in transitions and animations.
// Every longhand of props declares one; the zero value is none declared.
type animKind uint8

const (
	undeclared animKind = iota
	// still takes no part: a keyframe drops it with a warning, a
	// transition never runs on it.
	still
	// discrete flips halfway through an interval; no transition runs on it.
	discrete
	// interpolated moves between two values on Style, where layout and
	// inheritance read it.
	interpolated
	// painted moves between two values in Shown alone, repainted.
	painted
)

// anim is how a longhand animates. lerp writes its value at t between a's
// and b's into dst, and reports false where the two don't interpolate:
// it then flips halfway, and no transition runs between them. equal
// compares the two values. current is the CurrentColorFields bit of a
// colour that may be currentColor.
type anim struct {
	kind    animKind
	lerp    func(dst, a, b *style.Computed, t float32) bool
	equal   func(a, b *style.Computed) bool
	current style.CurrentColorFields
}

// moves reports whether the longhand interpolates.
func (a anim) moves() bool { return a.kind == interpolated || a.kind == painted }

// interp is how two values of one type interpolate: lerp's false where
// they don't. A nil lerp flips halfway.
type interp[T any] struct {
	lerp  func(a, b T, t float32) (T, bool)
	equal func(a, b T) bool
}

// flips is the interpolation of a value that flips halfway.
func flips[T any]() interp[T] { return interp[T]{} }

// on is h for the field get reads.
func (h interp[T]) on(get func(*style.Computed) *T) anim {
	if h.lerp == nil {
		return anim{kind: discrete}
	}
	return anim{
		kind: interpolated,
		lerp: func(dst, a, b *style.Computed, t float32) bool {
			v, ok := h.lerp(*get(a), *get(b), t)
			if ok {
				*get(dst) = v
			}
			return ok
		},
		equal: func(a, b *style.Computed) bool { return h.equal(*get(a), *get(b)) },
	}
}

func same[T comparable](a, b T) bool { return a == b }

func lerpf(a, b, t float32) float32 { return a + (b-a)*t }

func clampf(v, lo, hi float32) float32 { return min(max(v, lo), hi) }

// numbers interpolates numbers, clamped to [0, hi] as CSS does past the
// ends of an easing.
func numbers(hi float32) interp[float32] {
	return interp[float32]{
		lerp:  func(a, b, t float32) (float32, bool) { return clampf(lerpf(a, b, t), 0, hi), true },
		equal: same[float32],
	}
}

// integers interpolates as numbers rounded to the nearest, halves up.
func integers(lo, hi int) interp[int] {
	return interp[int]{
		lerp: func(a, b int, t float32) (int, bool) {
			v := int(math.Floor(float64(lerpf(float32(a), float32(b), t)) + 0.5))
			return min(max(v, lo), hi), true
		},
		equal: same[int],
	}
}

// counts is column-count: integers from 1, auto (0) flipping.
//
//nolint:gochecknoglobals // an interpolation, fixed
var counts = interp[int]{
	lerp: func(a, b int, t float32) (int, bool) {
		if a == 0 || b == 0 {
			return a, a == b
		}
		return integers(1, math.MaxInt).lerp(a, b, t)
	},
	equal: same[int],
}

// widths are border, outline and column-rule widths, in whole points as
// Chrome snaps them: a width under one point and above none is one.
//
//nolint:gochecknoglobals // an interpolation, fixed
var widths = interp[float32]{
	lerp: func(a, b, t float32) (float32, bool) {
		v := max(lerpf(a, b, t), 0)
		if v > 0 && v < 1 {
			return 1, true
		}
		return float32(math.Floor(float64(v))), true
	},
	equal: same[float32],
}

// offsets is outline-offset, in whole points towards zero, as Chrome
// keeps it.
//
//nolint:gochecknoglobals // an interpolation, fixed
var offsets = interp[float32]{
	lerp:  func(a, b, t float32) (float32, bool) { return float32(math.Trunc(float64(lerpf(a, b, t)))), true },
	equal: same[float32],
}

//nolint:gochecknoglobals // an interpolation, fixed
var colors = interp[color.Color]{
	lerp:  func(a, b color.Color, t float32) (color.Color, bool) { return lerpColor(a, b, t), true },
	equal: same[color.Color],
}

// lerpColor interpolates in premultiplied sRGB, as Chrome does for
// colours given as hex, rgb() or names.
func lerpColor(a, b color.Color, t float32) color.Color {
	sa, sb := a.SRGB(), b.SRGB()
	alpha := clampf(lerpf(a.A, b.A, t), 0, 1)
	var out [3]float32
	if alpha > 0 {
		for i := range out {
			out[i] = clampf(lerpf(sa[i]*a.A, sb[i]*b.A, t)/alpha, 0, 1)
		}
	}
	c := color.FromSRGB(out)
	c.A = alpha
	return c
}

// lengths interpolates length-percentages by computed value: px and %
// apart, a calc() of min() or max() as a sum of the two scaled; a
// keyword flips unless both are the same one. A fr interpolates with a
// fr.
func lengths(nonNegative bool) interp[style.Length] {
	return interp[style.Length]{
		lerp:  func(a, b style.Length, t float32) (style.Length, bool) { return lerpLength(a, b, t, nonNegative) },
		equal: same[style.Length],
	}
}

//nolint:gochecknoglobals // interpolations, fixed
var (
	anyLength  = lengths(false)
	sizeLength = lengths(true)
)

func lerpLength(a, b style.Length, t float32, nonNegative bool) (style.Length, bool) {
	if a.Kind != b.Kind {
		return a, false
	}
	switch a.Kind {
	case style.Fixed:
	case style.Fr:
		return style.Length{Kind: style.Fr, Px: max(lerpf(a.Px, b.Px, t), 0)}, true
	case style.Auto, style.MinContent, style.MaxContent, style.FitContent, style.None:
		return a, a == b
	}
	var none unique.Handle[style.Calc]
	if a.Calc != none || b.Calc != none {
		return a.Mix(b, t), true
	}
	l := style.Length{Kind: style.Fixed, Px: lerpf(a.Px, b.Px, t), Percent: lerpf(a.Percent, b.Percent, t),
		IsPercent: a.IsPercent || b.IsPercent}
	if nonNegative {
		switch {
		case !l.IsPercent:
			l.Px = max(l.Px, 0)
		case l.Px == 0:
			l.Percent = max(l.Percent, 0)
		}
	}
	return l, true
}

// pairs interpolates a pair of values both ways or not at all.
func pairs[T any](each interp[T]) interp[style.Size[T]] {
	return interp[style.Size[T]]{
		lerp: func(a, b style.Size[T], t float32) (style.Size[T], bool) {
			w, okW := each.lerp(a.Width, b.Width, t)
			h, okH := each.lerp(a.Height, b.Height, t)
			return style.Size[T]{Width: w, Height: h}, okW && okH
		},
		equal: func(a, b style.Size[T]) bool { return each.equal(a.Width, b.Width) && each.equal(a.Height, b.Height) },
	}
}

// sides interpolates the four sides of a value, all or none.
func sides[T any](each interp[T]) interp[style.Sides[T]] {
	return interp[style.Sides[T]]{
		lerp: func(a, b style.Sides[T], t float32) (style.Sides[T], bool) {
			var out style.Sides[T]
			ok := true
			for i := range 4 {
				v, k := each.lerp(*sideOf(&a, i), *sideOf(&b, i), t)
				*sideOf(&out, i), ok = v, ok && k
			}
			return out, ok
		},
		equal: func(a, b style.Sides[T]) bool {
			for i := range 4 {
				if !each.equal(*sideOf(&a, i), *sideOf(&b, i)) {
					return false
				}
			}
			return true
		},
	}
}

// fromStart is a position's offset from its start edge: one from the end
// is calc(100% - offset), as Chrome computes it.
func fromStart(l style.Length, fromEnd bool) style.Length {
	if !fromEnd {
		return l
	}
	from := style.Percent(100).Less(l)
	from.IsPercent = true
	return from
}

// positions interpolates background, mask and object positions, an
// offset from the end edge first made one from the start.
//
//nolint:gochecknoglobals // an interpolation, fixed
var positions = interp[style.BackgroundPosition]{
	lerp: func(a, b style.BackgroundPosition, t float32) (style.BackgroundPosition, bool) {
		if a == b {
			return a, true
		}
		x, okX := lerpLength(fromStart(a.X, a.FromRight), fromStart(b.X, b.FromRight), t, false)
		y, okY := lerpLength(fromStart(a.Y, a.FromBottom), fromStart(b.Y, b.FromBottom), t, false)
		return style.BackgroundPosition{X: x, Y: y}, okX && okY
	},
	equal: same[style.BackgroundPosition],
}

// backgroundSizes interpolates a width and a height, auto with auto; cover
// and contain flip.
//
//nolint:gochecknoglobals // an interpolation, fixed
var backgroundSizes = interp[style.BackgroundSize]{
	lerp: func(a, b style.BackgroundSize, t float32) (style.BackgroundSize, bool) {
		if a.Scale != style.ScaleNone || b.Scale != style.ScaleNone {
			return a, a == b
		}
		s, ok := pairs(sizeLength).lerp(style.Size[style.Length]{Width: a.Width, Height: a.Height},
			style.Size[style.Length]{Width: b.Width, Height: b.Height}, t)
		return style.BackgroundSize{Width: s.Width, Height: s.Height}, ok
	},
	equal: same[style.BackgroundSize],
}

// repeatable interpolates lists item by item, each repeated to the least
// common multiple of their lengths, as CSS's repeatable lists; nil is
// the list of first alone.
func repeatable[T comparable](each interp[T], first T) interp[[]T] {
	return interp[[]T]{
		lerp: func(a, b []T, t float32) ([]T, bool) {
			if len(a) == 0 {
				a = []T{first}
			}
			if len(b) == 0 {
				b = []T{first}
			}
			out := make([]T, lcm(len(a), len(b)))
			for i := range out {
				v, ok := each.lerp(a[i%len(a)], b[i%len(b)], t)
				if !ok {
					return nil, false
				}
				out[i] = v
			}
			return out, true
		},
		equal: slices.Equal[[]T],
	}
}

func lcm(a, b int) int {
	x, y := a, b
	for y != 0 {
		x, y = y, x%y
	}
	return a / x * b
}

// shadows interpolates shadow lists shadow by shadow, the shorter padded
// with transparent ones; an inset against an outer one flips.
//
//nolint:gochecknoglobals // an interpolation, fixed
var shadows = interp[[]style.Shadow]{lerp: lerpShadows, equal: slices.Equal[[]style.Shadow]}

func lerpShadows(a, b []style.Shadow, t float32) ([]style.Shadow, bool) {
	if len(a) == 0 && len(b) == 0 {
		return nil, true
	}
	out := make([]style.Shadow, max(len(a), len(b)))
	for i := range out {
		var x, y style.Shadow
		switch {
		case i < len(a) && i < len(b):
			x, y = a[i], b[i]
		case i < len(a):
			x, y = a[i], style.Shadow{Inset: a[i].Inset}
		default:
			x, y = style.Shadow{Inset: b[i].Inset}, b[i]
		}
		if x.Inset != y.Inset {
			return a, false
		}
		out[i] = style.Shadow{
			X: lerpf(x.X, y.X, t), Y: lerpf(x.Y, y.Y, t),
			Blur: max(lerpf(x.Blur, y.Blur, t), 0), Spread: lerpf(x.Spread, y.Spread, t),
			Color: lerpColor(x.Color, y.Color, t), Inset: x.Inset,
		}
	}
	return out, true
}

//nolint:gochecknoglobals // an interpolation, fixed
var transforms = interp[[]style.TransformOp]{
	lerp: func(a, b []style.TransformOp, t float32) ([]style.TransformOp, bool) {
		return lerpTransform(a, b, t), true
	},
	equal: slices.Equal[[]style.TransformOp],
}

// lineHeights interpolates a number with a number and a length with a
// length; normal, or one against the other, flips.
//
//nolint:gochecknoglobals // an interpolation, fixed
var lineHeights = interp[style.LineHeight]{
	lerp: func(a, b style.LineHeight, t float32) (style.LineHeight, bool) {
		switch {
		case a.Normal || b.Normal:
			return a, a == b
		case a.Factor != 0 && b.Factor != 0:
			return style.LineHeight{Factor: max(lerpf(a.Factor, b.Factor, t), 0)}, true
		case a.Factor == 0 && b.Factor == 0:
			return style.LineHeight{Px: max(lerpf(a.Px, b.Px, t), 0)}, true
		}
		return a, false
	},
	equal: same[style.LineHeight],
}

// tabSizes interpolate numbers with numbers and lengths with lengths.
//
//nolint:gochecknoglobals // an interpolation, fixed
var tabSizes = interp[style.TabSize]{
	lerp: func(a, b style.TabSize, t float32) (style.TabSize, bool) {
		if a.Length != b.Length {
			return a, false
		}
		return style.TabSize{Spaces: lerpf(a.Spaces, b.Spaces, t), Px: lerpf(a.Px, b.Px, t), Length: a.Length}, true
	},
	equal: same[style.TabSize],
}

//nolint:gochecknoglobals // an interpolation, fixed
var textSpacings = interp[style.Spacing]{
	lerp: func(a, b style.Spacing, t float32) (style.Spacing, bool) {
		return style.Spacing{Px: lerpf(a.Px, b.Px, t), Percent: lerpf(a.Percent, b.Percent, t)}, true
	},
	equal: same[style.Spacing],
}

// ratios interpolates aspect-ratio by its logarithm, as Chrome does; auto
// (0) flips.
//
//nolint:gochecknoglobals // an interpolation, fixed
var ratios = interp[float32]{
	lerp: func(a, b, t float32) (float32, bool) {
		if a == 0 || b == 0 {
			return a, a == b
		}
		return float32(math.Exp(float64(lerpf(float32(math.Log(float64(a))), float32(math.Log(float64(b))), t)))), true
	},
	equal: same[float32],
}

// spacings is border-spacing, in whole points as Chrome keeps it.
//
//nolint:gochecknoglobals // an interpolation, fixed
var spacings = interp[[2]float32]{
	lerp: func(a, b [2]float32, t float32) ([2]float32, bool) {
		var out [2]float32
		for i := range out {
			out[i] = float32(math.Floor(float64(max(lerpf(a[i], b[i], t), 0))))
		}
		return out, true
	},
	equal: same[[2]float32],
}

// imageSlices interpolates border-image-slice side by side; its 100%, kept as
// none, flips against numbers.
//
//nolint:gochecknoglobals // an interpolation, fixed
var imageSlices = interp[style.Sides[float32]]{
	lerp: func(a, b style.Sides[float32], t float32) (style.Sides[float32], bool) {
		var whole style.Sides[float32]
		if (a == whole) != (b == whole) {
			return a, false
		}
		return sides(numbers(math.MaxFloat32)).lerp(a, b, t)
	},
	equal: same[style.Sides[float32]],
}

// imageWidths interpolates border-image-width side by side, a number with
// a number and a length with a length or a percentage with a percentage;
// auto flips.
//
//nolint:gochecknoglobals // an interpolation, fixed
var imageWidths = sides(interp[style.ImageWidth]{
	lerp: func(a, b style.ImageWidth, t float32) (style.ImageWidth, bool) {
		if a.Kind != b.Kind || a.Kind == style.WidthAuto {
			return a, a == b
		}
		return style.ImageWidth{Kind: a.Kind, V: max(lerpf(a.V, b.V, t), 0)}, true
	},
	equal: same[style.ImageWidth],
})

// trackLists interpolates grid-template-columns and -rows: two lists of
// the same repeats of as many tracks, size by size; their line names
// flip halfway.
//
//nolint:gochecknoglobals // an interpolation, fixed
var trackLists = interp[[]style.TrackRepeat]{
	lerp: func(a, b []style.TrackRepeat, t float32) ([]style.TrackRepeat, bool) {
		if len(a) != len(b) || len(a) == 0 {
			return a, false
		}
		names := a
		if t >= 0.5 {
			names = b
		}
		out := make([]style.TrackRepeat, len(a))
		for i := range a {
			x, y := a[i], b[i]
			if x.Count != y.Count || len(x.Tracks) != len(y.Tracks) {
				return a, false
			}
			out[i] = style.TrackRepeat{Count: x.Count, Tracks: make([]style.Track, len(x.Tracks)),
				Names: names[i].Names, Before: names[i].Before, After: names[i].After}
			for k := range x.Tracks {
				lo, okLo := lerpLength(x.Tracks[k].Min, y.Tracks[k].Min, t, true)
				hi, okHi := lerpLength(x.Tracks[k].Max, y.Tracks[k].Max, t, true)
				if !okLo || !okHi {
					return a, false
				}
				out[i].Tracks[k] = style.Track{Min: lo, Max: hi}
			}
		}
		return out, true
	},
	equal: func(a, b []style.TrackRepeat) bool {
		return slices.EqualFunc(a, b, func(x, y style.TrackRepeat) bool {
			return x.Count == y.Count && slices.Equal(x.Tracks, y.Tracks) &&
				slices.EqualFunc(x.Names, y.Names, slices.Equal[[]string]) &&
				slices.Equal(x.Before, y.Before) && slices.Equal(x.After, y.After)
		})
	},
}

// visibility flips as CSS has it: visible all along the interval while
// one end is.
//
//nolint:gochecknoglobals // an interpolation, fixed
var visibility = interp[bool]{
	lerp: func(a, b bool, t float32) (bool, bool) {
		switch {
		case a == b, t <= 0:
			return a, true
		case t >= 1:
			return b, true
		}
		return true, true
	},
	equal: same[bool],
}

// clipPaths interpolates two basic shapes alike, as Chrome does
// (decision 0119): the same function in the same box and fill rule,
// circles and ellipses of the same radius keywords, polygons of as many
// points; any other two flip.
//
//nolint:gochecknoglobals // an interpolation, fixed
var clipPaths = interp[style.ClipPath]{
	lerp: func(a, b style.ClipPath, t float32) (style.ClipPath, bool) {
		switch {
		case !shapesAlike(&a, &b):
			return a, false
		case a.Shape == style.ShapeNone || a.Shape == style.ShapePath:
			return a, true
		}
		return lerpClip(&a, &b, t), true
	},
	equal: func(a, b style.ClipPath) bool { return a.Equal(&b) },
}

func shapesAlike(a, b *style.ClipPath) bool {
	if a.Shape != b.Shape || a.Box != b.Box || a.EvenOdd != b.EvenOdd {
		return false
	}
	switch a.Shape {
	case style.ShapeCircle, style.ShapeEllipse:
		for i := range a.Radius {
			if a.Radius[i].Side != b.Radius[i].Side {
				return false
			}
		}
	case style.ShapePolygon:
		return len(a.Points) == len(b.Points)
	case style.ShapeNone, style.ShapePath:
		return a.Equal(b)
	case style.ShapeInset:
	}
	return true
}

// filters interpolates filter lists function by function when one list
// starts with the other's functions, the shorter, or none, taking the
// neutral value of each it lacks (decision 0118); other lists flip.
//
//nolint:gochecknoglobals // an interpolation, fixed
var filters = interp[[]style.Filter]{lerp: lerpFilters, equal: slices.Equal[[]style.Filter]}

func lerpFilters(a, b []style.Filter, t float32) ([]style.Filter, bool) {
	n := max(len(a), len(b))
	if n == 0 {
		return nil, true
	}
	out := make([]style.Filter, n)
	for i := range n {
		var x, y style.Filter
		switch {
		case i < len(a) && i < len(b):
			x, y = a[i], b[i]
		case i < len(a):
			x, y = a[i], neutralFilter(a[i].Kind)
		default:
			x, y = neutralFilter(b[i].Kind), b[i]
		}
		if x.Kind != y.Kind {
			return a, false
		}
		out[i] = style.Filter{Kind: x.Kind, Amount: lerpf(x.Amount, y.Amount, t)}
		if x.Kind == style.DropShadow {
			out[i].Shadow = style.Shadow{
				X: lerpf(x.Shadow.X, y.Shadow.X, t), Y: lerpf(x.Shadow.Y, y.Shadow.Y, t),
				Blur: lerpf(x.Shadow.Blur, y.Shadow.Blur, t), Color: lerpColor(x.Shadow.Color, y.Shadow.Color, t),
			}
		}
	}
	return out, true
}

// neutralFilter is the function of kind that changes nothing.
func neutralFilter(kind style.FilterKind) style.Filter {
	switch kind {
	case style.Brightness, style.Contrast, style.OpacityFilter, style.Saturate:
		return style.Filter{Kind: kind, Amount: 1}
	case style.Blur, style.Grayscale, style.HueRotate, style.Invert, style.Sepia, style.DropShadow:
	}
	return style.Filter{Kind: kind}
}
