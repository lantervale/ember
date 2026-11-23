package colourfont

import (
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
)

// maxBitmap is the most pixels a bitmap glyph's picture may have.
const maxBitmap = 1 << 22

func (f *Font) bitmapHas(gid uint16) bool {
	if f.sbix != nil {
		for i := range int(f.sbix.u32(4)) {
			if _, _, _, ok := f.sbixGlyph(i, gid, 0); ok {
				return true
			}
		}
	}
	if f.cblc != nil {
		for i := range int(f.cblc.u32(4)) {
			if _, ok := f.cbdtGlyph(i, gid); ok {
				return true
			}
		}
	}
	return false
}

// strike picks of ppems the one drawn for ppem: the smallest at least
// as big, else the biggest.
func strike(ppems []int, ppem float32) int {
	best := -1
	for i, p := range ppems {
		switch {
		case best < 0:
			best = i
		case float32(p) >= ppem && (float32(ppems[best]) < ppem || p < ppems[best]):
			best = i
		case float32(ppems[best]) < ppem && p > ppems[best]:
			best = i
		}
	}
	return best
}

func (f *Font) drawBitmap(gid uint16, req Request) (*image.RGBA, image.Point, bool) {
	if f.sbix != nil {
		var ppems, strikes []int
		for i := range int(f.sbix.u32(4)) {
			if _, _, _, ok := f.sbixGlyph(i, gid, 0); ok {
				ppems = append(ppems, int(f.sbix.u16(int(f.sbix.u32(8+4*i)))))
				strikes = append(strikes, i)
			}
		}
		if s := strike(ppems, req.PPEM); s >= 0 {
			data, x, y, _ := f.sbixGlyph(strikes[s], gid, 0)
			img := decode(data)
			if img == nil {
				return nil, image.Point{}, false
			}
			k := req.PPEM / float32(ppems[s])
			h := img.Bounds().Dy()
			out, at, ok := scaled(img, k, float32(x)*k, -float32(y+h)*k)
			if ok {
				em := req.PPEM / f.upem
				clipTo(out, at, f.outlineBox(gid, scale(em, -em)))
			}
			return out, at, ok
		}
	}
	if f.cblc != nil {
		var ppems, sizes []int
		for i := range int(f.cblc.u32(4)) {
			if _, ok := f.cbdtGlyph(i, gid); ok {
				ppems = append(ppems, int(f.cblc.u8(8+48*i+45)))
				sizes = append(sizes, i)
			}
		}
		if s := strike(ppems, req.PPEM); s >= 0 {
			g, _ := f.cbdtGlyph(sizes[s], gid)
			img := decode(g.png)
			if img == nil || ppems[s] == 0 {
				return nil, image.Point{}, false
			}
			k := req.PPEM / float32(ppems[s])
			return scaled(img, k, float32(g.bearingX)*k, -float32(g.bearingY)*k)
		}
	}
	return nil, image.Point{}, false
}

// decode is a bitmap glyph's picture, nil when it can't be read or is
// too big.
func decode(data []byte) image.Image {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width*cfg.Height > maxBitmap || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil
	}
	var img image.Image
	switch format {
	case "png":
		img, err = png.Decode(bytes.NewReader(data))
	case "jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
	default:
		return nil
	}
	if err != nil {
		return nil
	}
	return img
}

// scaled is img scaled by k, its top-left at x, y pixels from the
// glyph's origin, on whole pixels: the fraction moves it within its
// picture.
func scaled(img image.Image, k, x, y float32) (*image.RGBA, image.Point, bool) {
	x0, y0 := int(math.Floor(float64(x))), int(math.Floor(float64(y)))
	fx, fy := x-float32(x0), y-float32(y0)
	b := img.Bounds()
	w := int(math.Ceil(float64(float32(b.Dx())*k+fx))) + 1
	h := int(math.Ceil(float64(float32(b.Dy())*k+fy))) + 1
	if w <= 0 || h <= 0 || w > maxSide || h > maxSide {
		return nil, image.Point{}, false
	}
	src := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(src, src.Bounds(), img, b.Min, draw.Src)
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	s2d := f64.Aff3{float64(k), 0, float64(fx), 0, float64(k), float64(fy)}
	xdraw.ApproxBiLinear.Transform(out, s2d, src, src.Bounds(), xdraw.Src, nil)
	return out, image.Pt(x0, y0), true
}

// clipTo clears what of img, its top-left at at, lies outside b, in
// pixels from the glyph's origin: an sbix glyph shows within its
// outline's bounds, as CoreText draws it. An empty b, a glyph without
// an outline, clips nothing.
func clipTo(img *image.RGBA, at image.Point, b box) {
	if b.empty() {
		return
	}
	r := img.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			px, py := float32(x+at.X), float32(y+at.Y)
			cover := span(px, b.x0, b.x1) * span(py, b.y0, b.y1)
			if cover >= 1 {
				continue
			}
			i := img.PixOffset(x, y)
			for c := range 4 {
				img.Pix[i+c] = uint8(float32(img.Pix[i+c])*cover + 0.5)
			}
		}
	}
}

// span is how much of the pixel from p to p+1 lies between lo and hi.
func span(p, lo, hi float32) float32 { return max(0, min(p+1, hi)-max(p, lo)) }

// sbixGlyph is glyph gid's data in strike i, and its origin offsets; a
// "dupe" glyph is the one it names, followed up to depth.
func (f *Font) sbixGlyph(i int, gid uint16, depth int) ([]byte, int, int, bool) {
	t := f.sbix
	if int(gid) >= f.numGlyphs || depth > 4 {
		return nil, 0, 0, false
	}
	at := int(t.u32(8 + 4*i))
	if at <= 0 {
		return nil, 0, 0, false
	}
	start, end := int(t.u32(at+4+4*int(gid))), int(t.u32(at+4+4*int(gid)+4))
	if end-start <= 8 || !t.in(at+start, end-start) {
		return nil, 0, 0, false
	}
	g := at + start
	x, y := int(t.i16(g)), int(t.i16(g+2))
	data := t[g+8 : at+end]
	switch string(t.bytes(g+4, 4)) {
	case "png ", "jpg ":
		return data, x, y, true
	case "dupe":
		if len(data) < 2 {
			return nil, 0, 0, false
		}
		return f.sbixGlyph(i, uint16(data[0])<<8|uint16(data[1]), depth+1)
	}
	return nil, 0, 0, false
}

// cbdtBitmap is a CBDT glyph: its PNG and where its top-left is from
// the origin, y up, in the pixels of its size.
type cbdtBitmap struct {
	png                []byte
	bearingX, bearingY int
}

// cbdtGlyph is glyph gid at CBLC's size i.
func (f *Font) cbdtGlyph(i int, gid uint16) (cbdtBitmap, bool) {
	t := f.cblc
	size := 8 + 48*i
	if gid < t.u16(size+40) || gid > t.u16(size+42) {
		return cbdtBitmap{}, false
	}
	array := int(t.u32(size))
	for k := range int(t.u32(size + 8)) {
		rec := array + 8*k
		first, last := t.u16(rec), t.u16(rec+2)
		if gid < first || gid > last {
			continue
		}
		sub := array + int(t.u32(rec+4))
		return f.cbdtSub(sub, gid, first)
	}
	return cbdtBitmap{}, false
}

// cbdtSub finds gid in the index subtable at sub, whose glyphs start
// at first.
func (f *Font) cbdtSub(sub int, gid, first uint16) (cbdtBitmap, bool) {
	t := f.cblc
	indexFormat, imageFormat, data := t.u16(sub), t.u16(sub+2), int(t.u32(sub+4))
	var start, end int
	var metrics []byte
	n := int(gid - first)
	switch indexFormat {
	case 1:
		start, end = int(t.u32(sub+8+4*n)), int(t.u32(sub+8+4*n+4))
	case 2:
		size := int(t.u32(sub + 8))
		start, end = size*n, size*(n+1)
		metrics = t.bytes(sub+12, 8)
	case 3:
		start, end = int(t.u16(sub+8+2*n)), int(t.u16(sub+8+2*n+2))
	case 4:
		count := int(t.u32(sub + 8))
		for j := range count {
			if t.u16(sub+12+4*j) == gid {
				start, end = int(t.u16(sub+12+4*j+2)), int(t.u16(sub+12+4*j+6))
				break
			}
		}
	case 5:
		size := int(t.u32(sub + 8))
		metrics = t.bytes(sub+12, 8)
		count := int(t.u32(sub + 20))
		found := false
		for j := range count {
			if t.u16(sub+24+2*j) == gid {
				start, end, found = size*j, size*(j+1), true
				break
			}
		}
		if !found {
			return cbdtBitmap{}, false
		}
	default:
		return cbdtBitmap{}, false
	}
	if end <= start {
		return cbdtBitmap{}, false
	}
	g := table(f.cbdt)
	at := data + start
	if !g.in(at, end-start) {
		return cbdtBitmap{}, false
	}
	var out cbdtBitmap
	var png int
	switch imageFormat {
	case 17:
		out.bearingX, out.bearingY = int(int8(g.u8(at+2))), int(int8(g.u8(at+3))) //nolint:gosec // signed bytes
		png = at + 5
	case 18:
		out.bearingX, out.bearingY = int(int8(g.u8(at+2))), int(int8(g.u8(at+3))) //nolint:gosec // signed bytes
		png = at + 8
	case 19:
		if len(metrics) < 8 {
			return cbdtBitmap{}, false
		}
		out.bearingX, out.bearingY = int(int8(metrics[2])), int(int8(metrics[3])) //nolint:gosec // signed bytes
		png = at
	default:
		return cbdtBitmap{}, false
	}
	n32 := int(g.u32(png))
	if !g.in(png+4, n32) {
		return cbdtBitmap{}, false
	}
	out.png = g[png+4 : png+4+n32]
	return out, true
}
