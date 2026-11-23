package colourtest

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

// Paint is a COLR version 1 paint table.
type Paint interface{ format() uint8 }

// Stop is a colour line's stop: palette entry Palette at Alpha.
type Stop struct {
	Offset  float32
	Palette uint16
	Alpha   float32
}

// Extend is how a colour line goes on past its ends.
type Extend uint8

const (
	Pad Extend = iota
	Repeat
	Reflect
)

// Line is a colour line.
type Line struct {
	Extend Extend
	Stops  []Stop
}

type (
	// Layers paints each of its paints over the ones before.
	Layers []Paint
	// Solid is palette entry Palette at Alpha.
	Solid struct {
		Palette uint16
		Alpha   float32
	}
	// Linear is a linear gradient from P0 to P1, its lines parallel to
	// P0–P2.
	Linear struct {
		Line                   Line
		X0, Y0, X1, Y1, X2, Y2 int16
	}
	// Radial is a gradient between two circles.
	Radial struct {
		Line           Line
		X0, Y0, X1, Y1 int16
		R0, R1         uint16
	}
	// Sweep is a gradient round a centre, from Start to End in half
	// turns counter-clockwise.
	Sweep struct {
		Line       Line
		X, Y       int16
		Start, End float32
	}
	// Glyph is Paint clipped to the outline of Shape's glyph.
	Glyph struct {
		Shape rune
		Paint Paint
	}
	// ColrGlyph is the paint of the base glyph of Base.
	ColrGlyph struct{ Base rune }
	// Transform is Paint through xx, yx, xy, yy, dx, dy.
	Transform struct {
		M     [6]float32
		Paint Paint
	}
	// Translate is Paint moved.
	Translate struct {
		DX, DY int16
		Paint  Paint
	}
	// Scale is Paint scaled, about (CX, CY) when Around.
	Scale struct {
		SX, SY float32
		Around bool
		CX, CY int16
		Paint  Paint
	}
	// Rotate is Paint turned by Angle half turns counter-clockwise.
	Rotate struct {
		Angle  float32
		Around bool
		CX, CY int16
		Paint  Paint
	}
	// Skew is Paint skewed by X and Y half turns.
	Skew struct {
		X, Y   float32
		Around bool
		CX, CY int16
		Paint  Paint
	}
	// Composite is Source composited over Backdrop by Mode.
	Composite struct {
		Source, Backdrop Paint
		Mode             uint8
	}
	// Var is a paint's variable format with the same values.
	Var struct{ Paint Paint }
)

func (Layers) format() uint8    { return 1 }
func (Solid) format() uint8     { return 2 }
func (Linear) format() uint8    { return 4 }
func (Radial) format() uint8    { return 6 }
func (Sweep) format() uint8     { return 8 }
func (Glyph) format() uint8     { return 10 }
func (ColrGlyph) format() uint8 { return 11 }
func (Transform) format() uint8 { return 12 }
func (Translate) format() uint8 { return 14 }
func (s Scale) format() uint8 {
	switch {
	case s.SX == s.SY && s.Around:
		return 22
	case s.SX == s.SY:
		return 20
	case s.Around:
		return 18
	}
	return 16
}

func (r Rotate) format() uint8 {
	if r.Around {
		return 26
	}
	return 24
}

func (s Skew) format() uint8 {
	if s.Around {
		return 30
	}
	return 28
}
func (Composite) format() uint8 { return 32 }
func (v Var) format() uint8     { return v.Paint.format() + 1 }

// encode writes p with the paints it points to after it, and the
// paints of its Layers appended to layers.
//
//nolint:funlen // a case for each kind of paint
func encode(p Paint, layers *[]Paint) []byte {
	var b bytes.Buffer
	variable := false
	if v, ok := p.(Var); ok {
		p, variable = v.Paint, true
	}
	format := p.format()
	if variable {
		format++
	}
	put8(&b, format)
	var children [][]byte
	var holes []int
	child := func(c Paint) {
		holes = append(holes, b.Len())
		put24(&b, 0)
		children = append(children, encode(c, layers))
	}
	line := func(l Line) {
		holes = append(holes, b.Len())
		put24(&b, 0)
		children = append(children, colorLine(l, variable))
	}
	varIndex := func() {
		if variable {
			put32(&b, 0xFFFFFFFF)
		}
	}
	switch p := p.(type) {
	case Layers:
		put8(&b, uint8(len(p)))         //nolint:gosec // a few
		put32(&b, uint32(len(*layers))) //nolint:gosec // a few
		*layers = append(*layers, p...)
	case Solid:
		put16(&b, p.Palette)
		put16(&b, f2dot14(p.Alpha))
		varIndex()
	case Linear:
		line(p.Line)
		for _, v := range []int16{p.X0, p.Y0, p.X1, p.Y1, p.X2, p.Y2} {
			put16(&b, uint16(v)) //nolint:gosec // signed
		}
		varIndex()
	case Radial:
		line(p.Line)
		put16(&b, uint16(p.X0)) //nolint:gosec // signed
		put16(&b, uint16(p.Y0)) //nolint:gosec // signed
		put16(&b, p.R0)
		put16(&b, uint16(p.X1)) //nolint:gosec // signed
		put16(&b, uint16(p.Y1)) //nolint:gosec // signed
		put16(&b, p.R1)
		varIndex()
	case Sweep:
		line(p.Line)
		put16(&b, uint16(p.X)) //nolint:gosec // signed
		put16(&b, uint16(p.Y)) //nolint:gosec // signed
		put16(&b, f2dot14(p.Start))
		put16(&b, f2dot14(p.End))
		varIndex()
	case Glyph:
		child(p.Paint)
		put16(&b, GlyphID(p.Shape))
	case ColrGlyph:
		put16(&b, GlyphID(p.Base))
	case Transform:
		child(p.Paint)
		holes = append(holes, b.Len())
		put24(&b, 0)
		var m bytes.Buffer
		for _, v := range p.M {
			put32(&m, fixed16(v))
		}
		if variable {
			put32(&m, 0xFFFFFFFF)
		}
		children = append(children, m.Bytes())
	case Translate:
		child(p.Paint)
		put16(&b, uint16(p.DX)) //nolint:gosec // signed
		put16(&b, uint16(p.DY)) //nolint:gosec // signed
		varIndex()
	case Scale:
		child(p.Paint)
		put16(&b, f2dot14(p.SX))
		if p.SX != p.SY {
			put16(&b, f2dot14(p.SY))
		}
		if p.Around {
			put16(&b, uint16(p.CX)) //nolint:gosec // signed
			put16(&b, uint16(p.CY)) //nolint:gosec // signed
		}
		varIndex()
	case Rotate:
		child(p.Paint)
		put16(&b, f2dot14(p.Angle))
		if p.Around {
			put16(&b, uint16(p.CX)) //nolint:gosec // signed
			put16(&b, uint16(p.CY)) //nolint:gosec // signed
		}
		varIndex()
	case Skew:
		child(p.Paint)
		put16(&b, f2dot14(p.X))
		put16(&b, f2dot14(p.Y))
		if p.Around {
			put16(&b, uint16(p.CX)) //nolint:gosec // signed
			put16(&b, uint16(p.CY)) //nolint:gosec // signed
		}
		varIndex()
	case Composite:
		child(p.Source)
		put8(&b, p.Mode)
		child(p.Backdrop)
	default:
		panic(fmt.Sprintf("colourtest: paint %T", p))
	}
	out := b.Bytes()
	for i, h := range holes {
		at := len(out)
		out[h], out[h+1], out[h+2] = byte(at>>16), byte(at>>8), byte(at) //nolint:gosec // its bytes
		out = append(out, children[i]...)
	}
	return out
}

func colorLine(l Line, variable bool) []byte {
	var b bytes.Buffer
	put8(&b, uint8(l.Extend))
	put16(&b, uint16(len(l.Stops))) //nolint:gosec // a few
	for _, s := range l.Stops {
		put16(&b, f2dot14(s.Offset))
		put16(&b, s.Palette)
		put16(&b, f2dot14(s.Alpha))
		if variable {
			put32(&b, 0xFFFFFFFF)
		}
	}
	return b.Bytes()
}

// svgTable is an SVG table of one document a glyph.
func svgTable(docs map[rune]string, compress bool) []byte {
	runes := sortedRunes(docs)
	var data [][]byte
	for _, r := range runes {
		d := []byte(docs[r])
		if compress {
			var z bytes.Buffer
			w := gzip.NewWriter(&z)
			_, _ = w.Write(d)
			_ = w.Close()
			d = z.Bytes()
		}
		data = append(data, d)
	}
	var b bytes.Buffer
	put16(&b, 0)
	put32(&b, 10)
	put32(&b, 0)
	put16(&b, uint16(len(runes))) //nolint:gosec // a few
	at := 2 + 12*len(runes)
	for i, r := range runes {
		put16(&b, GlyphID(r))
		put16(&b, GlyphID(r))
		put32(&b, uint32(at))           //nolint:gosec // small
		put32(&b, uint32(len(data[i]))) //nolint:gosec // small
		at += len(data[i])
	}
	for _, d := range data {
		b.Write(d)
	}
	return b.Bytes()
}
