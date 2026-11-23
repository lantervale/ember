package colourfont

import "math"

// composite is src composited with dst by COLR's mode: Porter and
// Duff's operators to 12, the blend modes of CSS after.
func composite(mode uint8, src, dst *layer, w, h int) *layer {
	if src == nil {
		src = newLayer(w, h)
	}
	if dst == nil {
		dst = newLayer(w, h)
	}
	out := newLayer(w, h)
	for i := 0; i < len(out.px); i += 4 {
		s := [4]float32(src.px[i : i+4])
		d := [4]float32(dst.px[i : i+4])
		r := compositePixel(mode, s, d)
		copy(out.px[i:i+4], r[:])
	}
	return out
}

func compositePixel(mode uint8, s, d [4]float32) [4]float32 {
	sa, da := s[3], d[3]
	if mode <= 12 {
		var fa, fb float32
		switch mode {
		case 0:
		case 1:
			fa = 1
		case 2:
			fb = 1
		case 3:
			fa, fb = 1, 1-sa
		case 4:
			fa, fb = 1-da, 1
		case 5:
			fa = da
		case 6:
			fb = sa
		case 7:
			fa = 1 - da
		case 8:
			fb = 1 - sa
		case 9:
			fa, fb = da, 1-sa
		case 10:
			fa, fb = 1-da, sa
		case 11:
			fa, fb = 1-da, 1-sa
		case 12:
			fa, fb = 1, 1
		}
		var out [4]float32
		for i := range out {
			out[i] = min(s[i]*fa+d[i]*fb, 1)
		}
		return out
	}
	if mode > 27 {
		return s
	}
	var cs, cd [3]float32
	for i := range 3 {
		if sa > 0 {
			cs[i] = s[i] / sa
		}
		if da > 0 {
			cd[i] = d[i] / da
		}
	}
	var b [3]float32
	if mode >= 24 {
		b = nonSeparable(mode, cs, cd)
	} else {
		for i := range 3 {
			b[i] = blend(mode, cs[i], cd[i])
		}
	}
	var out [4]float32
	for i := range 3 {
		out[i] = (1-da)*s[i] + (1-sa)*d[i] + sa*da*b[i]
	}
	out[3] = sa + da - sa*da
	return out
}

func blend(mode uint8, cs, cd float32) float32 {
	switch mode {
	case 13:
		return cs + cd - cs*cd
	case 14:
		return hardLight(cd, cs)
	case 15:
		return min(cs, cd)
	case 16:
		return max(cs, cd)
	case 17:
		switch {
		case cd == 0:
			return 0
		case cs >= 1:
			return 1
		}
		return min(1, cd/(1-cs))
	case 18:
		switch {
		case cd >= 1:
			return 1
		case cs <= 0:
			return 0
		}
		return 1 - min(1, (1-cd)/cs)
	case 19:
		return hardLight(cs, cd)
	case 20:
		if cs <= 0.5 {
			return cd - (1-2*cs)*cd*(1-cd)
		}
		var dd float32
		if cd <= 0.25 {
			dd = ((16*cd-12)*cd + 4) * cd
		} else {
			dd = float32(math.Sqrt(float64(cd)))
		}
		return cd + (2*cs-1)*(dd-cd)
	case 21:
		return float32(math.Abs(float64(cs - cd)))
	case 22:
		return cs + cd - 2*cs*cd
	case 23:
		return cs * cd
	}
	return cs
}

func hardLight(cs, cd float32) float32 {
	if cs <= 0.5 {
		return cd * 2 * cs
	}
	s := 2*cs - 1
	return cd + s - cd*s
}

// nonSeparable is CSS's hue, saturation, color and luminosity.
func nonSeparable(mode uint8, cs, cd [3]float32) [3]float32 {
	switch mode {
	case 24:
		return setLum(setSat(cs, sat(cd)), lum(cd))
	case 25:
		return setLum(setSat(cd, sat(cs)), lum(cd))
	case 26:
		return setLum(cs, lum(cd))
	}
	return setLum(cd, lum(cs))
}

func lum(c [3]float32) float32 { return 0.3*c[0] + 0.59*c[1] + 0.11*c[2] }

func sat(c [3]float32) float32 { return max(c[0], c[1], c[2]) - min(c[0], c[1], c[2]) }

func setLum(c [3]float32, l float32) [3]float32 {
	d := l - lum(c)
	for i := range c {
		c[i] += d
	}
	l = lum(c)
	n, x := min(c[0], c[1], c[2]), max(c[0], c[1], c[2])
	for i := range c {
		if n < 0 && l-n != 0 {
			c[i] = l + (c[i]-l)*l/(l-n)
		}
		if x > 1 && x-l != 0 {
			c[i] = l + (c[i]-l)*(1-l)/(x-l)
		}
	}
	return c
}

func setSat(c [3]float32, s float32) [3]float32 {
	hi, lo := 0, 0
	for i := range c {
		if c[i] > c[hi] {
			hi = i
		}
		if c[i] < c[lo] {
			lo = i
		}
	}
	if hi == lo {
		return [3]float32{}
	}
	mid := 3 - hi - lo
	var out [3]float32
	if c[hi] > c[lo] {
		out[mid] = (c[mid] - c[lo]) * s / (c[hi] - c[lo])
		out[hi] = s
	}
	return out
}
