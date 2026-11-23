package draw

import (
	"image"
	"math"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// obliqueSkew is how far a made-up italic leans: a quarter of a point to
// the right for each point up, as Chrome slants it.
const obliqueSkew = 0.25

// fakeBold is Skia's stroke for a made-up bold, as a fraction of the
// size: 1/24 at 9 px and below, 1/32 at 36 px and above, and between
// them in proportion. Chrome's text on macOS grows by a quarter of it on
// each side, what synthetic does, fitted to its pixels.
func fakeBold(size float32) float32 {
	const lo, hi = 9, 36
	t := min(max((size-lo)/(hi-lo), 0), 1)
	return size * (1.0/24 + (1.0/32-1.0/24)*t)
}

// boldRing are the directions a made-up bold's glyph is drawn again in,
// round its outline, to grow it on every side.
const boldRing = 16

// synthetic rasterizes glyph idx of source face at x, a fraction of a
// pixel to the right, slanted and grown as l has it: the glyph's
// rectangle about the origin and its coverage from (0, 0).
func (f *Font) synthetic(face int, idx sfnt.GlyphIndex, x fixed.Int26_6, l Look) (image.Rectangle, *image.Alpha, bool) {
	size := f.size * l.scale()
	ppem := fixed.Int26_6(math.Round(float64(size * oversample * 64)))
	segs, err := f.sources[face].LoadGlyph(&f.buf, idx, ppem, nil)
	if err != nil || len(segs) == 0 {
		return image.Rectangle{}, nil, false
	}
	pt := func(p fixed.Point26_6) [2]float32 {
		px, py := float32(p.X+x)/64, float32(p.Y)/64
		if l.Oblique {
			px -= obliqueSkew * py
		}
		return [2]float32{px, py}
	}
	var r float32
	if l.Bold {
		r = fakeBold(size) / 4 * oversample
	}
	lo := [2]float32{math.MaxFloat32, math.MaxFloat32}
	hi := [2]float32{-math.MaxFloat32, -math.MaxFloat32}
	for _, s := range segs {
		for _, a := range s.Args[:segmentArgs(s.Op)] {
			p := pt(a)
			lo = [2]float32{min(lo[0], p[0]), min(lo[1], p[1])}
			hi = [2]float32{max(hi[0], p[0]), max(hi[1], p[1])}
		}
	}
	dr := image.Rect(int(math.Floor(float64(lo[0]-r))), int(math.Floor(float64(lo[1]-r))),
		int(math.Ceil(float64(hi[0]+r))), int(math.Ceil(float64(hi[1]+r))))
	if dr.Empty() {
		return image.Rectangle{}, nil, false
	}
	mask := image.NewAlpha(image.Rect(0, 0, dr.Dx(), dr.Dy()))
	var rast vector.Rasterizer
	one := image.NewAlpha(mask.Rect)
	trace := func(dx, dy float32) {
		ox, oy := dx-float32(dr.Min.X), dy-float32(dr.Min.Y)
		rast.Reset(dr.Dx(), dr.Dy())
		for _, s := range segs {
			var p [3][2]float32
			for i, a := range s.Args[:segmentArgs(s.Op)] {
				q := pt(a)
				p[i] = [2]float32{q[0] + ox, q[1] + oy}
			}
			switch s.Op {
			case sfnt.SegmentOpMoveTo:
				rast.MoveTo(p[0][0], p[0][1])
			case sfnt.SegmentOpLineTo:
				rast.LineTo(p[0][0], p[0][1])
			case sfnt.SegmentOpQuadTo:
				rast.QuadTo(p[0][0], p[0][1], p[1][0], p[1][1])
			case sfnt.SegmentOpCubeTo:
				rast.CubeTo(p[0][0], p[0][1], p[1][0], p[1][1], p[2][0], p[2][1])
			}
		}
		rast.ClosePath()
		clear(one.Pix)
		rast.Draw(one, one.Rect, image.Opaque, image.Point{})
		for i, v := range one.Pix {
			mask.Pix[i] = max(mask.Pix[i], v)
		}
	}
	trace(0, 0)
	if r > 0 {
		for i := range boldRing {
			a := 2 * math.Pi * float64(i) / boldRing
			trace(r*float32(math.Cos(a)), r*float32(math.Sin(a)))
		}
	}
	return dr, mask, true
}

// segmentArgs is how many points a segment of op takes.
func segmentArgs(op sfnt.SegmentOp) int {
	switch op {
	case sfnt.SegmentOpQuadTo:
		return 2
	case sfnt.SegmentOpCubeTo:
		return 3
	case sfnt.SegmentOpMoveTo, sfnt.SegmentOpLineTo:
	}
	return 1
}
