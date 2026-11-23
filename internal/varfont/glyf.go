package varfont

import (
	"encoding/binary"
)

// Point is a point of a glyph's outline in font units, y up, on the
// curve or its quadratic control point.
type Point struct {
	X, Y float32
	On   bool
}

type fpoint struct{ x, y float32 }

// outline is a glyph as gvar varies it: its points, the last index of
// each contour, and its four phantom points (left and right of its
// advance, top and bottom).
type outline struct {
	points  []Point
	ends    []int
	phantom [4]fpoint
}

const (
	flagOnCurve = 0x01
	flagXShort  = 0x02
	flagYShort  = 0x04
	flagRepeat  = 0x08
	flagXSame   = 0x10
	flagYSame   = 0x20

	compArgsAreWords = 0x0001
	compArgsAreXY    = 0x0002
	compScale        = 0x0008
	compMore         = 0x0020
	compXYScale      = 0x0040
	comp2x2          = 0x0080
)

// maxDepth is how deep composite glyphs may nest.
const maxDepth = 8

// glyphData is glyph g's bytes in glyf, empty for a glyph with none.
func (f *Font) glyphData(g uint16) ([]byte, error) {
	be := binary.BigEndian
	i := int(g)
	var lo, hi int
	if f.longLoca {
		if 4*i+8 > len(f.loca) {
			return nil, errGlyph
		}
		lo, hi = int(be.Uint32(f.loca[4*i:])), int(be.Uint32(f.loca[4*i+4:]))
	} else {
		if 2*i+4 > len(f.loca) {
			return nil, errGlyph
		}
		lo, hi = 2*int(be.Uint16(f.loca[2*i:])), 2*int(be.Uint16(f.loca[2*i+2:]))
	}
	if lo > hi || hi > len(f.glyf) {
		return nil, errGlyph
	}
	return f.glyf[lo:hi], nil
}

// Outline is glyph g at c: its points and the last index of each
// contour.
func (f *Font) Outline(g uint16, c Coords) ([]Point, []int, error) {
	o, err := f.glyph(g, c, 0)
	if err != nil {
		return nil, nil, err
	}
	return o.points, o.ends, nil
}

// glyph is g's outline at c, its composites' components placed.
func (f *Font) glyph(g uint16, c Coords, depth int) (outline, error) {
	var o outline
	if int(g) >= f.numGlyphs || depth > maxDepth {
		return o, errGlyph
	}
	data, err := f.glyphData(g)
	if err != nil {
		return o, err
	}
	var xMin float32
	if len(data) >= 10 {
		xMin = float32(int16(binary.BigEndian.Uint16(data[2:]))) //nolint:gosec // a signed number
	}
	left := xMin - float32(f.lsb(g))
	o.phantom = [4]fpoint{{left, 0}, {left + float32(f.baseAdvance(g)), 0}}
	if len(data) < 10 {
		f.vary(g, c, &o)
		return o, nil
	}
	n := int(int16(binary.BigEndian.Uint16(data))) //nolint:gosec // a signed count
	if n >= 0 {
		if o.points, o.ends, err = simple(data, n); err != nil {
			return o, err
		}
		f.vary(g, c, &o)
		return o, nil
	}
	return f.composite(g, c, data[10:], o, depth)
}

// simple reads a glyph of n contours.
func simple(data []byte, n int) ([]Point, []int, error) {
	be := binary.BigEndian
	at := 10
	if at+2*n+2 > len(data) {
		return nil, nil, errGlyph
	}
	ends := make([]int, n)
	last := -1
	for i := range n {
		ends[i] = int(be.Uint16(data[at+2*i:]))
		if ends[i] <= last {
			return nil, nil, errGlyph
		}
		last = ends[i]
	}
	at += 2 * n
	at += 2 + int(be.Uint16(data[at:]))
	count := last + 1
	flags := make([]byte, 0, count)
	for len(flags) < count {
		if at >= len(data) {
			return nil, nil, errGlyph
		}
		fl := data[at]
		at++
		flags = append(flags, fl)
		if fl&flagRepeat != 0 {
			if at >= len(data) {
				return nil, nil, errGlyph
			}
			for r := int(data[at]); r > 0 && len(flags) < count; r-- {
				flags = append(flags, fl)
			}
			at++
		}
	}
	pts := make([]Point, count)
	var ok bool
	if at, ok = coords(data, at, flags, flagXShort, flagXSame, func(i int, v float32) { pts[i].X = v }); !ok {
		return nil, nil, errGlyph
	}
	if _, ok = coords(data, at, flags, flagYShort, flagYSame, func(i int, v float32) { pts[i].Y = v }); !ok {
		return nil, nil, errGlyph
	}
	for i, fl := range flags {
		pts[i].On = fl&flagOnCurve != 0
	}
	return pts, ends, nil
}

// coords reads one coordinate of every point from at, each relative to
// the one before, and returns where they end.
func coords(data []byte, at int, flags []byte, short, same byte, set func(int, float32)) (int, bool) {
	v := 0
	for i, fl := range flags {
		switch {
		case fl&short != 0:
			if at >= len(data) {
				return at, false
			}
			d := int(data[at])
			if fl&same == 0 {
				d = -d
			}
			v += d
			at++
		case fl&same == 0:
			if at+2 > len(data) {
				return at, false
			}
			v += int(int16(binary.BigEndian.Uint16(data[at:]))) //nolint:gosec // a signed number
			at += 2
		}
		set(i, float32(v))
	}
	return at, true
}

// component is one glyph a composite places, as glyf gives it.
type component struct {
	glyph  uint16
	flags  uint16
	dx, dy float32
	// m is the component's 2×2 transform: xx, xy, yx, yy.
	m [4]float32
}

// composite is the glyph of components data lists, each varied at c,
// transformed and placed where gvar moves its offset.
func (f *Font) composite(g uint16, c Coords, data []byte, o outline, depth int) (outline, error) {
	var comps []component
	be := binary.BigEndian
	for at := 0; ; {
		if at+4 > len(data) {
			return o, errGlyph
		}
		cp := component{flags: be.Uint16(data[at:]), glyph: be.Uint16(data[at+2:]), m: [4]float32{1, 0, 0, 1}}
		at += 4
		if cp.flags&compArgsAreWords != 0 {
			if at+4 > len(data) {
				return o, errGlyph
			}
			cp.dx, cp.dy = float32(int16(be.Uint16(data[at:]))), float32(int16(be.Uint16(data[at+2:]))) //nolint:gosec // signed
			at += 4
		} else {
			if at+2 > len(data) {
				return o, errGlyph
			}
			cp.dx, cp.dy = float32(int8(data[at])), float32(int8(data[at+1])) //nolint:gosec // signed
			at += 2
		}
		var n int
		switch {
		case cp.flags&compScale != 0:
			n = 1
		case cp.flags&compXYScale != 0:
			n = 2
		case cp.flags&comp2x2 != 0:
			n = 4
		}
		if at+2*n > len(data) {
			return o, errGlyph
		}
		switch n {
		case 1:
			s := f2dot14(data[at:])
			cp.m = [4]float32{s, 0, 0, s}
		case 2:
			cp.m = [4]float32{f2dot14(data[at:]), 0, 0, f2dot14(data[at+2:])}
		case 4:
			cp.m = [4]float32{f2dot14(data[at:]), f2dot14(data[at+2:]), f2dot14(data[at+4:]), f2dot14(data[at+6:])}
		}
		at += 2 * n
		comps = append(comps, cp)
		if cp.flags&compMore == 0 {
			break
		}
	}
	offsets := make([]Point, len(comps))
	for i, cp := range comps {
		offsets[i] = Point{X: cp.dx, Y: cp.dy}
	}
	o.points = offsets
	f.vary(g, c, &o)
	placed := o
	placed.points, placed.ends = nil, nil
	for i, cp := range comps {
		sub, err := f.glyph(cp.glyph, c, depth+1)
		if err != nil {
			return o, err
		}
		base := len(placed.points)
		for _, p := range sub.points {
			x := cp.m[0]*p.X + cp.m[2]*p.Y
			y := cp.m[1]*p.X + cp.m[3]*p.Y
			if cp.flags&compArgsAreXY != 0 {
				x += o.points[i].X
				y += o.points[i].Y
			}
			placed.points = append(placed.points, Point{X: x, Y: y, On: p.On})
		}
		if cp.flags&compArgsAreXY == 0 {
			// Point matching: the component's point dx lands on the
			// glyph's point dy, of those placed so far.
			a, b := int(cp.dx), base+int(cp.dy)
			if a >= 0 && b >= 0 && a < base && b < len(placed.points) {
				shiftX, shiftY := placed.points[a].X-placed.points[b].X, placed.points[a].Y-placed.points[b].Y
				for j := base; j < len(placed.points); j++ {
					placed.points[j].X += shiftX
					placed.points[j].Y += shiftY
				}
			}
		}
		for _, e := range sub.ends {
			placed.ends = append(placed.ends, base+e)
		}
	}
	return placed, nil
}

// Contours calls fn with each contour of points as quadratic segments
// from its first point on the curve: line(to) or quad(control, to), the
// on-curve point between two control points implied, as TrueType has
// it. move starts each contour.
func Contours(points []Point, ends []int, move func(x, y float32), line func(x, y float32),
	quad func(cx, cy, x, y float32)) {
	start := 0
	for _, end := range ends {
		if end >= len(points) || end < start {
			return
		}
		contour := points[start : end+1]
		start = end + 1
		n := len(contour)
		first := -1
		for i, p := range contour {
			if p.On {
				first = i
				break
			}
		}
		var sx, sy float32
		if first < 0 {
			a, b := contour[0], contour[n-1]
			sx, sy = (a.X+b.X)/2, (a.Y+b.Y)/2
			move(sx, sy)
			walk(contour, 0, n, sx, sy, line, quad)
			continue
		}
		sx, sy = contour[first].X, contour[first].Y
		move(sx, sy)
		walk(contour, first+1, n-1, sx, sy, line, quad)
	}
}

// walk draws count points of contour from from, round to where it
// started at (sx, sy).
func walk(contour []Point, from, count int, sx, sy float32, line func(x, y float32), quad func(cx, cy, x, y float32)) {
	var ctrl *Point
	for k := range count {
		p := contour[(from+k)%len(contour)]
		switch {
		case p.On && ctrl == nil:
			line(p.X, p.Y)
		case p.On:
			quad(ctrl.X, ctrl.Y, p.X, p.Y)
			ctrl = nil
		case ctrl != nil:
			quad(ctrl.X, ctrl.Y, (ctrl.X+p.X)/2, (ctrl.Y+p.Y)/2)
			q := p
			ctrl = &q
		default:
			q := p
			ctrl = &q
		}
	}
	if ctrl != nil {
		quad(ctrl.X, ctrl.Y, sx, sy)
	} else {
		line(sx, sy)
	}
}
