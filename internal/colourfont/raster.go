package colourfont

import (
	"image"
	"math"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// affine maps x, y to xx·x + xy·y + dx, yx·x + yy·y + dy.
type affine struct{ xx, yx, xy, yy, dx, dy float32 }

func identity() affine { return affine{xx: 1, yy: 1} }

// then is b applied first, then a.
func (a affine) then(b affine) affine {
	return affine{
		xx: a.xx*b.xx + a.xy*b.yx, yx: a.yx*b.xx + a.yy*b.yx,
		xy: a.xx*b.xy + a.xy*b.yy, yy: a.yx*b.xy + a.yy*b.yy,
		dx: a.xx*b.dx + a.xy*b.dy + a.dx, dy: a.yx*b.dx + a.yy*b.dy + a.dy,
	}
}

func (a affine) apply(x, y float32) (float32, float32) {
	return a.xx*x + a.xy*y + a.dx, a.yx*x + a.yy*y + a.dy
}

func (a affine) invert() (affine, bool) {
	det := a.xx*a.yy - a.xy*a.yx
	if det == 0 || math.IsNaN(float64(det)) || math.IsInf(float64(det), 0) {
		return affine{}, false
	}
	i := affine{xx: a.yy / det, yx: -a.yx / det, xy: -a.xy / det, yy: a.xx / det}
	i.dx = -(i.xx*a.dx + i.xy*a.dy)
	i.dy = -(i.yx*a.dx + i.yy*a.dy)
	return i, true
}

func translate(x, y float32) affine { return affine{xx: 1, yy: 1, dx: x, dy: y} }

func scale(x, y float32) affine { return affine{xx: x, yy: y} }

// box is a rectangle in pixels, empty when not lo < hi.
type box struct{ x0, y0, x1, y1 float32 }

func emptyBox() box {
	inf := float32(math.Inf(1))
	return box{inf, inf, -inf, -inf}
}

func (b box) empty() bool { return !(b.x0 < b.x1 && b.y0 < b.y1) }

func (b box) add(x, y float32) box {
	return box{min(b.x0, x), min(b.y0, y), max(b.x1, x), max(b.y1, y)}
}

func (b box) union(o box) box {
	if o.empty() {
		return b
	}
	return b.add(o.x0, o.y0).add(o.x1, o.y1)
}

// corners is the rectangle x0, y0, x1, y1 through m, bounded.
func corners(x0, y0, x1, y1 float32, m affine) box {
	b := emptyBox()
	for _, p := range [4][2]float32{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}} {
		b = b.add(m.apply(p[0], p[1]))
	}
	return b
}

// layer is a picture being drawn: w×h premultiplied colours, four
// floats a pixel.
type layer struct {
	w, h int
	px   []float32
}

func newLayer(w, h int) *layer { return &layer{w: w, h: h, px: make([]float32, 4*w*h)} }

// over draws src over l.
func (l *layer) over(src *layer) {
	if src == nil {
		return
	}
	for i := 0; i < len(l.px); i += 4 {
		a := src.px[i+3]
		if a == 0 {
			continue
		}
		k := 1 - a
		l.px[i] = src.px[i] + l.px[i]*k
		l.px[i+1] = src.px[i+1] + l.px[i+1]*k
		l.px[i+2] = src.px[i+2] + l.px[i+2]*k
		l.px[i+3] = a + l.px[i+3]*k
	}
}

// masked keeps l where mask covers it.
func (l *layer) masked(mask *image.Alpha) {
	for y := range l.h {
		for x := range l.w {
			k := float32(mask.Pix[y*mask.Stride+x]) / 255
			i := 4 * (y*l.w + x)
			l.px[i] *= k
			l.px[i+1] *= k
			l.px[i+2] *= k
			l.px[i+3] *= k
		}
	}
}

// rgba is l as 8-bit premultiplied colours.
func (l *layer) rgba() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, l.w, l.h))
	for i, v := range l.px {
		img.Pix[i] = uint8(min(max(v, 0), 1)*255 + 0.5)
	}
	return img
}

// coverage rasterizes glyph gid's outline in font units through m into
// a w×h mask; nil when the glyph has none.
func (f *Font) coverage(gid uint16, m affine, w, h int) *image.Alpha {
	if f.outlines == nil {
		return nil
	}
	segs, err := f.outlines.LoadGlyph(&f.buf, sfnt.GlyphIndex(gid), fixed.I(int(f.upem)), nil)
	if err != nil || len(segs) == 0 {
		return nil
	}
	r := vector.NewRasterizer(w, h)
	pt := func(p fixed.Point26_6) (float32, float32) {
		return m.apply(float32(p.X)/64, -float32(p.Y)/64)
	}
	open := false
	var sx, sy float32
	for _, s := range segs {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			if open {
				r.ClosePath()
			}
			sx, sy = pt(s.Args[0])
			r.MoveTo(sx, sy)
			open = true
		case sfnt.SegmentOpLineTo:
			r.LineTo(pt(s.Args[0]))
		case sfnt.SegmentOpQuadTo:
			x1, y1 := pt(s.Args[0])
			x2, y2 := pt(s.Args[1])
			r.QuadTo(x1, y1, x2, y2)
		case sfnt.SegmentOpCubeTo:
			x1, y1 := pt(s.Args[0])
			x2, y2 := pt(s.Args[1])
			x3, y3 := pt(s.Args[2])
			r.CubeTo(x1, y1, x2, y2, x3, y3)
		}
	}
	if open {
		r.ClosePath()
	}
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	r.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	return mask
}

// outlineBox is glyph gid's outline through m, bounded; empty for none.
func (f *Font) outlineBox(gid uint16, m affine) box {
	b := emptyBox()
	if f.outlines == nil {
		return b
	}
	segs, err := f.outlines.LoadGlyph(&f.buf, sfnt.GlyphIndex(gid), fixed.I(int(f.upem)), nil)
	if err != nil {
		return b
	}
	for _, s := range segs {
		n := 1
		switch s.Op {
		case sfnt.SegmentOpQuadTo:
			n = 2
		case sfnt.SegmentOpCubeTo:
			n = 3
		case sfnt.SegmentOpMoveTo, sfnt.SegmentOpLineTo:
		}
		for _, p := range s.Args[:n] {
			b = b.add(m.apply(float32(p.X)/64, -float32(p.Y)/64))
		}
	}
	return b
}
