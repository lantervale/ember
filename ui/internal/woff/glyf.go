package woff

import "encoding/binary"

// glyphs is what a transformed glyf table rebuilds: the glyf and loca
// tables, and each glyph's xMin, which a transformed hmtx leaves out.
type glyphs struct {
	glyf, loca []byte
	xMins      []int16
}

const (
	glyfOnCurve  = 0x01
	glyfXShort   = 0x02
	glyfYShort   = 0x04
	glyfRepeat   = 0x08
	glyfXSame    = 0x10
	glyfYSame    = 0x20
	glyfOverlap  = 0x40
	compWords    = 0x0001
	compScale    = 0x0008
	compMore     = 0x0020
	compXYScale  = 0x0040
	comp2x2      = 0x0080
	compInstruct = 0x0100
)

type point struct {
	x, y int
	on   bool
}

// glyfStreams are the seven streams of a transformed glyf table, the
// bits saying which glyphs overlap and which give their box, and the
// contour ends and points of the glyph being rebuilt.
type glyfStreams struct {
	contours, points, flags, glyph, composite, bbox, instructions *reader
	overlaps, explicit                                            []byte

	ends []uint16
	pts  []point
}

// untransformGlyf rebuilds the glyf and loca tables from the
// transformed glyf table of a WOFF2 file: seven streams, read side by
// side, glyph by glyph. Each glyph is written on a 4-byte boundary.
func untransformGlyf(data []byte) (glyphs, error) {
	var none glyphs
	head := &reader{b: data}
	head.u16()
	options := head.u16()
	numGlyphs := int(head.u16())
	indexFormat := head.u16()
	s, ok := glyfStreamsOf(head, options, numGlyphs)
	if !ok {
		return none, errTransform
	}

	out := glyphs{xMins: make([]int16, numGlyphs)}
	offsets := make([]uint32, numGlyphs+1)
	for i := 0; i < numGlyphs; i++ {
		offsets[i] = uint32(len(out.glyf)) //nolint:gosec // under MaxSize, checked below
		n := s.contours.u16()
		if s.contours.bad {
			return none, errTransform
		}
		hasBox := bit(s.explicit, i)
		switch n {
		case 0:
			if hasBox {
				return none, errTransform
			}
			continue
		case 0xffff:
			if !hasBox {
				return none, errTransform
			}
			out.glyf, out.xMins[i], ok = s.compositeGlyph(out.glyf)
		default:
			out.glyf, out.xMins[i], ok = s.simpleGlyph(out.glyf, n, hasBox, bit(s.overlaps, i))
		}
		if !ok {
			return none, errTransform
		}
		for len(out.glyf)&3 != 0 {
			out.glyf = append(out.glyf, 0)
		}
		if len(out.glyf) > MaxSize {
			return none, errTooLarge
		}
	}
	offsets[numGlyphs] = uint32(len(out.glyf)) //nolint:gosec // under MaxSize
	if out.loca, ok = locaOf(offsets, indexFormat); !ok {
		return none, errTransform
	}
	return out, nil
}

// glyfStreamsOf cuts the streams of a transformed glyf table out of
// what follows its header's first fields in head.
func glyfStreamsOf(head *reader, options uint16, numGlyphs int) (*glyfStreams, bool) {
	var streams [7]*reader
	sizes := make([]uint32, len(streams))
	for i := range sizes {
		sizes[i] = head.u32()
	}
	for i, size := range sizes {
		if uint64(size) > uint64(len(head.b)) {
			return nil, false
		}
		streams[i] = &reader{b: head.take(int(size))}
	}
	if head.bad {
		return nil, false
	}
	s := &glyfStreams{contours: streams[0], points: streams[1], flags: streams[2], glyph: streams[3],
		composite: streams[4], bbox: streams[5], instructions: streams[6]}
	if options&1 != 0 {
		if s.overlaps = head.take((numGlyphs + 7) >> 3); s.overlaps == nil {
			return nil, false
		}
	}
	s.explicit = s.bbox.take(((numGlyphs + 31) >> 5) << 2)
	if s.bbox.bad {
		return nil, false
	}
	return s, true
}

func bit(set []byte, i int) bool { return set != nil && set[i>>3]&(0x80>>(i&7)) != 0 }

// compositeGlyph appends the next composite glyph to glyf, and returns
// its xMin with it.
func (s *glyfStreams) compositeGlyph(glyf []byte) ([]byte, int16, bool) {
	be := binary.BigEndian
	parts, instructed, ok := components(s.composite)
	if !ok {
		return nil, 0, false
	}
	box := s.bbox.take(8)
	if box == nil {
		return nil, 0, false
	}
	glyf = be.AppendUint16(glyf, 0xffff)
	glyf = append(glyf, box...)
	glyf = append(glyf, parts...)
	if instructed {
		code := s.instructions.take(s.glyph.short255())
		if s.glyph.bad || s.instructions.bad {
			return nil, 0, false
		}
		glyf = be.AppendUint16(glyf, uint16(len(code))) //nolint:gosec // a 255UInt16
		glyf = append(glyf, code...)
	}
	return glyf, int16(be.Uint16(box)), true //nolint:gosec // the bits of an int16
}

// simpleGlyph appends the next glyph of n contours to glyf, and returns
// its xMin with it: from its box where the file gives one, else from
// its points.
func (s *glyfStreams) simpleGlyph(glyf []byte, n uint16, hasBox, overlap bool) ([]byte, int16, bool) {
	be := binary.BigEndian
	s.ends = s.ends[:0]
	total := 0
	for c := 0; c < int(n); c++ {
		total += s.points.short255()
		if s.points.bad || total > 0x10000 {
			return nil, 0, false
		}
		s.ends = append(s.ends, uint16(total-1)) //nolint:gosec // as the format stores it
	}
	var ok bool
	if s.pts, ok = triplets(s.pts[:0], s.flags.take(total), s.glyph); !ok || s.flags.bad {
		return nil, 0, false
	}
	code := s.instructions.take(s.glyph.short255())
	if s.glyph.bad || s.instructions.bad {
		return nil, 0, false
	}
	glyf = be.AppendUint16(glyf, n)
	var xMin int16
	if hasBox {
		box := s.bbox.take(8)
		if box == nil {
			return nil, 0, false
		}
		glyf = append(glyf, box...)
		xMin = int16(be.Uint16(box)) //nolint:gosec // the bits of an int16
	} else {
		box := bounds(s.pts)
		for _, v := range box {
			glyf = be.AppendUint16(glyf, uint16(v)) //nolint:gosec // the bits of an int16
		}
		xMin = box[0]
	}
	for _, e := range s.ends {
		glyf = be.AppendUint16(glyf, e)
	}
	glyf = be.AppendUint16(glyf, uint16(len(code))) //nolint:gosec // a 255UInt16
	glyf = append(glyf, code...)
	return storePoints(glyf, s.pts, overlap), xMin, true
}

// locaOf is the loca table of the glyphs at offsets: half each offset
// in two bytes when indexFormat is 0, else each in four.
func locaOf(offsets []uint32, indexFormat uint16) ([]byte, bool) {
	be := binary.BigEndian
	if indexFormat != 0 {
		loca := make([]byte, 0, 4*len(offsets))
		for _, o := range offsets {
			loca = be.AppendUint32(loca, o)
		}
		return loca, true
	}
	if offsets[len(offsets)-1] > 0x1fffe {
		return nil, false
	}
	loca := make([]byte, 0, 2*len(offsets))
	for _, o := range offsets {
		loca = be.AppendUint16(loca, uint16(o>>1)) //nolint:gosec // checked above
	}
	return loca, true
}

// components reads one composite glyph's components from r, as they
// are in a glyf table, and reports whether instructions follow them.
func components(r *reader) ([]byte, bool, bool) {
	start := r.b
	n := 0
	instructed := false
	for more := true; more; {
		f := r.u16()
		size := 2 + 2
		if f&compWords != 0 {
			size += 2
		}
		switch {
		case f&compScale != 0:
			size += 2
		case f&compXYScale != 0:
			size += 4
		case f&comp2x2 != 0:
			size += 8
		}
		if r.take(size) == nil {
			return nil, false, false
		}
		n += 2 + size
		instructed = instructed || f&compInstruct != 0
		more = f&compMore != 0
	}
	return start[:n], instructed, true
}

// triplets appends the points flags and the triplets in r say: one flag
// for each point, its top bit clear when the point is on the curve, the
// rest saying how the bytes that follow in r hold the move from the
// point before.
func triplets(pts []point, flags []byte, r *reader) ([]point, bool) {
	sign := func(flag, v int) int {
		if flag&1 != 0 {
			return v
		}
		return -v
	}
	x, y := 0, 0
	for _, f := range flags {
		on := f&0x80 == 0
		flag := int(f & 0x7f)
		var n int
		switch {
		case flag < 84:
			n = 1
		case flag < 120:
			n = 2
		case flag < 124:
			n = 3
		default:
			n = 4
		}
		in := r.take(n)
		if in == nil {
			return nil, false
		}
		var dx, dy int
		switch {
		case flag < 10:
			dy = sign(flag, (flag&14)<<7+int(in[0]))
		case flag < 20:
			dx = sign(flag, ((flag-10)&14)<<7+int(in[0]))
		case flag < 84:
			b0, b1 := flag-20, int(in[0])
			dx = sign(flag, 1+b0&0x30+b1>>4)
			dy = sign(flag>>1, 1+(b0&0x0c)<<2+b1&0x0f)
		case flag < 120:
			b0 := flag - 84
			dx = sign(flag, 1+(b0/12)<<8+int(in[0]))
			dy = sign(flag>>1, 1+((b0%12)>>2)<<8+int(in[1]))
		case flag < 124:
			b2 := int(in[1])
			dx = sign(flag, int(in[0])<<4+b2>>4)
			dy = sign(flag>>1, (b2&0x0f)<<8+int(in[2]))
		default:
			dx = sign(flag, int(in[0])<<8+int(in[1]))
			dy = sign(flag>>1, int(in[2])<<8+int(in[3]))
		}
		x += dx
		y += dy
		pts = append(pts, point{x, y, on})
	}
	return pts, true
}

// bounds is the box round pts: xMin, yMin, xMax, yMax.
func bounds(pts []point) [4]int16 {
	if len(pts) == 0 {
		return [4]int16{}
	}
	xMin, yMin, xMax, yMax := pts[0].x, pts[0].y, pts[0].x, pts[0].y
	for _, p := range pts[1:] {
		xMin, xMax = min(xMin, p.x), max(xMax, p.x)
		yMin, yMax = min(yMin, p.y), max(yMax, p.y)
	}
	//nolint:gosec // truncated as the format stores them
	return [4]int16{int16(xMin), int16(yMin), int16(xMax), int16(yMax)}
}

// storePoints appends pts as a simple glyph's flags and coordinates,
// packed as a glyf table packs them: short and repeated where they can
// be.
func storePoints(out []byte, pts []point, overlap bool) []byte {
	be := binary.BigEndian
	last, repeat := -1, 0
	x, y := 0, 0
	for i, p := range pts {
		flag := 0
		if p.on {
			flag |= glyfOnCurve
		}
		if overlap && i == 0 {
			flag |= glyfOverlap
		}
		dx, dy := p.x-x, p.y-y
		switch {
		case dx == 0:
			flag |= glyfXSame
		case dx > -256 && dx < 256:
			flag |= glyfXShort
			if dx > 0 {
				flag |= glyfXSame
			}
		}
		switch {
		case dy == 0:
			flag |= glyfYSame
		case dy > -256 && dy < 256:
			flag |= glyfYShort
			if dy > 0 {
				flag |= glyfYSame
			}
		}
		if flag == last && repeat != 255 {
			out[len(out)-1] |= glyfRepeat
			repeat++
		} else {
			if repeat != 0 {
				out = append(out, byte(repeat))
				repeat = 0
			}
			out = append(out, byte(flag))
		}
		x, y, last = p.x, p.y, flag
	}
	if repeat != 0 {
		out = append(out, byte(repeat))
	}
	for _, horizontal := range []bool{true, false} {
		prev := 0
		for _, p := range pts {
			v := p.y
			if horizontal {
				v = p.x
			}
			d := v - prev
			prev = v
			switch {
			case d == 0:
			case d > -256 && d < 256:
				if d < 0 {
					d = -d
				}
				out = append(out, byte(d)) //nolint:gosec // under 256
			default:
				out = be.AppendUint16(out, uint16(d)) //nolint:gosec // truncated as the format stores it
			}
		}
	}
	return out
}
