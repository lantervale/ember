package svg

import (
	"image/color"
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/colornames"
)

// units are CSS pixels per absolute unit.
var units = map[string]float64{ //nolint:gochecknoglobals // a constant table
	"px": 1, "pt": 4.0 / 3, "pc": 16, "mm": 96 / 25.4, "cm": 96 / 2.54, "in": 96, "q": 96 / 101.6,
}

// lengthContext is what relative lengths are of.
type lengthContext struct {
	// whole is what a percentage is a share of.
	whole float64
	// em is the font size.
	em float64
}

// number reads s as one plain number.
func number(s string) (float64, bool) {
	sc := &scanner{s: s}
	v, ok := sc.number()
	return v, ok && sc.done()
}

// length reads a number with an optional unit.
func length(s string, ctx lengthContext) (float64, bool) {
	sc := &scanner{s: s}
	v, ok := sc.number()
	if !ok {
		return 0, false
	}
	unit := strings.ToLower(strings.TrimSpace(sc.s[sc.i:]))
	switch unit {
	case "":
		return v, true
	case "%":
		return v * ctx.whole / 100, true
	case "em", "rem":
		return v * ctx.em, true
	case "ex", "ch":
		return v * ctx.em / 2, true
	}
	if mul, known := units[unit]; known {
		return v * mul, true
	}
	return 0, false
}

// fraction reads a number or a percentage as a share, 0–1: an opacity, a
// stop's offset.
func fraction(s string) (float64, bool) {
	sc := &scanner{s: s}
	v, ok := sc.number()
	if !ok {
		return 0, false
	}
	switch strings.TrimSpace(sc.s[sc.i:]) {
	case "":
	case "%":
		v /= 100
	default:
		return 0, false
	}
	return math.Max(0, math.Min(1, v)), true
}

func hexByte(s string) (uint8, bool) {
	v, err := strconv.ParseUint(s, 16, 8)
	return uint8(v), err == nil
}

func parseHex(h string) (color.NRGBA, bool) {
	if len(h) == 3 || len(h) == 4 {
		long := make([]byte, 0, 8)
		for i := range len(h) {
			long = append(long, h[i], h[i])
		}
		h = string(long)
	}
	if len(h) != 6 && len(h) != 8 {
		return color.NRGBA{}, false
	}
	c := color.NRGBA{A: 255}
	ok := true
	for i, into := range []*uint8{&c.R, &c.G, &c.B, &c.A}[:len(h)/2] {
		var one bool
		*into, one = hexByte(h[2*i : 2*i+2])
		ok = ok && one
	}
	return c, ok
}

// parseFunction reads rgb(), rgba(), hsl() or hsla(): name and what is
// between its brackets.
func parseFunction(name, inside string) (color.NRGBA, bool) {
	args := strings.FieldsFunc(inside, func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
	if len(args) != 3 && len(args) != 4 {
		return color.NRGBA{}, false
	}
	alpha, ok := 1.0, true
	if len(args) == 4 {
		alpha, ok = fraction(args[3])
	}
	var rgb [3]float64
	switch name {
	case "rgb", "rgba":
		for i, a := range args[:3] {
			v, isLength := length(a, lengthContext{whole: 255})
			ok = ok && isLength
			rgb[i] = v
		}
	case "hsl", "hsla":
		h, isNumber := number(strings.TrimSuffix(args[0], "deg"))
		sat, ok1 := fraction(args[1])
		light, ok2 := fraction(args[2])
		ok = ok && isNumber && ok1 && ok2 && strings.HasSuffix(args[1], "%") && strings.HasSuffix(args[2], "%")
		rgb = hsl(h, sat, light)
	default:
		ok = false
	}
	return color.NRGBA{round8(rgb[0]), round8(rgb[1]), round8(rgb[2]), round8(alpha * 255)}, ok
}

// parseColor reads a CSS colour: a name, #rgb, #rgba, #rrggbb, #rrggbbaa,
// rgb(), rgba(), hsl() and hsla() with commas or spaces. currentColor is
// the caller's.
func parseColor(s string) (color.NRGBA, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(s, "#") {
		return parseHex(s[1:])
	}
	if open := strings.IndexByte(s, '('); open > 0 && strings.HasSuffix(s, ")") {
		return parseFunction(s[:open], s[open+1:len(s)-1])
	}
	switch s {
	case "transparent":
		return color.NRGBA{}, true
	case "rebeccapurple":
		return color.NRGBA{102, 51, 153, 255}, true
	}
	c, ok := colornames.Map[s]
	return color.NRGBA(c), ok
}

func round8(v float64) uint8 { return uint8(math.Max(0, math.Min(255, v)) + 0.5) }

// hsl is the colour of hue h in degrees, saturation and lightness 0–1,
// as CSS Color 3 converts it, each channel 0–255.
func hsl(h, s, l float64) [3]float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	channel := func(n float64) float64 {
		k := math.Mod(n+h/30, 12)
		a := s * math.Min(l, 1-l)
		return 255 * (l - a*math.Max(-1, math.Min(math.Min(k-3, 9-k), 1)))
	}
	return [3]float64{channel(0), channel(8), channel(4)}
}

// transform is the transform function name of the numbers v; ok is
// false for another name or count of numbers.
func transform(name string, v []float64) (Matrix, bool) {
	arg := func(i int, def float64) float64 {
		if i < len(v) {
			return v[i]
		}
		return def
	}
	oneOrTwo := len(v) == 1 || len(v) == 2
	switch name {
	case "matrix":
		return Matrix{arg(0, 0), arg(1, 0), arg(2, 0), arg(3, 0), arg(4, 0), arg(5, 0)}, len(v) == 6
	case "translate":
		return translate(arg(0, 0), arg(1, 0)), oneOrTwo
	case "translateX":
		return translate(arg(0, 0), 0), len(v) == 1
	case "translateY":
		return translate(0, arg(0, 0)), len(v) == 1
	case "scale":
		return scale(arg(0, 1), arg(1, arg(0, 1))), oneOrTwo
	case "rotate":
		sin, cos := math.Sincos(arg(0, 0) * math.Pi / 180)
		turn := Matrix{cos, sin, -sin, cos, 0, 0}
		return translate(arg(1, 0), arg(2, 0)).Mul(turn).Mul(translate(-arg(1, 0), -arg(2, 0))), len(v) == 1 || len(v) == 3
	case "skewX":
		return Matrix{1, 0, math.Tan(arg(0, 0) * math.Pi / 180), 1, 0, 0}, len(v) == 1
	case "skewY":
		return Matrix{1, math.Tan(arg(0, 0) * math.Pi / 180), 0, 1, 0, 0}, len(v) == 1
	}
	return Identity(), false
}

// parseTransform reads a transform list; ok is false when a part of it
// is no transform, which makes the whole attribute void.
func parseTransform(s string) (Matrix, bool) {
	m := Identity()
	for count := 0; ; count++ {
		s = strings.TrimLeft(s, " \t\r\n\f,")
		if s == "" {
			return m, true
		}
		open := strings.IndexByte(s, '(')
		if open < 0 || count >= maxTransforms {
			return Identity(), false
		}
		end := strings.IndexByte(s[open:], ')')
		if end < 0 {
			return Identity(), false
		}
		name := strings.TrimSpace(s[:open])
		sc := &scanner{s: s[open+1 : open+end]}
		s = s[open+end+1:]
		v := sc.list(7)
		if !sc.done() {
			// CSS writes these with units: translate(4px), rotate(45deg).
			v = v[:0]
			for _, f := range strings.FieldsFunc(sc.s, func(r rune) bool { return r == ',' || r == ' ' }) {
				x, isNumber := number(strings.TrimSuffix(strings.TrimSuffix(f, "px"), "deg"))
				if !isNumber {
					return Identity(), false
				}
				v = append(v, x)
			}
		}
		t, ok := transform(name, v)
		if !ok {
			return Identity(), false
		}
		m = m.Mul(t)
	}
}
