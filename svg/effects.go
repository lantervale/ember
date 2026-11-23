package svg

import (
	"image"
	"math"
)

const (
	// maxEffectLayers is how deep groups with a clip or a mask are drawn;
	// deeper ones are not drawn at all.
	maxEffectLayers = 32
	// maxTileSide and maxTilePixels bound the raster of a pattern's tile.
	maxTileSide   = 2048
	maxTilePixels = 1 << 22
)

// clipBounds is the pixels a clip may keep.
func (d *drawer) clipBounds(c *Clip) image.Rectangle {
	var r image.Rectangle
	for _, s := range c.Shapes {
		r = r.Union(shapeBounds(s, d.view.Mul(s.Transform)))
	}
	if c.Clip != nil {
		r = r.Intersect(d.clipBounds(c.Clip))
	}
	return r
}

// clipCover is how much of each pixel of box the clip keeps, 0–1, row
// after row.
func (d *drawer) clipCover(c *Clip, box image.Rectangle) []float32 {
	w := box.Dx()
	out := make([]float32, w*box.Dy())
	for _, s := range c.Shapes {
		m := d.view.Mul(s.Transform)
		if !s.Path.sound() || !m.finite() {
			continue
		}
		r := box.Intersect(shapeBounds(s, m))
		if r.Empty() {
			continue
		}
		d.cover.reset(r)
		for _, sub := range flatten(s.Path, m.stretch(), 0) {
			d.cover.add(sub.pts, m)
		}
		d.cover.fill(s.FillRule == EvenOdd, func(y, x0 int, cover []float32) {
			row := out[(y-box.Min.Y)*w+x0-box.Min.X:]
			for i, v := range cover {
				row[i] = max(row[i], v)
			}
		})
	}
	if c.Clip != nil {
		inner := d.clipCover(c.Clip, box)
		for i := range out {
			out[i] *= inner[i]
		}
	}
	return out
}

// applyCover fades each pixel of layer by its cover.
func applyCover(layer *image.RGBA, cover []float32) {
	b := layer.Rect
	w := b.Dx()
	for y := range b.Dy() {
		pix := layer.Pix[y*layer.Stride:][:4*w]
		for x, c := range cover[y*w : (y+1)*w] {
			if c >= 1 {
				continue
			}
			px := pix[4*x : 4*x+4 : 4*x+4]
			for i := range px {
				px[i] = uint8(float32(px[i])*c + 0.5)
			}
		}
	}
}

// mask fades layer by what the mask's items draw over it: their
// luminance, or their alpha.
func (d *drawer) mask(layer *image.RGBA, mk *Mask, depth int) {
	box := layer.Rect
	ml := d.layer(depth, box)
	d.items(ml, box, mk.Items, 1, depth+1)
	cover := make([]float32, box.Dx()*box.Dy())
	for i := range cover {
		px := ml.Pix[4*i : 4*i+4 : 4*i+4]
		if mk.Alpha {
			cover[i] = float32(px[3]) / 255
		} else {
			cover[i] = (0.2125*float32(px[0]) + 0.7154*float32(px[1]) + 0.0721*float32(px[2])) / 255
		}
	}
	if mk.Region != nil {
		region := d.clipCover(mk.Region, box)
		for i := range cover {
			cover[i] *= region[i]
		}
	}
	applyCover(layer, cover)
}

// imageBounds is the pixels an image may touch through m.
func imageBounds(im *Image, m Matrix) image.Rectangle {
	if im.Pix == nil {
		return image.Rectangle{}
	}
	b := im.Pix.Rect
	return shapeBounds(&Shape{Path: rectPath(0, 0, float64(b.Dx()), float64(b.Dy()), 0, 0)}, m)
}

func (d *drawer) image(dst *image.RGBA, clip image.Rectangle, im *Image, fade float64) {
	m := d.view.Mul(im.Transform)
	opacity := float32(im.Opacity * fade)
	inv, ok := m.Inverse()
	if im.Pix == nil || !ok || !m.finite() || opacity <= 0 {
		return
	}
	clip = clip.Intersect(imageBounds(im, m))
	if clip.Empty() {
		return
	}
	w, h := float64(im.Pix.Rect.Dx()), float64(im.Pix.Rect.Dy())
	d.cover.reset(clip)
	d.cover.add([]Point{{0, 0}, {w, 0}, {w, h}, {0, h}}, m)
	d.cover.fill(false, func(y, x0 int, cover []float32) {
		pix := dst.Pix[dst.PixOffset(x0, y):]
		p := inv.Apply(Point{float64(x0) + 0.5, float64(y) + 0.5})
		step := Point{inv.A, inv.B}
		for i, c := range cover {
			if c > 0 {
				var src [4]float32
				if im.Pixelated {
					src = nearest(im.Pix, p)
				} else {
					src = bilinear(im.Pix, p, false)
				}
				for k := range src {
					src[k] *= opacity
				}
				blend(pix[4*i:4*i+4:4*i+4], &src, c)
			}
			p = p.add(step)
		}
	})
}

func nearest(img *image.RGBA, p Point) [4]float32 {
	b := img.Rect
	x := min(max(int(math.Floor(p.X)), 0), b.Dx()-1)
	y := min(max(int(math.Floor(p.Y)), 0), b.Dy()-1)
	px := img.Pix[y*img.Stride+4*x:]
	return [4]float32{float32(px[0]), float32(px[1]), float32(px[2]), float32(px[3])}
}

// bilinear samples img at p, in pixels from its corner, between its four
// nearest pixels: those past an edge are the edge's, or wrap round to
// the other side.
func bilinear(img *image.RGBA, p Point, wrap bool) [4]float32 {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	fx, fy := p.X-0.5, p.Y-0.5
	if math.IsNaN(fx+fy) || math.IsInf(fx+fy, 0) {
		return [4]float32{}
	}
	x0, y0 := math.Floor(fx), math.Floor(fy)
	tx, ty := float32(fx-x0), float32(fy-y0)
	at := func(v float64, n int) int {
		if wrap {
			i := int(math.Mod(v, float64(n)))
			if i < 0 {
				i += n
			}
			return i
		}
		return int(max(0, min(v, float64(n-1))))
	}
	xa, xb := at(x0, w), at(x0+1, w)
	ya, yb := at(y0, h), at(y0+1, h)
	var out [4]float32
	for k := range 4 {
		a := float32(img.Pix[ya*img.Stride+4*xa+k])
		b := float32(img.Pix[ya*img.Stride+4*xb+k])
		c := float32(img.Pix[yb*img.Stride+4*xa+k])
		e := float32(img.Pix[yb*img.Stride+4*xb+k])
		top, bottom := a+(b-a)*tx, c+(e-c)*tx
		out[k] = top + (bottom-top)*ty
	}
	return out
}

// pattern fills what the coverage holds with the pattern's tile, drawn
// once at the size it shows at, then repeated.
func (d *drawer) pattern(dst *image.RGBA, pat *Pattern, m Matrix, opacity float32, evenOdd bool) {
	if pat == nil || pat.Tile.Empty() || opacity <= 0 {
		return
	}
	toPixels := m.Mul(pat.Transform)
	sx := math.Hypot(toPixels.A, toPixels.B) * pat.Tile.W
	sy := math.Hypot(toPixels.C, toPixels.D) * pat.Tile.H
	if !(sx > 0) || !(sy > 0) || math.IsInf(sx+sy, 0) {
		return
	}
	k := min(1, maxTileSide/sx, maxTileSide/sy, math.Sqrt(maxTilePixels/(sx*sy)))
	tw, th := max(1, int(math.Ceil(sx*k))), max(1, int(math.Ceil(sy*k)))
	tileView := scale(float64(tw)/pat.Tile.W, float64(th)/pat.Tile.H).Mul(translate(-pat.Tile.X, -pat.Tile.Y))
	inv, ok := toPixels.Inverse()
	if !ok {
		return
	}
	tile := image.NewRGBA(image.Rect(0, 0, tw, th))
	sub := &drawer{view: tileView}
	sub.items(tile, tile.Rect, pat.Items, 1, 0)
	toTile := tileView.Mul(inv)
	d.cover.fill(evenOdd, func(y, x0 int, cover []float32) {
		pix := dst.Pix[dst.PixOffset(x0, y):]
		p := toTile.Apply(Point{float64(x0) + 0.5, float64(y) + 0.5})
		step := Point{toTile.A, toTile.B}
		for i, c := range cover {
			if c > 0 {
				src := bilinear(tile, p, true)
				for k := range src {
					src[k] *= opacity
				}
				blend(pix[4*i:4*i+4:4*i+4], &src, c)
			}
			p = p.add(step)
		}
	})
}
