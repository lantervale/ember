// Package varfonttest builds a variable font for tests: Go Regular given
// a weight axis that grows its outlines along their normals and a width
// axis that stretches them, so a test can set an instance drawn by
// ember beside the same file drawn by a browser.
package varfonttest

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/bits"
	"slices"
	"sort"
	"unicode/utf16"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/lantervale/ember/internal/varfont"
)

// The font's axes and how far each moves the glyphs at its ends, in font
// units of Go Regular's 2048 a em.
const (
	// Bolder is how far wght 900 moves an outline out along its normals,
	// Lighter how far wght 100 moves it in.
	Bolder, Lighter = 60, 24
	// Wider is how much of its x wdth 150 adds to each point, and wdth
	// 50 takes away.
	Wider = 0.4
	// Taller is how much of its y wght's intermediate region adds to each
	// point at its peak, a weight of 650.
	Taller = 0.08
)

// Options change the font built.
type Options struct {
	// NoHVAR leaves HVAR out: advances come from gvar's phantom points.
	NoHVAR bool
	// NoAvar leaves avar out: wght maps linearly.
	NoAvar bool
}

// Weighty is Go Regular made variable: wght from 100 to 900, 400 its
// default, and wdth from 50 to 150, 100 its default, with named
// instances Light (300), Bold (700) and Condensed Bold (700, 75). avar
// bends wght so that 650 is three quarters of the way to 900.
func Weighty() []byte { return Build(Options{}) }

// Build is Weighty as opts has it.
func Build(opts Options) []byte {
	tables := directory(goregular.TTF)
	f, err := varfont.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	}
	glyf, loca := tables["glyf"], tables["loca"]
	head := tables["head"]
	long := binary.BigEndian.Uint16(head[50:]) != 0
	n := f.NumGlyphs()
	advances := make([]float32, n)
	glyphs := make([][]byte, n)
	for g := range n {
		gid := uint16(g) //nolint:gosec // under the font's glyph count
		advances[g] = f.Advance(gid, nil)
		data := glyphData(glyf, loca, long, g)
		switch {
		case len(data) < 10:
			glyphs[g] = variations(nil, nil, advances[g])
		case int16(binary.BigEndian.Uint16(data)) < 0: //nolint:gosec // a signed count
			glyphs[g] = compositeVariations(data[10:], advances[g])
		default:
			pts, ends, err := f.Outline(gid, nil)
			if err != nil {
				panic(err)
			}
			glyphs[g] = variations(pts, ends, advances[g])
		}
	}
	tables["fvar"] = fvar()
	tables["gvar"] = gvar(glyphs)
	if !opts.NoAvar {
		tables["avar"] = avar()
	}
	if !opts.NoHVAR {
		tables["HVAR"] = hvar(advances)
	}
	tables["name"] = names(tables["name"])
	return build(tables)
}

func directory(ttf []byte) map[string][]byte {
	be := binary.BigEndian
	out := map[string][]byte{}
	for i := range int(be.Uint16(ttf[4:])) {
		rec := ttf[12+16*i:]
		off, size := be.Uint32(rec[8:]), be.Uint32(rec[12:])
		out[string(rec[:4])] = slices.Clone(ttf[off : off+size])
	}
	return out
}

func glyphData(glyf, loca []byte, long bool, g int) []byte {
	be := binary.BigEndian
	if long {
		return glyf[be.Uint32(loca[4*g:]):be.Uint32(loca[4*g+4:])]
	}
	return glyf[2*int(be.Uint16(loca[2*g:])) : 2*int(be.Uint16(loca[2*g+2:]))]
}

// tuple is one tuple variation of a glyph: its peak (shared index, or
// embedded), intermediate region, the points it names (nil for all) and
// their deltas.
type tuple struct {
	shared      int
	peak        [2]float32
	start, end  [2]float32
	inter       bool
	points      []int
	xs, ys      []int
	sharedPeaks bool
}

// shared peaks: wght at 900 and at 100.
var sharedPeaks = [][2]float32{{1, 0}, {-1, 0}} //nolint:gochecknoglobals // the font's shared tuples

// variations are a simple glyph's tuples: wght's ends grow or shrink it
// along its normals, its intermediate region makes it taller, and wdth's
// ends stretch it, at 150 naming only its points on the curve (the rest
// inferred).
func variations(pts []varfont.Point, ends []int, advance float32) []byte {
	n := len(pts)
	normals := normalsOf(pts, ends)
	grow := func(e float32) tuple {
		t := tuple{xs: make([]int, n+4), ys: make([]int, n+4), sharedPeaks: true}
		for i, nv := range normals {
			t.xs[i] = int(math.Round(float64(nv[0]*e + e)))
			t.ys[i] = int(math.Round(float64(nv[1] * e)))
		}
		t.xs[n+1] = int(math.Round(float64(2 * e)))
		return t
	}
	bold, light := grow(Bolder), grow(-Lighter)
	light.shared = 1
	tall := tuple{peak: [2]float32{0.5, 0}, start: [2]float32{0, 0}, end: [2]float32{1, 0}, inter: true,
		xs: make([]int, n+4), ys: make([]int, n+4)}
	for i, p := range pts {
		tall.ys[i] = int(math.Round(float64(p.Y * Taller)))
	}
	wide := tuple{peak: [2]float32{0, 1}}
	narrow := tuple{peak: [2]float32{0, -1}, xs: make([]int, n+4), ys: make([]int, n+4)}
	for i, p := range pts {
		if p.On {
			wide.points = append(wide.points, i)
			wide.xs = append(wide.xs, int(math.Round(float64(p.X*Wider))))
			wide.ys = append(wide.ys, 0)
		}
		narrow.xs[i] = int(math.Round(float64(-p.X * Wider)))
	}
	wide.points = append(wide.points, n, n+1, n+2, n+3)
	wide.xs = append(wide.xs, 0, int(math.Round(float64(advance*Wider))), 0, 0)
	wide.ys = append(wide.ys, 0, 0, 0, 0)
	narrow.xs[n+1] = int(math.Round(float64(-advance * Wider)))
	return glyphVariationData([]tuple{bold, light, tall, wide, narrow})
}

// compositeVariations are a composite glyph's tuples: its components
// are varied by their own, so only its offsets move with wdth, and its
// advance with both axes.
func compositeVariations(data []byte, advance float32) []byte {
	be := binary.BigEndian
	var offsets [][2]float32
	for at := 0; ; {
		flags := be.Uint16(data[at:])
		at += 4
		var dx, dy float32
		if flags&1 != 0 {
			dx, dy = float32(int16(be.Uint16(data[at:]))), float32(int16(be.Uint16(data[at+2:]))) //nolint:gosec // signed
			at += 4
		} else {
			dx, dy = float32(int8(data[at])), float32(int8(data[at+1])) //nolint:gosec // signed
			at += 2
		}
		switch {
		case flags&0x08 != 0:
			at += 2
		case flags&0x40 != 0:
			at += 4
		case flags&0x80 != 0:
			at += 8
		}
		offsets = append(offsets, [2]float32{dx, dy})
		if flags&0x20 == 0 {
			break
		}
	}
	n := len(offsets)
	axis := func(peak [2]float32, by float32) tuple {
		t := tuple{peak: peak, xs: make([]int, n+4), ys: make([]int, n+4)}
		for i, o := range offsets {
			t.xs[i] = int(math.Round(float64(o[0] * by)))
		}
		t.xs[n+1] = int(math.Round(float64(advance * by)))
		return t
	}
	bold := tuple{sharedPeaks: true, xs: make([]int, n+4), ys: make([]int, n+4)}
	bold.xs[n+1] = 2 * Bolder
	light := tuple{sharedPeaks: true, shared: 1, xs: make([]int, n+4), ys: make([]int, n+4)}
	light.xs[n+1] = -2 * Lighter
	return glyphVariationData([]tuple{bold, light, axis([2]float32{0, 1}, Wider), axis([2]float32{0, -1}, -Wider)})
}

// normalsOf is each point's outward normal: the mean of its two edges',
// the outer contours running clockwise.
func normalsOf(pts []varfont.Point, ends []int) [][2]float32 {
	out := make([][2]float32, len(pts))
	start := 0
	for _, end := range ends {
		k := end - start + 1
		for i := start; i <= end; i++ {
			prev, next := pts[start+(i-start+k-1)%k], pts[start+(i-start+1)%k]
			a := edgeNormal(prev, pts[i])
			b := edgeNormal(pts[i], next)
			x, y := a[0]+b[0], a[1]+b[1]
			if l := float32(math.Hypot(float64(x), float64(y))); l > 1e-3 {
				out[i] = [2]float32{x / l, y / l}
			}
		}
		start = end + 1
	}
	return out
}

func edgeNormal(a, b varfont.Point) [2]float32 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := float32(math.Hypot(float64(dx), float64(dy)))
	if l == 0 {
		return [2]float32{}
	}
	return [2]float32{-dy / l, dx / l}
}

const (
	embeddedPeak       = 0x8000
	intermediateRegion = 0x4000
	privatePointNumber = 0x2000
)

func glyphVariationData(tuples []tuple) []byte {
	var headers, serial bytes.Buffer
	for _, t := range tuples {
		var body bytes.Buffer
		if t.points != nil {
			packPoints(&body, t.points)
		}
		packDeltas(&body, t.xs)
		packDeltas(&body, t.ys)
		index := uint16(t.shared) //nolint:gosec // 0 or 1
		if !t.sharedPeaks {
			index = embeddedPeak
		}
		if t.inter {
			index |= intermediateRegion
		}
		if t.points != nil {
			index |= privatePointNumber
		}
		put16(&headers, uint16(body.Len())) //nolint:gosec // a glyph's deltas are under 64 KiB
		put16(&headers, index)
		if !t.sharedPeaks {
			for _, v := range t.peak {
				put16(&headers, f2dot14(v))
			}
		}
		if t.inter {
			for _, v := range t.start {
				put16(&headers, f2dot14(v))
			}
			for _, v := range t.end {
				put16(&headers, f2dot14(v))
			}
		}
		serial.Write(body.Bytes())
	}
	var out bytes.Buffer
	put16(&out, uint16(len(tuples)))     //nolint:gosec // a handful
	put16(&out, uint16(4+headers.Len())) //nolint:gosec // small
	out.Write(headers.Bytes())
	out.Write(serial.Bytes())
	if out.Len()%2 != 0 {
		out.WriteByte(0)
	}
	return out.Bytes()
}

func packPoints(b *bytes.Buffer, points []int) {
	n := len(points)
	if n < 0x80 {
		b.WriteByte(byte(n))
	} else {
		put16(b, uint16(n)|0x8000) //nolint:gosec // under 32768 points
	}
	last := 0
	for i := 0; i < n; {
		run := min(n-i, 128)
		b.WriteByte(0x80 | byte(run-1))
		for _, p := range points[i : i+run] {
			put16(b, uint16(p-last)) //nolint:gosec // ascending point numbers
			last = p
		}
		i += run
	}
}

func packDeltas(b *bytes.Buffer, ds []int) {
	for i := 0; i < len(ds); {
		run := min(len(ds)-i, 64)
		b.WriteByte(0x40 | byte(run-1))
		for _, d := range ds[i : i+run] {
			put16(b, uint16(int16(d))) //nolint:gosec // deltas within a glyph's units
		}
		i += run
	}
}

func put16(b *bytes.Buffer, v uint16) { b.Write([]byte{byte(v >> 8), byte(v)}) }

func put32(b *bytes.Buffer, v uint32) { put16(b, uint16(v>>16)); put16(b, uint16(v)) } //nolint:gosec // halves

func f2dot14(v float32) uint16 { return uint16(int16(math.Round(float64(v) * 16384))) } //nolint:gosec // -2 to 2

func fixed(v float32) uint32 { return uint32(int32(math.Round(float64(v) * 65536))) } //nolint:gosec // a 16.16 number

// The name IDs of the axes and instances.
const (
	nameWeight = 256 + iota
	nameWidth
	nameLight
	nameBold
	nameCondensedBold
)

func fvar() []byte {
	var b bytes.Buffer
	put32(&b, 0x00010000)
	put16(&b, 16)
	put16(&b, 2)
	put16(&b, 2)
	put16(&b, 20)
	put16(&b, 3)
	put16(&b, 12)
	for _, a := range []struct {
		tag         string
		lo, def, hi float32
		name        uint16
	}{{"wght", 100, 400, 900, nameWeight}, {"wdth", 50, 100, 150, nameWidth}} {
		b.WriteString(a.tag)
		put32(&b, fixed(a.lo))
		put32(&b, fixed(a.def))
		put32(&b, fixed(a.hi))
		put16(&b, 0)
		put16(&b, a.name)
	}
	for _, in := range []struct {
		name uint16
		at   [2]float32
	}{{nameLight, [2]float32{300, 100}}, {nameBold, [2]float32{700, 100}}, {nameCondensedBold, [2]float32{700, 75}}} {
		put16(&b, in.name)
		put16(&b, 0)
		put32(&b, fixed(in.at[0]))
		put32(&b, fixed(in.at[1]))
	}
	return b.Bytes()
}

func avar() []byte {
	var b bytes.Buffer
	put32(&b, 0x00010000)
	put16(&b, 0)
	put16(&b, 2)
	for _, m := range [][][2]float32{{{-1, -1}, {0, 0}, {0.5, 0.75}, {1, 1}}, {{-1, -1}, {0, 0}, {1, 1}}} {
		put16(&b, uint16(len(m))) //nolint:gosec // a few points
		for _, p := range m {
			put16(&b, f2dot14(p[0]))
			put16(&b, f2dot14(p[1]))
		}
	}
	return b.Bytes()
}

func gvar(glyphs [][]byte) []byte {
	var b bytes.Buffer
	shared := 20 + 4*(len(glyphs)+1)
	data := shared + 4*len(sharedPeaks)
	put32(&b, 0x00010000)
	put16(&b, 2)
	put16(&b, uint16(len(sharedPeaks))) //nolint:gosec // two
	put32(&b, uint32(shared))           //nolint:gosec // small
	put16(&b, uint16(len(glyphs)))      //nolint:gosec // the font's glyph count
	put16(&b, 1)
	put32(&b, uint32(data)) //nolint:gosec // small
	at := 0
	for _, g := range glyphs {
		put32(&b, uint32(at)) //nolint:gosec // under the table's size
		at += len(g)
	}
	put32(&b, uint32(at)) //nolint:gosec // as above
	for _, p := range sharedPeaks {
		put16(&b, f2dot14(p[0]))
		put16(&b, f2dot14(p[1]))
	}
	for _, g := range glyphs {
		b.Write(g)
	}
	return b.Bytes()
}

// hvar gives each glyph its advance's deltas at wght's and wdth's ends,
// as gvar's phantom points move it.
func hvar(advances []float32) []byte {
	regions := [][2][3]float32{{{-1, -1, 0}, {}}, {{0, 1, 1}, {}}, {{}, {-1, -1, 0}}, {{}, {0, 1, 1}}}
	var store bytes.Buffer
	put16(&store, 1)
	put32(&store, 12)
	put16(&store, 1)
	dataAt := 12 + 4 + 12*len(regions)
	put32(&store, uint32(dataAt)) //nolint:gosec // small
	put16(&store, 2)
	put16(&store, uint16(len(regions))) //nolint:gosec // four
	for _, r := range regions {
		for _, axis := range r {
			for _, v := range axis {
				put16(&store, f2dot14(v))
			}
		}
	}
	put16(&store, uint16(len(advances))) //nolint:gosec // the font's glyph count
	put16(&store, uint16(len(regions)))  //nolint:gosec // four
	put16(&store, uint16(len(regions)))  //nolint:gosec // four
	for i := range regions {
		put16(&store, uint16(i)) //nolint:gosec // four
	}
	for _, a := range advances {
		wide := int(math.Round(float64(a * Wider)))
		for _, d := range []int{-2 * Lighter, 2 * Bolder, -wide, wide} {
			put16(&store, uint16(int16(d))) //nolint:gosec // within a glyph's units
		}
	}
	var b bytes.Buffer
	put32(&b, 0x00010000)
	put32(&b, 20)
	put32(&b, 0)
	put32(&b, 0)
	put32(&b, 0)
	b.Write(store.Bytes())
	return b.Bytes()
}

// names is the name table with the names of the axes and instances
// added, for Windows' platform in US English.
func names(table []byte) []byte {
	be := binary.BigEndian
	count, storageAt := int(be.Uint16(table[2:])), int(be.Uint16(table[4:]))
	type record struct {
		head [10]byte
		text []byte
	}
	var recs []record
	for i := range count {
		r := table[6+12*i:]
		var rec record
		copy(rec.head[:8], r[:8])
		length, off := int(be.Uint16(r[8:])), int(be.Uint16(r[10:]))
		rec.text = table[storageAt+off : storageAt+off+length]
		recs = append(recs, rec)
	}
	for id, text := range map[uint16]string{nameWeight: "Weight", nameWidth: "Width", nameLight: "Light",
		nameBold: "Bold", nameCondensedBold: "Condensed Bold"} {
		var rec record
		be.PutUint16(rec.head[0:], 3)
		be.PutUint16(rec.head[2:], 1)
		be.PutUint16(rec.head[4:], 0x409)
		be.PutUint16(rec.head[6:], id)
		for _, u := range utf16.Encode([]rune(text)) {
			rec.text = be.AppendUint16(rec.text, u)
		}
		recs = append(recs, rec)
	}
	sort.SliceStable(recs, func(a, b int) bool { return bytes.Compare(recs[a].head[:8], recs[b].head[:8]) < 0 })
	var b, storage bytes.Buffer
	put16(&b, 0)
	put16(&b, uint16(len(recs)))      //nolint:gosec // a few dozen
	put16(&b, uint16(6+12*len(recs))) //nolint:gosec // small
	for _, r := range recs {
		b.Write(r.head[:8])
		put16(&b, uint16(len(r.text)))   //nolint:gosec // short names
		put16(&b, uint16(storage.Len())) //nolint:gosec // small
		storage.Write(r.text)
	}
	b.Write(storage.Bytes())
	return b.Bytes()
}

// build writes the sfnt file of tables, sorted by tag, each on a 4-byte
// boundary, head's checkSumAdjustment set.
func build(tables map[string][]byte) []byte {
	tags := make([]string, 0, len(tables))
	for t := range tables {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	n := len(tags)
	be := binary.BigEndian
	head := 12 + 16*n
	out := make([]byte, head)
	be.PutUint32(out, 0x00010000)
	sel := bits.Len(uint(n)) - 1
	be.PutUint16(out[4:], uint16(n))             //nolint:gosec // a few dozen tables
	be.PutUint16(out[6:], uint16(16<<sel))       //nolint:gosec // as the format has it
	be.PutUint16(out[8:], uint16(sel))           //nolint:gosec // under 16
	be.PutUint16(out[10:], uint16(16*n-16<<sel)) //nolint:gosec // as the format has it
	headAt := 0
	for i, tag := range tags {
		data := tables[tag]
		if tag == "head" {
			be.PutUint32(data[8:], 0)
			headAt = len(out)
		}
		rec := out[12+16*i:]
		copy(rec, tag)
		be.PutUint32(rec[4:], checksum(data))
		be.PutUint32(rec[8:], uint32(len(out)))   //nolint:gosec // small
		be.PutUint32(rec[12:], uint32(len(data))) //nolint:gosec // small
		out = append(out, data...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	be.PutUint32(out[headAt+8:], 0xb1b0afba-checksum(out))
	return out
}

func checksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var w [4]byte
		copy(w[:], data[i:])
		sum += binary.BigEndian.Uint32(w[:])
	}
	return sum
}
