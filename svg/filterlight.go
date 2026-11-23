package svg

import (
	"image/color"
	"math"
	"strings"
)

// lighting reads feDiffuseLighting or feSpecularLighting, and its light;
// one with no light, or a constant below zero, draws nothing.
func (b *builder) lighting(p *Primitive, n *node, props map[string]string, st *style, c filterContext) {
	num := func(props map[string]string, k string, def float64) float64 {
		if v, ok := number(props[k]); ok {
			return v
		}
		return def
	}
	p.Kind = DiffuseLighting
	constant := "diffuseConstant"
	if n.name == "feSpecularLighting" {
		p.Kind, constant = SpecularLighting, "specularConstant"
	}
	p.Lighting = Lighting{
		SurfaceScale: num(props, "surfaceScale", 1),
		Constant:     num(props, constant, 1),
		Exponent:     min(max(num(props, "specularExponent", 1), 1), 128),
	}
	p.Color = color.NRGBA{255, 255, 255, 255}
	if v, ok := props["lighting-color"]; ok {
		if strings.EqualFold(strings.TrimSpace(v), "currentColor") {
			p.Color = st.color
		} else if parsed, valid := parseColor(v); valid {
			p.Color = parsed
		}
	}
	var light *node
	for _, k := range n.kids {
		if k.name == "feDistantLight" || k.name == "fePointLight" || k.name == "feSpotLight" {
			light = k
			break
		}
	}
	if light == nil || p.Lighting.Constant < 0 {
		p.Kind = Clear
		return
	}
	lp := b.props(light)
	at := func(x, y, z string) Point3 {
		q := Point3{num(lp, x, 0), num(lp, y, 0), num(lp, z, 0)}
		if c.boxed {
			bb := c.bbox
			q = Point3{bb.X + q.X*bb.W, bb.Y + q.Y*bb.H, q.Z * math.Sqrt((bb.W*bb.W+bb.H*bb.H)/2)}
		}
		return q
	}
	l := &p.Lighting.Light
	switch light.name {
	case "feDistantLight":
		l.Kind, l.Azimuth, l.Elevation = DistantLight, num(lp, "azimuth", 0), num(lp, "elevation", 0)
	case "fePointLight":
		l.Kind, l.At = PointLight, at("x", "y", "z")
	default:
		l.Kind, l.At, l.PointsAt = SpotLight, at("x", "y", "z"), at("pointsAtX", "pointsAtY", "pointsAtZ")
		l.Exponent = min(max(num(lp, "specularExponent", 1), 1), 128)
		l.Cone = math.Abs(num(lp, "limitingConeAngle", 0))
	}
}

// coneEdge is how wide, in the cosine of the angle, the edge of a spot
// light's cone fades, as Skia fades it.
const coneEdge = 0.016

// light writes in's alpha lit as a surface by p to out, its light placed
// in pixels through the run's transform.
func (fr *filterRun) light(out, in *fbuf, p *Primitive) {
	lt := &p.Lighting
	m := fr.m
	toPixels := func(q Point3) Point3 {
		at := m.Apply(Point{q.X, q.Y})
		return Point3{at.X - float64(fr.box.Min.X), at.Y - float64(fr.box.Min.Y), q.Z * math.Sqrt(math.Abs(m.det()))}
	}
	l := lt.Light
	var dir, from, axis Point3
	cosCone := 0.0
	switch l.Kind {
	case DistantLight:
		az, el := l.Azimuth*math.Pi/180, l.Elevation*math.Pi/180
		dir = Point3{math.Cos(az) * math.Cos(el), math.Sin(az) * math.Cos(el), math.Sin(el)}
	case PointLight, SpotLight:
		from = toPixels(l.At)
		axis = unit3(sub3(toPixels(l.PointsAt), from))
		if l.Cone > 0 && l.Cone < 90 {
			cosCone = math.Cos(l.Cone * math.Pi / 180)
		}
	}
	colour := premultiplied(p.Color, p.LinearRGB)
	a := out.area.Sub(fr.box.Min)
	if a.Empty() {
		return
	}
	alpha := func(x, y int) float64 { return float64(in.pix[4*(y*fr.w+x)+3]) }
	ss := lt.SurfaceScale * math.Sqrt(math.Abs(m.det()))
	for y := a.Min.Y; y < a.Max.Y; y++ {
		for x := a.Min.X; x < a.Max.X; x++ {
			nx, ny := sobel(alpha, x, y, a.Min.X, a.Min.Y, a.Max.X, a.Max.Y)
			normal := unit3(Point3{-ss * nx, -ss * ny, 1})
			ray, shade := dir, [3]float64{float64(colour[0]), float64(colour[1]), float64(colour[2])}
			if l.Kind != DistantLight {
				ray = unit3(sub3(from, Point3{float64(x), float64(y), ss * alpha(x, y)}))
			}
			if l.Kind == SpotLight {
				cos := -dot3(ray, axis)
				f := 0.0
				if cos >= cosCone {
					f = math.Pow(cos, l.Exponent)
					if cos < cosCone+coneEdge {
						f *= (cos - cosCone) / coneEdge
					}
				}
				for k := range shade {
					shade[k] *= f
				}
			}
			var factor float64
			if p.Kind == DiffuseLighting {
				factor = lt.Constant * dot3(normal, ray)
			} else {
				half := unit3(Point3{ray.X, ray.Y, ray.Z + 1})
				factor = lt.Constant * math.Pow(math.Max(dot3(normal, half), 0), lt.Exponent)
			}
			var px [4]float32
			for k := range 3 {
				px[k] = float32(min(max(factor*shade[k], 0), 1))
			}
			px[3] = 1
			if p.Kind == SpecularLighting {
				px[3] = max(px[0], px[1], px[2])
			}
			copy(out.pix[4*(y*fr.w+x):][:4], px[:])
		}
	}
}

// sobel is the slope of the surface at x, y along x and y, by Filter
// Effects' Sobel kernels, cut short at the edges of x0, y0 to x1, y1.
func sobel(alpha func(x, y int) float64, x, y, x0, y0, x1, y1 int) (float64, float64) {
	l, r := max(x-1, x0), min(x+1, x1-1)
	t, b := max(y-1, y0), min(y+1, y1-1)
	diff := func(along bool) float64 {
		lo, hi := l, r
		if !along {
			lo, hi = t, b
		}
		if hi == lo {
			return 0
		}
		sum, weight := 0.0, 0.0
		for o := -1; o <= 1; o++ {
			w := 2.0
			if o != 0 {
				w = 1
			}
			if along {
				row := y + o
				if row < y0 || row >= y1 {
					continue
				}
				sum += w * (alpha(hi, row) - alpha(lo, row))
			} else {
				col := x + o
				if col < x0 || col >= x1 {
					continue
				}
				sum += w * (alpha(col, hi) - alpha(col, lo))
			}
			weight += w
		}
		return sum * 2 / (weight * float64(hi-lo))
	}
	return diff(true), diff(false)
}

func sub3(a, b Point3) Point3     { return Point3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func dot3(a, b Point3) float64    { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func unit3(a Point3) Point3 {
	l := math.Sqrt(dot3(a, a))
	if l == 0 {
		return a
	}
	return Point3{a.X / l, a.Y / l, a.Z / l}
}
