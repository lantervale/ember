package svg

import (
	"image"
	"image/color"
	"math"
)

const (
	// maxReach is the furthest, in pixels, a filter is followed moving or
	// spreading a picture.
	maxReach = MaxSide
	// maxFilterPixels is the most pixels a filter runs over; a larger one
	// is not run.
	maxFilterPixels = 1 << 22
)

// fbuf is a picture a filter works on: premultiplied colours, 0–1, four
// to a pixel over the filter's box, in linear light or encoded sRGB.
type fbuf struct {
	pix    []float32
	linear bool
	// area is where it may be other than clear.
	area image.Rectangle
}

// pixelArea is the pixels the box r touches through m.
func pixelArea(r Box, m Matrix) image.Rectangle {
	lo, hi := Point{math.Inf(1), math.Inf(1)}, Point{math.Inf(-1), math.Inf(-1)}
	for _, c := range [...]Point{{r.X, r.Y}, {r.X + r.W, r.Y}, {r.X, r.Y + r.H}, {r.X + r.W, r.Y + r.H}} {
		q := m.Apply(c)
		lo = Point{math.Min(lo.X, q.X), math.Min(lo.Y, q.Y)}
		hi = Point{math.Max(hi.X, q.X), math.Max(hi.Y, q.Y)}
	}
	if !lo.finite() || !hi.finite() {
		return image.Rectangle{}
	}
	limit := func(v float64) int { return int(math.Max(-1<<30, math.Min(1<<30, v))) }
	return image.Rect(limit(math.Floor(lo.X)), limit(math.Floor(lo.Y)), limit(math.Ceil(hi.X)), limit(math.Ceil(hi.Y)))
}

// scales are how many pixels a unit of the filter's space is along x
// and y through m.
func scales(m Matrix) Point { return Point{math.Hypot(m.A, m.B), math.Hypot(m.C, m.D)} }

// moved is the offset p of the filter's space in pixels through m.
func moved(p Point, m Matrix) Point { return Point{m.A*p.X + m.C*p.Y, m.B*p.X + m.D*p.Y} }

// reach is how far, in pixels, f may move or spread what it reads.
func (f *Filter) reach(m Matrix) int {
	s := scales(m)
	total := 0.0
	for i := range f.Primitives {
		p := &f.Primitives[i]
		switch p.Kind {
		case GaussianBlur, DropShadow:
			total += 3 * math.Max(p.StdDev.X*s.X, p.StdDev.Y*s.Y)
			total += moved(p.Offset, m).len()
		case Offset:
			total += moved(p.Offset, m).len()
		case Morphology:
			total += math.Max(p.Radius.X*s.X, p.Radius.Y*s.Y)
		case DisplacementMap:
			total += math.Abs(p.Scale) / 2 * math.Max(s.X, s.Y)
		case ConvolveMatrix:
			total += float64(max(p.Convolve.Order[0], p.Convolve.Order[1]))
		case Clear, Flood, ColorMatrix, ComponentTransfer, Composite, Merge, Blend, Tile, Turbulence,
			DiffuseLighting, SpecularLighting, FeImage:
		}
	}
	if !(total < maxReach) {
		return maxReach
	}
	return int(math.Ceil(total)) + 1
}

// filterArea is the pixels a group's filters may draw, what its items
// draw being from.
func (d *drawer) filterArea(filters []*Filter, from image.Rectangle) image.Rectangle {
	for _, f := range filters {
		m := d.view.Mul(f.Transform)
		if f.Region.Empty() {
			r := f.reach(m)
			from = from.Inset(-r)
		} else {
			from = pixelArea(f.Region, m)
		}
	}
	return from
}

// filtered is a layer holding g's items run through its filters, over
// what of clip they draw; nil when nothing shows.
func (d *drawer) filtered(g *Group, clip image.Rectangle, depth int) *image.RGBA {
	if own, turned := d.ownSpace(g); turned {
		return d.filteredTurned(g, clip, depth, own)
	}
	reach := 0
	for _, f := range g.Filters {
		reach = min(maxReach, reach+f.reach(d.view.Mul(f.Transform)))
	}
	area := d.bounds(g.Items)
	work := area
	for _, f := range g.Filters {
		m := d.view.Mul(f.Transform)
		if f.Region.Empty() {
			area = area.Inset(-f.reach(m))
		} else {
			area = pixelArea(f.Region, m)
		}
		work = work.Union(area.Inset(-reach))
	}
	work = work.Intersect(clip.Inset(-reach))
	if work.Empty() {
		return nil
	}
	if work.Dx() > MaxSide || work.Dy() > MaxSide || work.Dx()*work.Dy() > maxFilterPixels {
		if work = d.bounds(g.Items).Intersect(clip); work.Empty() {
			return nil
		}
		layer := d.layer(depth, work)
		d.items(layer, work, g.Items, 1, depth+1)
		return layer
	}
	layer := d.layer(depth, work)
	d.items(layer, work, g.Items, 1, depth+1)
	for _, f := range g.Filters {
		d.run(layer, f)
	}
	shown := work.Intersect(clip)
	if shown.Empty() {
		return nil
	}
	return &image.RGBA{Pix: layer.Pix[layer.PixOffset(shown.Min.X, shown.Min.Y):], Stride: layer.Stride, Rect: shown}
}

// ownSpace is the view drawing g's filtered element in its own space,
// scaled to the picture's pixels but neither turned nor skewed; turned
// is false when the element's space is not.
func (d *drawer) ownSpace(g *Group) (Matrix, bool) {
	t := g.Filters[0].Transform
	m := d.view.Mul(t)
	inv, ok := t.Inverse()
	if math.Abs(m.B) < 1e-9 && math.Abs(m.C) < 1e-9 || !ok || !m.finite() {
		return Matrix{}, false
	}
	s := scales(m)
	return scale(s.X, s.Y).Mul(inv), true
}

// filteredTurned is filtered for an element turned or skewed: its
// filters run in its own space under the view own, as Chrome runs them,
// and their result is drawn turned into a layer over the picture's
// pixels.
func (d *drawer) filteredTurned(g *Group, clip image.Rectangle, depth int, own Matrix) *image.RGBA {
	back, _ := own.Inverse()
	toOwn, ok := d.view.Mul(back).Inverse()
	if !ok {
		return nil
	}
	sub := &drawer{view: own, unit: d.unit}
	area := Box{float64(clip.Min.X), float64(clip.Min.Y), float64(clip.Dx()), float64(clip.Dy())}
	src := sub.filtered(g, pixelArea(area, toOwn), depth)
	if src == nil {
		return nil
	}
	im := &Image{Pix: src, Transform: back.Mul(translate(float64(src.Rect.Min.X), float64(src.Rect.Min.Y))), Opacity: 1}
	box := imageBounds(im, d.view.Mul(im.Transform)).Intersect(clip)
	if box.Empty() {
		return nil
	}
	layer := d.layer(depth, box)
	d.image(layer, box, im, 1)
	return layer
}

// run makes the picture of layer anew through f.
func (d *drawer) run(layer *image.RGBA, f *Filter) {
	m := d.view.Mul(f.Transform)
	box := layer.Rect
	fr := &filterRun{box: box, w: box.Dx(), m: m, view: d.view, unit: d.unit, results: make([]*fbuf, len(f.Primitives))}
	fr.src = &fbuf{pix: make([]float32, 4*box.Dx()*box.Dy()), area: box}
	for y := range box.Dy() {
		row := layer.Pix[y*layer.Stride:][:4*box.Dx()]
		out := fr.src.pix[4*y*box.Dx():]
		for i, v := range row {
			out[i] = float32(v) / 255
		}
	}
	lastUse := make([]int, len(f.Primitives))
	for i := range f.Primitives {
		for _, in := range f.Primitives[i].In {
			if in >= 0 && int(in) < i {
				lastUse[in] = i
			}
		}
	}
	last := fr.src
	for i := range f.Primitives {
		last = fr.primitive(&f.Primitives[i], f)
		fr.results[i] = last
		for k := range i {
			if lastUse[k] <= i {
				fr.results[k] = nil
				delete(fr.converted, k)
			}
		}
	}
	last = fr.convert(last, false)
	for y := range box.Dy() {
		row := layer.Pix[y*layer.Stride:][:4*box.Dx()]
		in := last.pix[4*y*box.Dx():]
		for i := 0; i < len(row); i += 4 {
			a := min(max(in[i+3], 0), 1)
			row[i+3] = uint8(a*255 + 0.5)
			for c := range 3 {
				row[i+c] = uint8(min(max(in[i+c], 0), a)*255 + 0.5)
			}
		}
	}
	if f.Region.Empty() {
		return
	}
	if m.B != 0 || m.C != 0 {
		applyCover(layer, d.clipCover(rectClip(f.Region, f.Transform), box))
		return
	}
	region := pixelArea(f.Region, m)
	for y := box.Min.Y; y < box.Max.Y; y++ {
		row := layer.Pix[(y-box.Min.Y)*layer.Stride:][:4*box.Dx()]
		if y < region.Min.Y || y >= region.Max.Y {
			clear(row)
			continue
		}
		clear(row[:4*min(box.Dx(), max(0, region.Min.X-box.Min.X))])
		clear(row[4*max(0, min(box.Dx(), region.Max.X-box.Min.X)):])
	}
}

// filterRun is one filter running over a box of pixels.
type filterRun struct {
	box image.Rectangle
	w   int
	m   Matrix
	// view and unit are the drawer's, what an feImage draws through.
	view    Matrix
	unit    float64
	src     *fbuf
	srcLin  *fbuf
	alpha   *fbuf
	results []*fbuf
	// converted are results in the other colour space, by index.
	converted map[int]*fbuf
}

func (fr *filterRun) blank(linear bool, area image.Rectangle) *fbuf {
	return &fbuf{pix: make([]float32, 4*fr.w*fr.box.Dy()), linear: linear, area: area.Intersect(fr.box)}
}

// ownSpace is whether what in names is in linear light as it was made.
func (fr *filterRun) ownSpace(in Input) bool {
	if in >= 0 && int(in) < len(fr.results) && fr.results[in] != nil {
		return fr.results[in].linear
	}
	return false
}

// drawn is items drawn over the run's box, encoded in sRGB.
func (fr *filterRun) drawn(items []Item, area image.Rectangle) *fbuf {
	out := fr.blank(false, area)
	if len(items) == 0 || out.area.Empty() {
		return out
	}
	layer := image.NewRGBA(fr.box)
	sub := &drawer{view: fr.view, unit: fr.unit}
	sub.items(layer, out.area, items, 1, 0)
	for i, v := range layer.Pix {
		out.pix[i] = float32(v) / 255
	}
	return out
}

// input is what in names, in linear light or not.
func (fr *filterRun) input(in Input, linear bool) *fbuf {
	switch {
	case in == SourceGraphic:
		if !linear {
			return fr.src
		}
		if fr.srcLin == nil {
			fr.srcLin = fr.convert(fr.src, true)
		}
		return fr.srcLin
	case in == SourceAlpha:
		if fr.alpha == nil {
			fr.alpha = fr.blank(false, fr.box)
			for i := 3; i < len(fr.src.pix); i += 4 {
				fr.alpha.pix[i] = fr.src.pix[i]
			}
		}
		return fr.alpha
	case in < 0 || int(in) >= len(fr.results) || fr.results[in] == nil:
		return fr.blank(linear, image.Rectangle{})
	}
	r := fr.results[in]
	if r.linear == linear {
		return r
	}
	if fr.converted == nil {
		fr.converted = map[int]*fbuf{}
	}
	if c, ok := fr.converted[int(in)]; ok {
		return c
	}
	c := fr.convert(r, linear)
	fr.converted[int(in)] = c
	return c
}

func toLinear(c float32) float32 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return float32(math.Pow((float64(c)+0.055)/1.055, 2.4))
}

func toSRGB(c float32) float32 {
	if c <= 0.0031308 {
		return c * 12.92
	}
	return float32(1.055*math.Pow(float64(c), 1/2.4) - 0.055)
}

// convert is b in linear light, or encoded in sRGB.
func (fr *filterRun) convert(b *fbuf, linear bool) *fbuf {
	if b.linear == linear {
		return b
	}
	out := &fbuf{pix: make([]float32, len(b.pix)), linear: linear, area: b.area}
	f := toSRGB
	if linear {
		f = toLinear
	}
	for i := 0; i < len(b.pix); i += 4 {
		a := b.pix[i+3]
		if a <= 0 {
			continue
		}
		out.pix[i+3] = a
		for c := range 3 {
			out.pix[i+c] = f(min(b.pix[i+c]/a, 1)) * a
		}
	}
	return out
}

// area is the pixels p's region covers.
func (fr *filterRun) area(p *Primitive) image.Rectangle {
	if p.Region.Empty() {
		return fr.box
	}
	return pixelArea(p.Region, fr.m).Intersect(fr.box)
}

func (fr *filterRun) primitive(p *Primitive, f *Filter) *fbuf {
	area := fr.area(p)
	in := func(i int) *fbuf {
		if i < len(p.In) {
			return fr.input(p.In[i], p.LinearRGB)
		}
		return fr.blank(p.LinearRGB, image.Rectangle{})
	}
	out := fr.blank(p.LinearRGB, area)
	s := scales(fr.m)
	switch p.Kind {
	case Clear:
	case GaussianBlur:
		copy(out.pix, in(0).pix)
		fr.blur(out.pix, p.StdDev.X*s.X, p.StdDev.Y*s.Y)
	case Offset:
		fr.offset(out.pix, in(0).pix, moved(p.Offset, fr.m))
	case Flood:
		c := premultiplied(p.Color, p.LinearRGB)
		for i := 0; i < len(out.pix); i += 4 {
			copy(out.pix[i:i+4], c[:])
		}
	case ColorMatrix:
		mapColours(out.pix, in(0).pix, func(c *[4]float32) { applyMatrix(c, &p.Matrix) })
	case ComponentTransfer:
		mapColours(out.pix, in(0).pix, func(c *[4]float32) {
			for k := range c {
				c[k] = p.Transfer[k].apply(c[k])
			}
		})
	case Composite:
		composeBuffers(out.pix, in(0).pix, in(1).pix, p.Operator, p.K)
	case Merge:
		for i := range p.In {
			composeBuffers(out.pix, in(i).pix, out.pix, CompositeOver, [4]float64{})
		}
	case Blend:
		blendBuffers(out.pix, in(0).pix, in(1).pix, p.Mode)
	case DropShadow:
		fr.dropShadow(out.pix, in(0).pix, p)
	case Morphology:
		copy(out.pix, in(0).pix)
		if p.Radius.X >= 0 && p.Radius.Y >= 0 {
			fr.morphology(out.pix, int(math.Round(p.Radius.X*s.X)), int(math.Round(p.Radius.Y*s.Y)), p.Dilate)
		}
	case Tile:
		src := in(0)
		tile := fr.box
		if p.In[0] >= 0 {
			tile = fr.area(&f.Primitives[p.In[0]])
		} else if !f.Region.Empty() {
			tile = pixelArea(f.Region, fr.m).Intersect(fr.box)
		}
		fr.tile(out.pix, src.pix, tile)
	case Turbulence:
		fr.turbulence(out, p)
	case ConvolveMatrix:
		fr.convolve(out, in(0), p)
	case DisplacementMap:
		out.linear = fr.ownSpace(p.In[0])
		fr.displace(out, fr.input(p.In[0], out.linear), in(1), p)
	case DiffuseLighting, SpecularLighting:
		fr.light(out, in(0), p)
	case FeImage:
		out = fr.convert(fr.drawn(p.Items, area), p.LinearRGB)
	}
	fr.crop(out)
	return out
}

// crop clears b outside its area.
func (fr *filterRun) crop(b *fbuf) {
	a := b.area
	for y := fr.box.Min.Y; y < fr.box.Max.Y; y++ {
		row := b.pix[4*(y-fr.box.Min.Y)*fr.w:][:4*fr.w]
		if y < a.Min.Y || y >= a.Max.Y {
			clear(row)
			continue
		}
		clear(row[:4*max(0, a.Min.X-fr.box.Min.X)])
		clear(row[4*min(fr.w, a.Max.X-fr.box.Min.X):])
	}
}

func premultiplied(c color.NRGBA, linear bool) [4]float32 {
	a := float32(c.A) / 255
	out := [4]float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255, a}
	for k := range 3 {
		if linear {
			out[k] = toLinear(out[k])
		}
		out[k] *= a
	}
	return out
}

// mapColours writes each pixel of in through fn, not premultiplied, to
// out.
func mapColours(out, in []float32, fn func(c *[4]float32)) {
	for i := 0; i < len(in); i += 4 {
		var c [4]float32
		if a := in[i+3]; a > 0 {
			c = [4]float32{in[i] / a, in[i+1] / a, in[i+2] / a, a}
		}
		fn(&c)
		a := min(max(c[3], 0), 1)
		out[i+3] = a
		for k := range 3 {
			out[i+k] = min(max(c[k], 0), 1) * a
		}
	}
}

func applyMatrix(c *[4]float32, m *[20]float64) {
	var out [4]float32
	for r := range 4 {
		row := m[5*r : 5*r+5]
		out[r] = float32(row[0]*float64(c[0]) + row[1]*float64(c[1]) + row[2]*float64(c[2]) + row[3]*float64(c[3]) + row[4])
	}
	*c = out
}

func (t *Transfer) apply(c float32) float32 {
	v := float64(min(max(c, 0), 1))
	switch t.Kind {
	case TransferIdentity:
		return c
	case TransferTable:
		n := len(t.Table) - 1
		if n == 0 || v >= 1 {
			return float32(t.Table[n])
		}
		k := int(v * float64(n))
		return float32(t.Table[k] + (v*float64(n)-float64(k))*(t.Table[k+1]-t.Table[k]))
	case TransferDiscrete:
		n := len(t.Table)
		return float32(t.Table[min(int(v*float64(n)), n-1)])
	case TransferLinear:
		return float32(t.Slope*v + t.Intercept)
	case TransferGamma:
		return float32(t.Amplitude*math.Pow(v, t.Exponent) + t.Shift)
	}
	return c
}

// composeBuffers writes a put with b by op, both premultiplied, to out,
// which may be b.
func composeBuffers(out, a, b []float32, op CompositeOperator, k [4]float64) {
	k1, k2, k3, k4 := float32(k[0]), float32(k[1]), float32(k[2]), float32(k[3])
	for i := 0; i < len(out); i += 4 {
		aa, ba := a[i+3], b[i+3]
		var fa, fb float32
		switch op {
		case CompositeOver:
			fa, fb = 1, 1-aa
		case CompositeIn:
			fa, fb = ba, 0
		case CompositeOut:
			fa, fb = 1-ba, 0
		case CompositeAtop:
			fa, fb = ba, 1-aa
		case CompositeXor:
			fa, fb = 1-ba, 1-aa
		case CompositeLighter:
			for c := range 4 {
				out[i+c] = min(a[i+c]+b[i+c], 1)
			}
			continue
		case CompositeArithmetic:
			var px [4]float32
			for c := range 4 {
				px[c] = min(max(k1*a[i+c]*b[i+c]+k2*a[i+c]+k3*b[i+c]+k4, 0), 1)
			}
			for c := range 3 {
				px[c] = min(px[c], px[3])
			}
			copy(out[i:i+4], px[:])
			continue
		}
		for c := range 4 {
			out[i+c] = a[i+c]*fa + b[i+c]*fb
		}
	}
}

// blendBuffers writes src blended over dst by mode to out.
func blendBuffers(out, src, dst []float32, mode BlendMode) {
	for i := 0; i < len(out); i += 4 {
		var s, b [4]float32
		copy(s[:], src[i:i+4])
		copy(b[:], dst[i:i+4])
		px := blendPixel(s, b, mode)
		copy(out[i:i+4], px[:])
	}
}

// offset writes in moved by d pixels to out, between pixels where d is
// not whole.
func (fr *filterRun) offset(out, in []float32, d Point) {
	h := fr.box.Dy()
	ix, iy := math.Floor(d.X), math.Floor(d.Y)
	tx, ty := float32(d.X-ix), float32(d.Y-iy)
	at := func(x, y int) []float32 {
		if x < 0 || y < 0 || x >= fr.w || y >= h {
			return nil
		}
		return in[4*(y*fr.w+x):][:4]
	}
	weights := [4]float32{(1 - tx) * (1 - ty), tx * (1 - ty), (1 - tx) * ty, tx * ty}
	for y := range h {
		for x := range fr.w {
			sx, sy := x-int(ix), y-int(iy)
			o := out[4*(y*fr.w+x):][:4]
			for k, p := range [4][]float32{at(sx, sy), at(sx-1, sy), at(sx, sy-1), at(sx-1, sy-1)} {
				if p == nil || weights[k] == 0 {
					continue
				}
				for c := range 4 {
					o[c] += p[c] * weights[k]
				}
			}
		}
	}
}

func (fr *filterRun) dropShadow(out, in []float32, p *Primitive) {
	shadow := make([]float32, len(in))
	for i := 3; i < len(in); i += 4 {
		shadow[i] = in[i]
	}
	s := scales(fr.m)
	fr.blur(shadow, p.StdDev.X*s.X, p.StdDev.Y*s.Y)
	shifted := make([]float32, len(in))
	fr.offset(shifted, shadow, moved(p.Offset, fr.m))
	c := premultiplied(p.Color, p.LinearRGB)
	for i := 0; i < len(out); i += 4 {
		a := shifted[i+3]
		for k := range 4 {
			out[i+k] = c[k] * a
		}
	}
	composeBuffers(out, in, out, CompositeOver, [4]float64{})
}

func (fr *filterRun) tile(out, in []float32, tile image.Rectangle) {
	if tile.Empty() {
		return
	}
	tw, th := tile.Dx(), tile.Dy()
	for y := range fr.box.Dy() {
		ty := ((y+fr.box.Min.Y-tile.Min.Y)%th+th)%th + tile.Min.Y - fr.box.Min.Y
		for x := range fr.w {
			tx := ((x+fr.box.Min.X-tile.Min.X)%tw+tw)%tw + tile.Min.X - fr.box.Min.X
			copy(out[4*(y*fr.w+x):][:4], in[4*(ty*fr.w+tx):][:4])
		}
	}
}
