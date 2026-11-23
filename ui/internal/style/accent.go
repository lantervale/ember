package style

import (
	"math"

	"github.com/lantervale/ember/color"
)

// Shade is the state a control's accent is drawn in.
type Shade uint8

const (
	ShadeRest Shade = iota
	ShadeHovered
	ShadePressed
)

// accentShift is how far Chrome moves an accent's HSL lightness for a
// hovered or a pressed control.
const accentShift = 0.11

// rangeLight is the relative luminance from which Chrome shades a range's
// accent as a light one, measured between two greys.
const rangeLight = 0.256

// darkTickLuminance is the relative luminance of the dark grey tick
// Chrome draws on a checkbox of a light accent.
const darkTickLuminance = 0.0437

// luminance is c's relative luminance.
func luminance(c color.Color) float32 { return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B }

// LightRange reports whether Chrome shades a range of accent a as a light
// accent.
func LightRange(a color.Color) bool { return luminance(a) >= rangeLight }

// LightCheck reports whether Chrome draws a checkbox of accent a with its
// dark tick, which contrasts more with it than white, and shades it as a
// light accent; a radio is shaded as a checkbox.
func LightCheck(a color.Color) bool {
	l := luminance(a)
	return (l+0.05)*(l+0.05) > 1.05*(darkTickLuminance+0.05)
}

// AccentShade is accent a as Chrome draws it on a control in state s: a
// hovered control's darker, a pressed one's lighter, the other way round
// for a light accent, its hue, saturation and alpha kept.
func AccentShade(a color.Color, s Shade, light bool) color.Color {
	if s == ShadeRest {
		return a
	}
	h, sat, l := hsl(a.SRGB())
	shift := accentShift
	if !light == (s == ShadeHovered) {
		shift = -shift
	}
	out := color.FromSRGB(fromHSL(h, sat, min(max(l+shift, 0), 1)))
	out.A = a.A
	return out
}

// hsl is the hue (0 to 6), saturation and lightness of the sRGB colour c.
func hsl(c [3]float32) (float64, float64, float64) {
	r, g, b := float64(c[0]), float64(c[1]), float64(c[2])
	hi, lo := max(r, g, b), min(r, g, b)
	l := (hi + lo) / 2
	d := hi - lo
	if d == 0 {
		return 0, 0, l
	}
	s := d / (1 - math.Abs(2*l-1))
	var h float64
	switch hi {
	case r:
		h = math.Mod((g-b)/d+6, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h, s, l
}

// fromHSL is the sRGB colour of hue h (0 to 6), saturation s and
// lightness l.
func fromHSL(h, s, l float64) [3]float32 {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch int(h) {
	case 0:
		r, g = c, x
	case 1:
		r, g = x, c
	case 2:
		g, b = c, x
	case 3:
		g, b = x, c
	case 4:
		r, b = x, c
	default:
		r, b = c, x
	}
	return [3]float32{float32(r + m), float32(g + m), float32(b + m)}
}
