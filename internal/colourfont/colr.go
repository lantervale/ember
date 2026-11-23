package colourfont

import (
	"image"
	"math"
)

// maxDepth is how deep a paint graph is followed: deeper, or round a
// cycle, it draws nothing more.
const maxDepth = 64

// maxSide is the longest side of a glyph's picture, in pixels.
const maxSide = 4096

func (f *Font) colrVersion() uint16 { return f.colr.u16(0) }

// v0Layers is base glyph gid's first layer and count in COLR version 0.
func (f *Font) v0Layers(gid uint16) (int, int, bool) {
	t := f.colr
	n, at := int(t.u16(2)), int(t.u32(4))
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		rec := at + 6*mid
		switch g := t.u16(rec); {
		case g < gid:
			lo = mid + 1
		case g > gid:
			hi = mid
		default:
			return int(t.u16(rec + 2)), int(t.u16(rec + 4)), true
		}
	}
	return 0, 0, false
}

// basePaint is where base glyph gid's paint is in COLR version 1, 0 for
// none.
func (f *Font) basePaint(gid uint16) int {
	if f.colrVersion() < 1 {
		return 0
	}
	t := f.colr
	list := int(t.u32(14))
	if list == 0 {
		return 0
	}
	lo, hi := 0, int(t.u32(list))
	for lo < hi {
		mid := (lo + hi) / 2
		rec := list + 4 + 6*mid
		switch g := t.u16(rec); {
		case g < gid:
			lo = mid + 1
		case g > gid:
			hi = mid
		default:
			if off := int(t.u32(rec + 2)); off > 0 {
				return list + off
			}
			return 0
		}
	}
	return 0
}

// clipBox is base glyph gid's clip box in font units, false for none.
func (f *Font) clipBox(gid uint16) (box, bool) {
	if f.colrVersion() < 1 {
		return box{}, false
	}
	t := f.colr
	list := int(t.u32(22))
	if list == 0 {
		return box{}, false
	}
	n := int(t.u32(list + 1))
	for i := range n {
		rec := list + 5 + 7*i
		if gid < t.u16(rec) || gid > t.u16(rec+2) {
			continue
		}
		at := list + int(t.u24(rec+4))
		return box{float32(t.i16(at + 1)), float32(t.i16(at + 3)), float32(t.i16(at + 5)), float32(t.i16(at + 7))}, true
	}
	return box{}, false
}

func (f *Font) colrHas(gid uint16) bool {
	if f.colr == nil {
		return false
	}
	if f.basePaint(gid) != 0 {
		return true
	}
	_, n, ok := f.v0Layers(gid)
	return ok && n > 0
}

// colrCtx is a COLR glyph being drawn.
type colrCtx struct {
	f      *Font
	req    Request
	w, h   int
	inside map[int]bool
}

func (f *Font) drawCOLR(gid uint16, req Request) (*image.RGBA, image.Point, bool) {
	s := req.PPEM / f.upem
	dev := scale(s, -s)
	paint := f.basePaint(gid)
	b := emptyBox()
	if clip, ok := f.clipBox(gid); ok {
		b = corners(clip.x0, clip.y0, clip.x1, clip.y1, dev)
	} else if paint != 0 {
		b = f.paintBox(paint, dev, 0, map[int]bool{})
	} else {
		first, n, _ := f.v0Layers(gid)
		for i := range n {
			b = b.union(f.outlineBox(f.colr.u16(int(f.colr.u32(8))+4*(first+i)), dev))
		}
	}
	if b.empty() {
		return nil, image.Point{}, false
	}
	x0, y0 := int(math.Floor(float64(b.x0)))-1, int(math.Floor(float64(b.y0)))-1
	x1, y1 := int(math.Ceil(float64(b.x1)))+1, int(math.Ceil(float64(b.y1)))+1
	if x1-x0 > maxSide || y1-y0 > maxSide {
		return nil, image.Point{}, false
	}
	c := &colrCtx{f: f, req: req, w: x1 - x0, h: y1 - y0, inside: map[int]bool{}}
	dev = translate(float32(-x0), float32(-y0)).then(dev)
	var out *layer
	if paint != 0 {
		out = c.paint(paint, dev, 0)
		out = c.clipped(out, gid, dev)
	} else {
		out = c.v0(gid, dev)
	}
	if out == nil {
		out = newLayer(c.w, c.h)
	}
	return out.rgba(), image.Pt(x0, y0), true
}

// v0 is a COLR version 0 glyph: its layers over each other.
func (c *colrCtx) v0(gid uint16, m affine) *layer {
	t := c.f.colr
	first, n, _ := c.f.v0Layers(gid)
	at := int(t.u32(8))
	out := newLayer(c.w, c.h)
	for i := range n {
		rec := at + 4*(first+i)
		mask := c.f.coverage(t.u16(rec), m, c.w, c.h)
		if mask == nil {
			continue
		}
		l := c.solid(c.f.colour(t.u16(rec+2), 1, c.req))
		l.masked(mask)
		out.over(l)
	}
	return out
}

// clipped is l cut to base glyph gid's clip box, when it has one.
func (c *colrCtx) clipped(l *layer, gid uint16, m affine) *layer {
	clip, ok := c.f.clipBox(gid)
	if !ok || l == nil {
		return l
	}
	mask := rectMask(clip, m, c.w, c.h)
	l.masked(mask)
	return l
}

func (c *colrCtx) solid(col [4]float32) *layer {
	l := newLayer(c.w, c.h)
	for i := 0; i < len(l.px); i += 4 {
		copy(l.px[i:i+4], col[:])
	}
	return l
}

// paint draws the paint at at in COLR through m, nil for nothing.
func (c *colrCtx) paint(at int, m affine, depth int) *layer {
	if depth > maxDepth || at <= 0 || at >= len(c.f.colr) {
		return nil
	}
	t := c.f.colr
	format := t.u8(at)
	child := func(off int) int {
		if o := int(t.u24(at + off)); o > 0 {
			return at + o
		}
		return 0
	}
	switch format {
	case 1:
		n, first := int(t.u8(at+1)), int(t.u32(at+2))
		list := int(t.u32(18))
		if list == 0 {
			return nil
		}
		out := newLayer(c.w, c.h)
		for i := range n {
			off := int(t.u32(list + 4 + 4*(first+i)))
			if off == 0 || first+i >= int(t.u32(list)) {
				continue
			}
			out.over(c.paint(list+off, m, depth+1))
		}
		return out
	case 2, 3:
		return c.solid(c.f.colour(t.u16(at+1), t.f2dot14(at+3), c.req))
	case 4, 5:
		line := c.line(child(1), format == 5)
		p := [6]float32{}
		for i := range p {
			p[i] = float32(t.i16(at + 4 + 2*i))
		}
		return c.gradient(m, line, linear(p))
	case 6, 7:
		line := c.line(child(1), format == 7)
		x0, y0, r0 := float32(t.i16(at+4)), float32(t.i16(at+6)), float32(t.u16(at+8))
		x1, y1, r1 := float32(t.i16(at+10)), float32(t.i16(at+12)), float32(t.u16(at+14))
		return c.gradient(m, line, radial(x0, y0, r0, x1, y1, r1))
	case 8, 9:
		line := c.line(child(1), format == 9)
		cx, cy := float32(t.i16(at+4)), float32(t.i16(at+6))
		start, end := (t.f2dot14(at+8)+1)*180, (t.f2dot14(at+10)+1)*180
		return c.gradient(m, line, sweep(cx, cy, start, end))
	case 10:
		mask := c.f.coverage(t.u16(at+4), m, c.w, c.h)
		if mask == nil {
			return nil
		}
		l := c.paint(child(1), m, depth+1)
		if l == nil {
			return nil
		}
		l.masked(mask)
		return l
	case 11:
		gid := t.u16(at + 1)
		base := c.f.basePaint(gid)
		if base == 0 || c.inside[base] {
			return nil
		}
		c.inside[base] = true
		defer delete(c.inside, base)
		return c.clipped(c.paint(base, m, depth+1), gid, m)
	case 12, 13:
		a := child(4)
		tr := affine{xx: t.fixed(a), yx: t.fixed(a + 4), xy: t.fixed(a + 8), yy: t.fixed(a + 12),
			dx: t.fixed(a + 16), dy: t.fixed(a + 20)}
		return c.paint(child(1), m.then(tr), depth+1)
	}
	if format >= 14 && format <= 31 {
		return c.paint(child(1), m.then(transformOf(t, at, format)), depth+1)
	}
	if format == 32 {
		src := c.paint(child(1), m, depth+1)
		dst := c.paint(child(5), m, depth+1)
		return composite(t.u8(at+4), src, dst, c.w, c.h)
	}
	return nil
}

// transformOf is the transform of a PaintTranslate to PaintVarSkewAroundCenter
// at at, its fields after its child's offset.
func transformOf(t table, at int, format uint8) affine {
	v := at + 4
	around := func(m affine, cx, cy float32) affine {
		return translate(cx, cy).then(m).then(translate(-cx, -cy))
	}
	turn := func(a float32) float32 { return a * math.Pi }
	switch format {
	case 14, 15:
		return translate(float32(t.i16(v)), float32(t.i16(v+2)))
	case 16, 17:
		return scale(t.f2dot14(v), t.f2dot14(v+2))
	case 18, 19:
		return around(scale(t.f2dot14(v), t.f2dot14(v+2)), float32(t.i16(v+4)), float32(t.i16(v+6)))
	case 20, 21:
		s := t.f2dot14(v)
		return scale(s, s)
	case 22, 23:
		s := t.f2dot14(v)
		return around(scale(s, s), float32(t.i16(v+2)), float32(t.i16(v+4)))
	case 24, 25, 26, 27:
		a := turn(t.f2dot14(v))
		sin, cos := float32(math.Sin(float64(a))), float32(math.Cos(float64(a)))
		r := affine{xx: cos, yx: sin, xy: -sin, yy: cos}
		if format >= 26 {
			return around(r, float32(t.i16(v+2)), float32(t.i16(v+4)))
		}
		return r
	case 28, 29, 30, 31:
		x, y := turn(t.f2dot14(v)), turn(t.f2dot14(v+2))
		s := affine{xx: 1, yx: float32(math.Tan(float64(y))), xy: -float32(math.Tan(float64(x))), yy: 1}
		if format >= 30 {
			return around(s, float32(t.i16(v+4)), float32(t.i16(v+6)))
		}
		return s
	}
	return identity()
}

// paintBox bounds what the paint at at draws through m: the outlines of
// its glyphs, empty when it draws none.
func (f *Font) paintBox(at int, m affine, depth int, inside map[int]bool) box {
	t := f.colr
	b := emptyBox()
	if depth > maxDepth || at <= 0 || at >= len(t) {
		return b
	}
	format := t.u8(at)
	child := func(off int) int {
		if o := int(t.u24(at + off)); o > 0 {
			return at + o
		}
		return 0
	}
	switch {
	case format == 1:
		n, first := int(t.u8(at+1)), int(t.u32(at+2))
		list := int(t.u32(18))
		if list == 0 {
			return b
		}
		for i := range n {
			if off := int(t.u32(list + 4 + 4*(first+i))); off > 0 && first+i < int(t.u32(list)) {
				b = b.union(f.paintBox(list+off, m, depth+1, inside))
			}
		}
	case format == 10:
		b = f.outlineBox(t.u16(at+4), m)
	case format == 11:
		gid := t.u16(at + 1)
		if clip, ok := f.clipBox(gid); ok {
			return corners(clip.x0, clip.y0, clip.x1, clip.y1, m)
		}
		if base := f.basePaint(gid); base != 0 && !inside[base] {
			inside[base] = true
			b = f.paintBox(base, m, depth+1, inside)
			delete(inside, base)
		}
	case format == 12 || format == 13:
		a := child(4)
		tr := affine{xx: t.fixed(a), yx: t.fixed(a + 4), xy: t.fixed(a + 8), yy: t.fixed(a + 12),
			dx: t.fixed(a + 16), dy: t.fixed(a + 20)}
		b = f.paintBox(child(1), m.then(tr), depth+1, inside)
	case format >= 14 && format <= 31:
		b = f.paintBox(child(1), m.then(transformOf(t, at, format)), depth+1, inside)
	case format == 32:
		b = f.paintBox(child(1), m, depth+1, inside).union(f.paintBox(child(5), m, depth+1, inside))
	}
	return b
}

// rectMask covers the rectangle r in font units through m.
func rectMask(r box, m affine, w, h int) *image.Alpha {
	var corners [4][2]float32
	for i, p := range [4][2]float32{{r.x0, r.y0}, {r.x1, r.y0}, {r.x1, r.y1}, {r.x0, r.y1}} {
		corners[i][0], corners[i][1] = m.apply(p[0], p[1])
	}
	return polygonMask(corners[:], w, h)
}
