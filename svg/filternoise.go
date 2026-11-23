package svg

import (
	"math"
	"strings"
)

// The constants of Filter Effects' reference code for feTurbulence.
const (
	noiseB       = 0x100
	noiseBM      = 0xff
	noisePerlinN = 0x1000
	randM        = 2147483647
	randA        = 16807
	randQ        = 127773
	randR        = 2836
	// maxOctaves is the most octaves noise is summed over: those past it
	// add less than a level.
	maxOctaves = 10
)

// noise reads an feTurbulence; ok is false for one in error, which
// draws nothing.
func noise(props map[string]string, c filterContext) (Noise, bool) {
	n := Noise{Octaves: 1}
	freq, ok := pair(props["baseFrequency"], filterContext{}, 0)
	if !ok || freq.X < 0 || freq.Y < 0 {
		return n, false
	}
	n.BaseFrequency = freq
	if v, valid := number(props["numOctaves"]); valid {
		n.Octaves = int(max(0, min(v, maxOctaves)))
	}
	if v, valid := number(props["seed"]); valid {
		n.Seed = v
	}
	n.Fractal = strings.TrimSpace(props["type"]) == "fractalNoise"
	n.Stitch = strings.TrimSpace(props["stitchTiles"]) == "stitch"
	if c.boxed && c.bbox.W > 0 && c.bbox.H > 0 {
		n.BaseFrequency = Point{freq.X / c.bbox.W, freq.Y / c.bbox.H}
	}
	return n, true
}

// perlin is the lattice and gradients the reference code's init makes
// from a seed.
type perlin struct {
	lattice  [noiseB + noiseB + 2]int
	gradient [4][noiseB + noiseB + 2][2]float64
}

func random(seed int64) int64 {
	r := randA*(seed%randQ) - randR*(seed/randQ)
	if r <= 0 {
		r += randM
	}
	return r
}

func newPerlin(seed float64) *perlin {
	s := int64(max(-randM, min(seed, randM)))
	if s <= 0 {
		s = -(s % (randM - 1)) + 1
	}
	if s > randM-1 {
		s = randM - 1
	}
	p := &perlin{}
	for k := range 4 {
		for i := range noiseB {
			p.lattice[i] = i
			for j := range 2 {
				s = random(s)
				p.gradient[k][i][j] = float64(s%(noiseB+noiseB)-noiseB) / noiseB
			}
			g := &p.gradient[k][i]
			if l := math.Sqrt(g[0]*g[0] + g[1]*g[1]); l > 0 {
				g[0] /= l
				g[1] /= l
			}
		}
	}
	for i := noiseB - 1; i > 0; i-- {
		s = random(s)
		j := int(s % noiseB)
		p.lattice[i], p.lattice[j] = p.lattice[j], p.lattice[i]
	}
	for i := range noiseB + 2 {
		p.lattice[noiseB+i] = p.lattice[i]
		for k := range 4 {
			p.gradient[k][noiseB+i] = p.gradient[k][i]
		}
	}
	return p
}

// stitch is where the lattice wraps round to tile a region.
type stitch struct{ width, height, wrapX, wrapY int }

func sCurve(t float64) float64 { return t * t * (3 - 2*t) }

func (p *perlin) noise2(channel int, x, y float64, st *stitch) float64 {
	t := x + noisePerlinN
	bx0 := int(t)
	bx1 := bx0 + 1
	rx0 := t - float64(int(t))
	rx1 := rx0 - 1
	t = y + noisePerlinN
	by0 := int(t)
	by1 := by0 + 1
	ry0 := t - float64(int(t))
	ry1 := ry0 - 1
	if st != nil {
		if bx0 >= st.wrapX {
			bx0 -= st.width
		}
		if bx1 >= st.wrapX {
			bx1 -= st.width
		}
		if by0 >= st.wrapY {
			by0 -= st.height
		}
		if by1 >= st.wrapY {
			by1 -= st.height
		}
	}
	bx0 &= noiseBM
	bx1 &= noiseBM
	by0 &= noiseBM
	by1 &= noiseBM
	i, j := p.lattice[bx0], p.lattice[bx1]
	b00, b10 := p.lattice[i+by0], p.lattice[j+by0]
	b01, b11 := p.lattice[i+by1], p.lattice[j+by1]
	sx, sy := sCurve(rx0), sCurve(ry0)
	g := &p.gradient[channel]
	u := rx0*g[b00][0] + ry0*g[b00][1]
	v := rx1*g[b10][0] + ry0*g[b10][1]
	a := u + sx*(v-u)
	u = rx0*g[b01][0] + ry1*g[b01][1]
	v = rx1*g[b11][0] + ry1*g[b11][1]
	b := u + sx*(v-u)
	return a + sy*(b-a)
}

// turbulence is the reference code's turbulence(): the noise of one
// channel at a point, over a tile when stitching.
func (p *perlin) turbulence(channel int, at Point, n *Noise, tile Box) float64 {
	fx, fy := n.BaseFrequency.X, n.BaseFrequency.Y
	var st *stitch
	if n.Stitch {
		adjust := func(f, size float64) float64 {
			if f == 0 {
				return f
			}
			lo, hi := math.Floor(size*f)/size, math.Ceil(size*f)/size
			if f/lo < hi/f {
				return lo
			}
			return hi
		}
		fx, fy = adjust(fx, tile.W), adjust(fy, tile.H)
		st = &stitch{width: int(tile.W*fx + 0.5), height: int(tile.H*fy + 0.5)}
		st.wrapX = int(tile.X*fx + noisePerlinN + float64(st.width))
		st.wrapY = int(tile.Y*fy + noisePerlinN + float64(st.height))
	}
	sum, ratio := 0.0, 1.0
	x, y := at.X*fx, at.Y*fy
	for range n.Octaves {
		v := p.noise2(channel, x, y, st)
		if !n.Fractal {
			v = math.Abs(v)
		}
		sum += v / ratio
		x, y, ratio = x*2, y*2, ratio*2
		if st != nil {
			st.width *= 2
			st.wrapX = 2*st.wrapX - noisePerlinN
			st.height *= 2
			st.wrapY = 2*st.wrapY - noisePerlinN
		}
	}
	return sum
}

// turbulence paints out's area with p's noise. Each pixel's is from the
// point of the filter's space at its centre and half a unit further, as
// Chrome reads it, and a stitched tile is the region's size in whole
// pixels with its corner at the origin, as Chrome cuts it.
func (fr *filterRun) turbulence(out *fbuf, p *Primitive) {
	inv, ok := fr.m.Inverse()
	if !ok || p.Noise.Octaves == 0 {
		return
	}
	s := scales(fr.m)
	tile := Box{W: math.Round(p.Region.W*s.X) / s.X, H: math.Round(p.Region.H*s.Y) / s.Y}
	pn := newPerlin(p.Noise.Seed)
	a := out.area
	for y := a.Min.Y; y < a.Max.Y; y++ {
		for x := a.Min.X; x < a.Max.X; x++ {
			at := inv.Apply(Point{float64(x) + 0.5, float64(y) + 0.5}).add(Point{0.5, 0.5})
			var c [4]float32
			for k := range 4 {
				v := pn.turbulence(k, at, &p.Noise, tile)
				if p.Noise.Fractal {
					v = (v + 1) / 2
				}
				c[k] = float32(min(max(v, 0), 1))
			}
			px := out.pix[4*((y-fr.box.Min.Y)*fr.w+x-fr.box.Min.X):][:4]
			px[3] = c[3]
			for k := range 3 {
				px[k] = c[k] * c[3]
			}
		}
	}
}
