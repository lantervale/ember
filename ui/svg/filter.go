package svg

import (
	"github.com/lantervale/ember/color"
	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/svg"
	"github.com/lantervale/ember/ui/internal/fx"
)

// FilterGraph is the filter element id names in the picture, read for an
// element w×h points large outside it, as filter: url(#id) on it reads
// it; ok is false when the picture has none.
func (p picture) FilterGraph(id string, w, h float32) (*fx.Graph, bool) {
	f, ok := p.Filter(id, svg.Box{W: float64(w), H: float64(h)})
	if !ok {
		return nil, false
	}
	g := &fx.Graph{Region: rect(f.Region), Primitives: make([]fx.Primitive, len(f.Primitives))}
	for i, s := range f.Primitives {
		d := &g.Primitives[i]
		*d = fx.Primitive{
			Kind: fx.Kind(s.Kind), Region: rect(s.Region), LinearRGB: s.LinearRGB,
			StdDev: vec(s.StdDev), Offset: vec(s.Offset), Operator: fx.Operator(s.Operator),
			Mode: uint8(s.Mode), Radius: vec(s.Radius), Dilate: s.Dilate,
		}
		c := s.Color
		d.Color = color.FromSRGB([3]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255})
		d.Color.A = float32(c.A) / 255
		for _, in := range s.In {
			d.In = append(d.In, fx.Input(in))
		}
		for k := range d.Matrix {
			d.Matrix[k] = float32(s.Matrix[k])
		}
		for k := range d.K {
			d.K[k] = float32(s.K[k])
		}
		for k, t := range s.Transfer {
			o := fx.Transfer{Kind: fx.TransferKind(t.Kind), Slope: float32(t.Slope), Intercept: float32(t.Intercept),
				Amplitude: float32(t.Amplitude), Exponent: float32(t.Exponent), Shift: float32(t.Shift)}
			for _, v := range t.Table {
				o.Table = append(o.Table, float32(v))
			}
			d.Transfer[k] = o
		}
	}
	return g, true
}

func rect(b svg.Box) geom.Rect {
	return geom.Rect{X: float32(b.X), Y: float32(b.Y), W: float32(b.W), H: float32(b.H)}
}

func vec(p svg.Point) geom.Vec2 { return geom.V2(float32(p.X), float32(p.Y)) }
