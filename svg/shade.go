package svg

import "math"

// rampSize is how many colours a gradient is sampled into.
const rampSize = 1024

// shader paints one gradient: its colours sampled once along the
// gradient, premultiplied, then each pixel's place along it looked up.
type shader struct {
	ramp   *[rampSize][4]float32
	of     *Gradient
	faded  float32
	g      *Gradient
	inv    Matrix // pixels → the gradient's space
	axis   Point
	simple bool // a radial gradient from its centre
	// The cone of a radial gradient with a focal point or radius.
	cd    Point
	dr, a float64
}

// set readies the shader for g on a shape drawn through m; false when
// the gradient cannot be drawn.
func (s *shader) set(g *Gradient, m Matrix, opacity float32) bool {
	inv, ok := m.Mul(g.Transform).Inverse()
	if !ok || len(g.Stops) == 0 {
		return false
	}
	s.g, s.inv = g, inv
	if s.ramp == nil {
		s.ramp = new([rampSize][4]float32)
	}
	if s.of != g || s.faded != opacity {
		s.of, s.faded = g, opacity
		at := 0
		for i := range s.ramp {
			t := float64(i) / (rampSize - 1)
			for at+1 < len(g.Stops) && g.Stops[at+1].Offset <= t {
				at++
			}
			from, to := g.Stops[at], g.Stops[at]
			if t < from.Offset {
				to = from
			} else if at+1 < len(g.Stops) {
				to = g.Stops[at+1]
			}
			f := float32(0)
			if to.Offset > from.Offset {
				f = float32((t - from.Offset) / (to.Offset - from.Offset))
			}
			mix := func(x, y uint8) float32 { return float32(x) + (float32(y)-float32(x))*f }
			a := mix(from.Color.A, to.Color.A) / 255 * opacity
			s.ramp[i] = [4]float32{
				mix(from.Color.R, to.Color.R) * a, mix(from.Color.G, to.Color.G) * a,
				mix(from.Color.B, to.Color.B) * a, 255 * a,
			}
		}
	}
	if !g.Radial {
		s.axis = g.End.sub(g.Start)
		s.axis = s.axis.mul(1 / s.axis.dot(s.axis))
		return true
	}
	s.cd = g.End.sub(g.Start)
	s.dr = g.Radius - g.FocalRadius
	s.a = s.cd.dot(s.cd) - s.dr*s.dr
	s.simple = s.cd == (Point{}) && g.FocalRadius == 0
	return true
}

// spread folds t, a place along the gradient, into 0–1.
func (s *shader) spread(t float64) float64 {
	switch s.g.Spread {
	case Repeat:
		return t - math.Floor(t)
	case Reflect:
		t = math.Mod(t, 2)
		if t < 0 {
			t += 2
		}
		if t > 1 {
			t = 2 - t
		}
		return t
	case Pad:
	}
	return math.Max(0, math.Min(1, t))
}

// cone is where p lies along a radial gradient whose circles grow from
// the focal one to the outer one; ok is false outside the cone they
// sweep.
func (s *shader) cone(p Point) (float64, bool) {
	g := s.g
	pd := p.sub(g.Start)
	b := pd.dot(s.cd) + g.FocalRadius*s.dr
	c := pd.dot(pd) - g.FocalRadius*g.FocalRadius
	if math.Abs(s.a) < 1e-12 {
		if b == 0 {
			return 0, false
		}
		t := c / (2 * b)
		return t, g.FocalRadius+t*s.dr >= 0
	}
	disc := b*b - s.a*c
	if disc < 0 {
		return 0, false
	}
	root := math.Sqrt(disc)
	t0, t1 := (b+root)/s.a, (b-root)/s.a
	if t0 < t1 {
		t0, t1 = t1, t0
	}
	if g.FocalRadius+t0*s.dr >= 0 {
		return t0, true
	}
	return t1, g.FocalRadius+t1*s.dr >= 0
}

// row paints the pixels of row y from x0 that cover says, into pix.
func (s *shader) row(pix []uint8, x0, y int, cover []float32) {
	g := s.g
	p := s.inv.Apply(Point{float64(x0) + 0.5, float64(y) + 0.5})
	step := Point{s.inv.A, s.inv.B}
	for i, c := range cover {
		if c > 0 {
			var t float64
			ok := true
			switch {
			case !g.Radial:
				t = p.sub(g.Start).dot(s.axis)
			case s.simple:
				t = p.sub(g.End).len() / g.Radius
			default:
				t, ok = s.cone(p)
			}
			// A place that is no number, from numbers no picture has, paints nothing.
			if u := s.spread(t); ok && u >= 0 && u <= 1 {
				blend(pix[4*i:4*i+4:4*i+4], &s.ramp[int(u*(rampSize-1)+0.5)], c)
			}
		}
		p = p.add(step)
	}
}
