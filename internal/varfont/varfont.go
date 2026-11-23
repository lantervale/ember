// Package varfont reads what golang.org/x/image's sfnt leaves out of a
// TrueType font: the axes of a variable font (fvar, avar) and its glyphs
// and advances at any place on them (gvar, HVAR), for drawing an
// instance other than the default. It is pure Go, and the bytes may be
// a stranger's: every length, offset and count is checked, and a table
// that says anything else is an error or left out, never a panic.
package varfont

import (
	"encoding/binary"
	"errors"
	"math"
)

var (
	errShort  = errors.New("varfont: the font ends early")
	errTables = errors.New("varfont: the font has no glyf, loca, head, hhea, hmtx or maxp")
	errGlyph  = errors.New("varfont: a glyph is wrong")
)

// Axis is one of a variable font's axes, in its own units: wght in
// weights, wdth in percents, slnt in degrees, opsz in points.
type Axis struct {
	Tag               string
	Min, Default, Max float32
	Hidden            bool
	nameID            uint16
}

// Instance is a named instance: a place on every axis, in the axes'
// units, in the order of Font.Axes.
type Instance struct {
	Coords []float32
	nameID uint16
}

// Font is a TrueType font's glyphs, advances and, when it is variable,
// its axes and variations.
type Font struct {
	Axes      []Axis
	Instances []Instance

	upem        uint16
	numGlyphs   int
	glyf, loca  []byte
	longLoca    bool
	hmtx        []byte
	numHMetrics int
	avar        [][]axisMap
	gvar        gvar
	hvar        *itemStore
	hvarMap     deltaSetMap
}

// Coords are a place on a font's axes normalized as the font varies
// over them: -1 at an axis's minimum, 0 at its default, 1 at its
// maximum, after avar's mapping. Nil is the default instance.
type Coords []float32

// Parse reads the TrueType font ttf. A font with no fvar has no Axes and
// draws as it is.
func Parse(ttf []byte) (*Font, error) {
	tables, err := directory(ttf)
	if err != nil {
		return nil, err
	}
	f := &Font{glyf: tables["glyf"], loca: tables["loca"], hmtx: tables["hmtx"]}
	head, hhea, maxp := tables["head"], tables["hhea"], tables["maxp"]
	if f.glyf == nil || f.loca == nil || f.hmtx == nil || len(head) < 54 || len(hhea) < 36 || len(maxp) < 6 {
		return nil, errTables
	}
	be := binary.BigEndian
	f.upem = be.Uint16(head[18:])
	f.longLoca = be.Uint16(head[50:]) != 0
	f.numGlyphs = int(be.Uint16(maxp[4:]))
	f.numHMetrics = int(be.Uint16(hhea[34:]))
	if f.upem == 0 || f.numHMetrics == 0 || len(f.hmtx) < 4*f.numHMetrics {
		return nil, errTables
	}
	if fvar := tables["fvar"]; fvar != nil {
		if err = f.readFvar(fvar); err != nil {
			return nil, err
		}
	}
	if len(f.Axes) == 0 {
		return f, nil
	}
	if avar := tables["avar"]; avar != nil {
		f.avar = readAvar(avar, len(f.Axes))
	}
	if gv := tables["gvar"]; gv != nil {
		f.gvar, err = readGvar(gv, len(f.Axes), f.numGlyphs)
		if err != nil {
			return nil, err
		}
	}
	if hv := tables["HVAR"]; len(hv) >= 20 {
		store, err := readItemStore(hv, int(be.Uint32(hv[4:])), len(f.Axes))
		if err == nil {
			f.hvar = store
			f.hvarMap = readDeltaSetMap(hv, int(be.Uint32(hv[8:])))
		}
	}
	return f, nil
}

// directory is the font's tables by tag.
func directory(ttf []byte) (map[string][]byte, error) {
	if len(ttf) < 12 {
		return nil, errShort
	}
	be := binary.BigEndian
	n := int(be.Uint16(ttf[4:]))
	if len(ttf) < 12+16*n {
		return nil, errShort
	}
	out := make(map[string][]byte, n)
	for i := range n {
		rec := ttf[12+16*i:]
		off, size := uint64(be.Uint32(rec[8:])), uint64(be.Uint32(rec[12:]))
		if off+size > uint64(len(ttf)) {
			return nil, errShort
		}
		out[string(rec[:4])] = ttf[off : off+size]
	}
	return out, nil
}

// Variable reports whether the font has axes.
func (f *Font) Variable() bool { return len(f.Axes) > 0 }

// UnitsPerEm is the font's units per em.
func (f *Font) UnitsPerEm() uint16 { return f.upem }

// NumGlyphs is how many glyphs the font has.
func (f *Font) NumGlyphs() int { return f.numGlyphs }

func (f *Font) readFvar(b []byte) error {
	be := binary.BigEndian
	if len(b) < 16 {
		return errShort
	}
	axesAt, axisCount, axisSize := int(be.Uint16(b[4:])), int(be.Uint16(b[8:])), int(be.Uint16(b[10:]))
	instCount, instSize := int(be.Uint16(b[12:])), int(be.Uint16(b[14:]))
	if axisSize < 20 || instSize < 4+4*axisCount || axesAt+axisCount*axisSize+instCount*instSize > len(b) {
		return errShort
	}
	for i := range axisCount {
		a := b[axesAt+i*axisSize:]
		axis := Axis{Tag: string(a[:4]), Min: fixed16(a[4:]), Default: fixed16(a[8:]), Max: fixed16(a[12:]),
			Hidden: be.Uint16(a[16:])&1 != 0, nameID: be.Uint16(a[18:])}
		axis.Min, axis.Max = min(axis.Min, axis.Default), max(axis.Max, axis.Default)
		f.Axes = append(f.Axes, axis)
	}
	at := axesAt + axisCount*axisSize
	for i := range instCount {
		in := b[at+i*instSize:]
		inst := Instance{nameID: be.Uint16(in), Coords: make([]float32, axisCount)}
		for j := range axisCount {
			inst.Coords[j] = fixed16(in[4+4*j:])
		}
		f.Instances = append(f.Instances, inst)
	}
	return nil
}

func fixed16(b []byte) float32 { return float32(int32(binary.BigEndian.Uint32(b))) / 65536 } //nolint:gosec // a 16.16 number

func f2dot14(b []byte) float32 { return float32(int16(binary.BigEndian.Uint16(b))) / 16384 } //nolint:gosec // a 2.14 number

// axisMap is one point of avar's mapping for an axis: from a normalized
// coordinate to another.
type axisMap struct{ from, to float32 }

// readAvar is avar's mappings, one list per axis, nil for an axis that
// maps none or a table that is wrong.
func readAvar(b []byte, axes int) [][]axisMap {
	be := binary.BigEndian
	if len(b) < 8 || int(be.Uint16(b[6:])) != axes {
		return nil
	}
	out := make([][]axisMap, axes)
	at := 8
	for i := range axes {
		if at+2 > len(b) {
			return nil
		}
		n := int(be.Uint16(b[at:]))
		at += 2
		if at+4*n > len(b) {
			return nil
		}
		for j := range n {
			out[i] = append(out[i], axisMap{f2dot14(b[at+4*j:]), f2dot14(b[at+4*j+2:])})
		}
		at += 4 * n
	}
	return out
}

// Normalize is the place user gives, one value an axis in the order of
// Axes (the default for one it lacks), as the font varies over it:
// clamped to each axis, made -1 to 1 about its default, mapped by avar,
// and rounded to the 2.14 numbers the font's tables are in.
func (f *Font) Normalize(user []float32) Coords {
	out := make(Coords, len(f.Axes))
	any := false
	for i, a := range f.Axes {
		v := a.Default
		if i < len(user) {
			v = min(max(user[i], a.Min), a.Max)
		}
		var n float32
		switch {
		case v < a.Default && a.Default > a.Min:
			n = (v - a.Default) / (a.Default - a.Min)
		case v > a.Default && a.Max > a.Default:
			n = (v - a.Default) / (a.Max - a.Default)
		}
		if f.avar != nil {
			n = mapAxis(f.avar[i], n)
		}
		out[i] = float32(math.Round(float64(n)*16384)) / 16384
		any = any || out[i] != 0
	}
	if !any {
		return nil
	}
	return out
}

// mapAxis is v through an axis's avar mapping, linear between its
// points.
func mapAxis(m []axisMap, v float32) float32 {
	if len(m) == 0 {
		return v
	}
	if v <= m[0].from {
		return v - m[0].from + m[0].to
	}
	for i := 1; i < len(m); i++ {
		if v <= m[i].from {
			a, b := m[i-1], m[i]
			if b.from == a.from {
				return b.to
			}
			return a.to + (v-a.from)*(b.to-a.to)/(b.from-a.from)
		}
	}
	last := m[len(m)-1]
	return v - last.from + last.to
}

// Tag is the index of the axis tagged tag, -1 when the font has none.
func (f *Font) Tag(tag string) int {
	for i, a := range f.Axes {
		if a.Tag == tag {
			return i
		}
	}
	return -1
}

// Advance is glyph g's advance at c, in font units: hmtx's, moved by
// HVAR or, without it, by gvar's phantom points.
func (f *Font) Advance(g uint16, c Coords) float32 {
	if int(g) >= f.numGlyphs {
		return 0
	}
	adv := float32(f.baseAdvance(g))
	if c == nil {
		return adv
	}
	if f.hvar != nil {
		outer, inner := f.hvarMap.at(int(g))
		return adv + f.hvar.delta(outer, inner, c)
	}
	o, err := f.glyph(g, c, 0)
	if err != nil {
		return adv
	}
	return o.phantom[1].x - o.phantom[0].x
}

// baseAdvance is g's advance in hmtx.
func (f *Font) baseAdvance(g uint16) uint16 {
	i := min(int(g), f.numHMetrics-1)
	return binary.BigEndian.Uint16(f.hmtx[4*i:])
}

// lsb is g's left side bearing in hmtx.
func (f *Font) lsb(g uint16) int16 {
	be := binary.BigEndian
	if int(g) < f.numHMetrics {
		return int16(be.Uint16(f.hmtx[4*int(g)+2:])) //nolint:gosec // a signed number
	}
	at := 4*f.numHMetrics + 2*(int(g)-f.numHMetrics)
	if at+2 > len(f.hmtx) {
		return 0
	}
	return int16(be.Uint16(f.hmtx[at:])) //nolint:gosec // a signed number
}
