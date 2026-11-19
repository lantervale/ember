package svg

import (
	"image"
	"math"
)

// Rasterise draws the picture into a new image of w×h pixels, its view
// box fitted as its preserveAspectRatio asks. The image is the straight
// input of a texture: 8-bit sRGB, alpha premultiplied, clear where the
// picture draws nothing. A caller drawing for a screen of scale s asks
// for s times the size in points. Drawing leaves the picture as it was,
// so one picture may be drawn from several goroutines at once.
func (p *Picture) Rasterise(w, h int) (*image.RGBA, error) { return p.RasteriseScaled(w, h, 1) }

// RasteriseScaled is Rasterise for a screen of scale pixels a CSS pixel,
// w and h still in pixels: a picture with no view box keeps a unit a
// CSS pixel along an axis it gives no size for, so it is scale pixels
// there, where Rasterise draws it one.
func (p *Picture) RasteriseScaled(w, h int, scale float64) (*image.RGBA, error) {
	if w <= 0 || h <= 0 || w > MaxSide || h > MaxSide || w*h > MaxPixels {
		return nil, ErrTooLarge
	}
	if !(scale > 0) || math.IsInf(scale, 0) {
		scale = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	p.draw(dst, dst.Bounds(), scale)
	return dst, nil
}

// Draw draws the picture over what dst holds, fitted into r as Rasterise
// fits it into its image, and nothing outside r.
func (p *Picture) Draw(dst *image.RGBA, r image.Rectangle) { p.draw(dst, r, 1) }

func (p *Picture) draw(dst *image.RGBA, r image.Rectangle, unit float64) {
	clip := r.Intersect(dst.Bounds())
	if clip.Empty() || r.Dx() > MaxSide || r.Dy() > MaxSide {
		return
	}
	if p.hasWidth && p.width == 0 || p.hasHeight && p.height == 0 {
		return
	}
	view := translate(float64(r.Min.X), float64(r.Min.Y)).Mul(p.viewTransform(float64(r.Dx()), float64(r.Dy()), unit))
	d := &drawer{view: view}
	d.items(dst, clip, p.Items, 1, 0)
}

// drawer draws items through one view, and keeps what it allocates from
// one shape to the next.
type drawer struct {
	view   Matrix
	cover  coverage
	shader shader
	layers []*image.RGBA
}

func (d *drawer) items(dst *image.RGBA, clip image.Rectangle, items []Item, fade float64, depth int) {
	for _, it := range items {
		switch {
		case it.Shape != nil:
			d.shape(dst, clip, it.Shape, fade)
		case it.Group == nil:
		case depth >= maxLayers:
			d.items(dst, clip, it.Group.Items, fade*it.Group.Opacity, depth)
		default:
			box := d.bounds(it.Group.Items).Intersect(clip)
			if box.Empty() {
				continue
			}
			layer := d.layer(depth, box)
			d.items(layer, box, it.Group.Items, 1, depth+1)
			composite(dst, layer, fade*it.Group.Opacity)
		}
	}
}

// layer is a clear image over box, the one of its depth reused.
func (d *drawer) layer(depth int, box image.Rectangle) *image.RGBA {
	for len(d.layers) <= depth {
		d.layers = append(d.layers, &image.RGBA{})
	}
	l := d.layers[depth]
	n := 4 * box.Dx() * box.Dy()
	if cap(l.Pix) < n {
		l.Pix = make([]uint8, n)
	} else {
		l.Pix = l.Pix[:n]
		clear(l.Pix)
	}
	l.Stride, l.Rect = 4*box.Dx(), box
	return l
}

// composite draws layer over dst through opacity.
func composite(dst, layer *image.RGBA, opacity float64) {
	o := uint32(opacity*255 + 0.5)
	b := layer.Rect
	for y := b.Min.Y; y < b.Max.Y; y++ {
		src := layer.Pix[layer.PixOffset(b.Min.X, y):][:4*b.Dx()]
		out := dst.Pix[dst.PixOffset(b.Min.X, y):][:4*b.Dx()]
		for i := 0; i < len(src); i += 4 {
			a := uint32(src[i+3]) * o
			if a == 0 {
				continue
			}
			keep := 255*255 - a
			for c := range 4 {
				mixed := (uint32(src[i+c])*o*255 + uint32(out[i+c])*keep + 255*255/2) / (255 * 255)
				out[i+c] = uint8(mixed) //nolint:gosec // a mix of two bytes is a byte
			}
		}
	}
}

// reach is how far, in the path's units, a shape's stroke goes beyond
// its path.
func (s *Shape) reach() float64 {
	if s.StrokeWidth <= 0 {
		return 0
	}
	far := 1.0
	if s.Join == MiterJoin {
		far = math.Max(far, s.MiterLimit)
	}
	if s.Cap == SquareCap {
		far = math.Max(far, math.Sqrt2)
	}
	return s.StrokeWidth / 2 * far
}

// shapeBounds is the pixels a shape may touch through m.
func shapeBounds(s *Shape, m Matrix) image.Rectangle {
	lo, hi := Point{math.Inf(1), math.Inf(1)}, Point{math.Inf(-1), math.Inf(-1)}
	for _, p := range s.Path.Points {
		lo = Point{math.Min(lo.X, p.X), math.Min(lo.Y, p.Y)}
		hi = Point{math.Max(hi.X, p.X), math.Max(hi.Y, p.Y)}
	}
	r := s.reach()
	lo, hi = lo.sub(Point{r, r}), hi.add(Point{r, r})
	dlo, dhi := Point{math.Inf(1), math.Inf(1)}, Point{math.Inf(-1), math.Inf(-1)}
	for _, corner := range [...]Point{lo, {hi.X, lo.Y}, hi, {lo.X, hi.Y}} {
		q := m.Apply(corner)
		dlo = Point{math.Min(dlo.X, q.X), math.Min(dlo.Y, q.Y)}
		dhi = Point{math.Max(dhi.X, q.X), math.Max(dhi.Y, q.Y)}
	}
	if !dlo.finite() || !dhi.finite() {
		return image.Rectangle{}
	}
	limit := func(v float64) int { return int(math.Max(-1<<30, math.Min(1<<30, v))) }
	return image.Rect(limit(math.Floor(dlo.X)), limit(math.Floor(dlo.Y)), limit(math.Ceil(dhi.X)), limit(math.Ceil(dhi.Y)))
}

func (d *drawer) bounds(items []Item) image.Rectangle {
	var r image.Rectangle
	for _, it := range items {
		if it.Shape != nil {
			r = r.Union(shapeBounds(it.Shape, d.view.Mul(it.Shape.Transform)))
		} else if it.Group != nil {
			r = r.Union(d.bounds(it.Group.Items))
		}
	}
	return r
}

// sound reports a path whose verbs have the points they take: any path
// this package parsed, and what Rasterise asks of one made by hand.
func (p Path) sound() bool {
	need := 0
	for _, v := range p.Verbs {
		switch v {
		case MoveTo, LineTo:
			need++
		case QuadTo:
			need += 2
		case CubicTo:
			need += 3
		case Close:
		default:
			return false
		}
	}
	return need == len(p.Points) && (len(p.Verbs) == 0 || p.Verbs[0] == MoveTo)
}

// dashes are the shape's dashes when they are a pattern to cut along:
// an even count of lengths, none negative, not all zero.
func (s *Shape) dashes() []float64 {
	sum := 0.0
	for _, d := range s.Dashes {
		if !(d >= 0) || math.IsInf(d, 0) {
			return nil
		}
		sum += d
	}
	if len(s.Dashes)%2 != 0 || sum == 0 {
		return nil
	}
	return s.Dashes
}

func (d *drawer) shape(dst *image.RGBA, clip image.Rectangle, s *Shape, fade float64) {
	m := d.view.Mul(s.Transform)
	if !s.Path.sound() || !m.finite() {
		return
	}
	clip = clip.Intersect(shapeBounds(s, m))
	if clip.Empty() {
		return
	}
	scale := m.stretch()
	stroked := s.StrokeWidth > 0 && s.Stroke.Kind != NoPaint
	reach := 0.0
	if stroked {
		reach = s.StrokeWidth / 2 * scale
	}
	subs := flatten(s.Path, scale, reach)
	fill := func() {
		if s.Fill.Kind == NoPaint {
			return
		}
		d.cover.reset(clip)
		for _, sub := range subs {
			d.cover.add(sub.pts, m)
		}
		d.paint(dst, s.Fill, m, fade, s.FillRule == EvenOdd)
	}
	stroke := func() {
		if !stroked {
			return
		}
		lines := subs
		if pattern := s.dashes(); pattern != nil && !math.IsNaN(s.DashOffset) && !math.IsInf(s.DashOffset, 0) {
			lines = dash(subs, pattern, s.DashOffset)
		}
		d.cover.reset(clip)
		for _, poly := range strokeOutline(lines, s, scale) {
			d.cover.add(poly, m)
		}
		d.paint(dst, s.Stroke, m, fade, false)
	}
	if s.StrokeFirst {
		stroke()
		fill()
	} else {
		fill()
		stroke()
	}
}

// paint fills what the coverage holds with p, whose gradient is in the
// space m takes to pixels.
func (d *drawer) paint(dst *image.RGBA, p Paint, m Matrix, fade float64, evenOdd bool) {
	opacity := float32(p.Opacity * fade)
	if p.Kind == GradientPaint {
		if !d.shader.set(p.Gradient, m, opacity) {
			return
		}
		d.cover.fill(evenOdd, func(y, x0 int, cover []float32) {
			d.shader.row(dst.Pix[dst.PixOffset(x0, y):], x0, y, cover)
		})
		return
	}
	a := float32(p.Color.A) / 255 * opacity
	if a <= 0 {
		return
	}
	// The colour premultiplied, 0–255.
	src := [4]float32{float32(p.Color.R) * a, float32(p.Color.G) * a, float32(p.Color.B) * a, 255 * a}
	opaque := [4]uint8{p.Color.R, p.Color.G, p.Color.B, 255}
	d.cover.fill(evenOdd, func(y, x0 int, cover []float32) {
		pix := dst.Pix[dst.PixOffset(x0, y):]
		for i, c := range cover {
			if c <= 0 {
				continue
			}
			px := pix[4*i : 4*i+4 : 4*i+4]
			if c >= 0.999 && a >= 0.999 {
				copy(px, opaque[:])
				continue
			}
			blend(px, &src, c)
		}
	})
}

// blend draws src, premultiplied 0–255, over the pixel px through cover.
func blend(px []uint8, src *[4]float32, cover float32) {
	keep := 1 - cover*src[3]/255
	a := uint8(src[3]*cover + float32(px[3])*keep + 0.5)
	px[0] = min(uint8(src[0]*cover+float32(px[0])*keep+0.5), a)
	px[1] = min(uint8(src[1]*cover+float32(px[1])*keep+0.5), a)
	px[2] = min(uint8(src[2]*cover+float32(px[2])*keep+0.5), a)
	px[3] = a
}
