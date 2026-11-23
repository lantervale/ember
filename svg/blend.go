package svg

import (
	"image"
	"math"
)

// blendOnto draws layer over dst through opacity, its colours mixed with
// those under them by mode.
func blendOnto(dst, layer *image.RGBA, opacity float64, mode BlendMode) {
	o := float32(opacity) / 255
	b := layer.Rect
	for y := b.Min.Y; y < b.Max.Y; y++ {
		src := layer.Pix[layer.PixOffset(b.Min.X, y):][:4*b.Dx()]
		out := dst.Pix[dst.PixOffset(b.Min.X, y):][:4*b.Dx()]
		for i := 0; i < len(src); i += 4 {
			if src[i+3] == 0 {
				continue
			}
			s := [4]float32{float32(src[i]) * o, float32(src[i+1]) * o, float32(src[i+2]) * o, float32(src[i+3]) * o}
			d := [4]float32{float32(out[i]) / 255, float32(out[i+1]) / 255, float32(out[i+2]) / 255, float32(out[i+3]) / 255}
			px := blendPixel(s, d, mode)
			a := min(max(px[3], 0), 1)
			out[i+3] = uint8(a*255 + 0.5)
			for c := range 3 {
				out[i+c] = uint8(min(max(px[c], 0), a)*255 + 0.5)
			}
		}
	}
}

// blendPixel is the premultiplied colour s drawn over b by mode, as CSS
// compositing mixes them.
func blendPixel(s, b [4]float32, mode BlendMode) [4]float32 {
	as, ab := s[3], b[3]
	if mode == BlendPlusLighter {
		return [4]float32{min(s[0]+b[0], 1), min(s[1]+b[1], 1), min(s[2]+b[2], 1), min(as+ab, 1)}
	}
	var cs, cb [3]float64
	for k := range 3 {
		if as > 0 {
			cs[k] = float64(min(s[k]/as, 1))
		}
		if ab > 0 {
			cb[k] = float64(min(b[k]/ab, 1))
		}
	}
	mixed := mix(cb, cs, mode)
	var out [4]float32
	out[3] = as + ab - as*ab
	for k := range 3 {
		out[k] = s[k]*(1-ab) + b[k]*(1-as) + as*ab*float32(mixed[k])
	}
	return out
}

// mix is B(cb, cs) of CSS compositing: the colour of the backdrop cb
// and the source cs mixed by mode, neither premultiplied.
func mix(cb, cs [3]float64, mode BlendMode) [3]float64 {
	switch mode {
	case BlendHue:
		return setLum(setSat(cs, sat(cb)), lum(cb))
	case BlendSaturation:
		return setLum(setSat(cb, sat(cs)), lum(cb))
	case BlendColor:
		return setLum(cs, lum(cb))
	case BlendLuminosity:
		return setLum(cb, lum(cs))
	case BlendNormal, BlendMultiply, BlendScreen, BlendOverlay, BlendDarken, BlendLighten, BlendColorDodge,
		BlendColorBurn, BlendHardLight, BlendSoftLight, BlendDifference, BlendExclusion, BlendPlusLighter:
	}
	var out [3]float64
	for k := range 3 {
		out[k] = separable(cb[k], cs[k], mode)
	}
	return out
}

func separable(b, s float64, mode BlendMode) float64 {
	switch mode {
	case BlendMultiply:
		return b * s
	case BlendScreen:
		return b + s - b*s
	case BlendOverlay:
		return hardLight(s, b)
	case BlendDarken:
		return math.Min(b, s)
	case BlendLighten:
		return math.Max(b, s)
	case BlendColorDodge:
		switch {
		case b == 0:
			return 0
		case s >= 1:
			return 1
		}
		return math.Min(1, b/(1-s))
	case BlendColorBurn:
		switch {
		case b >= 1:
			return 1
		case s == 0:
			return 0
		}
		return 1 - math.Min(1, (1-b)/s)
	case BlendHardLight:
		return hardLight(b, s)
	case BlendSoftLight:
		if s <= 0.5 {
			return b - (1-2*s)*b*(1-b)
		}
		d := math.Sqrt(b)
		if b <= 0.25 {
			d = ((16*b-12)*b + 4) * b
		}
		return b + (2*s-1)*(d-b)
	case BlendDifference:
		return math.Abs(b - s)
	case BlendExclusion:
		return b + s - 2*b*s
	case BlendNormal, BlendHue, BlendSaturation, BlendColor, BlendLuminosity, BlendPlusLighter:
	}
	return s
}

func hardLight(b, s float64) float64 {
	if s <= 0.5 {
		return b * 2 * s
	}
	s = 2*s - 1
	return b + s - b*s
}

func lum(c [3]float64) float64 { return 0.3*c[0] + 0.59*c[1] + 0.11*c[2] }

func clipColor(c [3]float64) [3]float64 {
	l := lum(c)
	lo, hi := math.Min(c[0], math.Min(c[1], c[2])), math.Max(c[0], math.Max(c[1], c[2]))
	for k := range c {
		if lo < 0 && l != lo {
			c[k] = l + (c[k]-l)*l/(l-lo)
		}
		if hi > 1 && hi != l {
			c[k] = l + (c[k]-l)*(1-l)/(hi-l)
		}
	}
	return c
}

func setLum(c [3]float64, l float64) [3]float64 {
	d := l - lum(c)
	return clipColor([3]float64{c[0] + d, c[1] + d, c[2] + d})
}

func sat(c [3]float64) float64 {
	return math.Max(c[0], math.Max(c[1], c[2])) - math.Min(c[0], math.Min(c[1], c[2]))
}

func setSat(c [3]float64, s float64) [3]float64 {
	lo, mid, hi := 0, 1, 2
	if c[lo] > c[mid] {
		lo, mid = mid, lo
	}
	if c[mid] > c[hi] {
		mid, hi = hi, mid
	}
	if c[lo] > c[mid] {
		lo, mid = mid, lo
	}
	var out [3]float64
	if c[hi] > c[lo] {
		out[mid] = (c[mid] - c[lo]) * s / (c[hi] - c[lo])
		out[hi] = s
	}
	return out
}
