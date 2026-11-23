// Package colourtest builds colour fonts for tests: Go Regular given
// COLR and CPAL, sbix, CBDT and CBLC or SVG tables whose glyphs draw
// letters in colour, so a test can set them drawn by ember beside the
// same files drawn by a browser.
package colourtest

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/bits"
	"slices"
	"sort"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

// Options are the colour tables of a font built.
type Options struct {
	// Layers are COLR version 0's base glyphs, by the letter each draws.
	Layers map[rune][]Layer
	// Paints are COLR version 1's, by letter.
	Paints map[rune]Paint
	// Clips are COLR version 1's clip boxes, by letter: xMin, yMin, xMax,
	// yMax in font units.
	Clips map[rune][4]int16
	// Palettes are CPAL's, each as long as the first; Types their flags
	// (CPAL version 1 when given): 1 for light backgrounds, 2 for dark.
	Palettes [][]color.NRGBA
	Types    []uint32
	// Strikes are sbix's bitmaps: by ppem, the PNG each letter draws.
	Strikes map[int]map[rune]Bitmap
	// Bitmaps are CBDT's, at ppem BitmapPPEM.
	Bitmaps    map[rune]Bitmap
	BitmapPPEM int
	// SVG are the SVG table's documents, by letter, each drawing the
	// element glyphN; Gzip compresses them.
	SVG  map[rune]string
	Gzip bool
	// NoOutlines leaves glyf and loca out, as a font of bitmaps alone.
	NoOutlines bool
	// Family is the font's family name in its name table, kept when "".
	Family string
	// Advances are the letters' advances in place of Go Regular's, in
	// font units.
	Advances map[rune]uint16
}

// Layer is a COLR version 0 layer: the outline of the glyph of Shape in
// palette entry Palette, 0xFFFF for the text's colour.
type Layer struct {
	Shape   rune
	Palette uint16
}

// Bitmap is a PNG drawn for a letter: X and Y place its bottom-left
// (sbix) or top-left (CBDT, Y the bearing above the baseline) in its
// pixels from the glyph's origin.
type Bitmap struct {
	PNG  []byte
	X, Y int
}

// Foreground is the palette index standing for the text's colour.
const Foreground = 0xFFFF

// GlyphID is the glyph of r in Go Regular.
func GlyphID(r rune) uint16 {
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	}
	var buf sfnt.Buffer
	g, err := f.GlyphIndex(&buf, r)
	if err != nil {
		panic(err)
	}
	return uint16(g)
}

// Build is Go Regular with opts' colour tables.
func Build(opts Options) []byte {
	tables := Tables(goregular.TTF)
	numGlyphs := int(binary.BigEndian.Uint16(tables["maxp"][4:]))
	if len(opts.Layers) > 0 || len(opts.Paints) > 0 {
		tables["COLR"] = colr(opts)
	}
	if len(opts.Palettes) > 0 {
		tables["CPAL"] = cpal(opts.Palettes, opts.Types)
	}
	if len(opts.Strikes) > 0 {
		tables["sbix"] = sbix(opts.Strikes, numGlyphs)
	}
	if len(opts.Bitmaps) > 0 {
		tables["CBLC"], tables["CBDT"] = cblc(opts.Bitmaps, opts.BitmapPPEM)
	}
	if len(opts.SVG) > 0 {
		tables["SVG "] = svgTable(opts.SVG, opts.Gzip)
	}
	if opts.NoOutlines {
		delete(tables, "glyf")
		delete(tables, "loca")
		delete(tables, "fpgm")
		delete(tables, "prep")
		delete(tables, "cvt ")
		delete(tables, "gasp")
	}
	for r, a := range opts.Advances {
		binary.BigEndian.PutUint16(tables["hmtx"][4*int(GlyphID(r)):], a)
	}
	if opts.Family != "" {
		tables["name"] = Names(opts.Family)
	}
	return Assemble(tables)
}

// Collection is a TrueType collection of fonts, in order.
func Collection(fonts ...[]byte) []byte {
	be := binary.BigEndian
	head := 12 + 4*len(fonts)
	out := make([]byte, head, head+len(fonts)*1024)
	copy(out, "ttcf")
	be.PutUint32(out[4:], 0x00010000)
	be.PutUint32(out[8:], uint32(len(fonts))) //nolint:gosec // a few fonts
	for i, f := range fonts {
		at := len(out)
		be.PutUint32(out[12+4*i:], uint32(at)) //nolint:gosec // small
		n := int(be.Uint16(f[4:]))
		dir := slices.Clone(f[:12+16*n])
		out = append(out, dir...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
		for t := range n {
			rec := f[12+16*t:]
			off, size := be.Uint32(rec[8:]), be.Uint32(rec[12:])
			be.PutUint32(out[at+12+16*t+8:], uint32(len(out))) //nolint:gosec // small
			out = append(out, f[off:off+size]...)
			for len(out)%4 != 0 {
				out = append(out, 0)
			}
		}
	}
	return out
}

// Tables are the tables of the font file ttf, by tag.
func Tables(ttf []byte) map[string][]byte {
	be := binary.BigEndian
	out := map[string][]byte{}
	for i := range int(be.Uint16(ttf[4:])) {
		rec := ttf[12+16*i:]
		off, size := be.Uint32(rec[8:]), be.Uint32(rec[12:])
		out[string(rec[:4])] = slices.Clone(ttf[off : off+size])
	}
	return out
}

// Assemble writes the sfnt file of tables, sorted by tag, each on a
// 4-byte boundary, head's checkSumAdjustment set; CFF outlines when
// tables has a "CFF " or "CFF2" one.
func Assemble(tables map[string][]byte) []byte {
	tags := make([]string, 0, len(tables))
	for t := range tables {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	n := len(tags)
	be := binary.BigEndian
	head := 12 + 16*n
	out := make([]byte, head, head+1024)
	be.PutUint32(out, 0x00010000)
	if _, ok := tables["CFF "]; ok {
		copy(out, "OTTO")
	} else if _, ok = tables["CFF2"]; ok {
		copy(out, "OTTO")
	}
	sel := bits.Len(uint(n)) - 1
	be.PutUint16(out[4:], uint16(n)) //nolint:gosec // a few dozen tables
	be.PutUint16(out[6:], uint16(16<<sel))
	be.PutUint16(out[8:], uint16(sel))           //nolint:gosec // under 16
	be.PutUint16(out[10:], uint16(16*n-16<<sel)) //nolint:gosec // as the format has it
	headAt := -1
	for i, tag := range tags {
		data := slices.Clone(tables[tag])
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
	if headAt >= 0 {
		be.PutUint32(out[headAt+8:], 0xb1b0afba-checksum(out))
	}
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

// Names is a name table naming family, Regular.
func Names(family string) []byte {
	var b, storage bytes.Buffer
	ids := []uint16{1, 2, 4, 6}
	texts := []string{family, "Regular", family, family}
	put16(&b, 0)
	put16(&b, uint16(len(ids)))      //nolint:gosec // four
	put16(&b, uint16(6+12*len(ids))) //nolint:gosec // small
	for i, id := range ids {
		text := utf16BE(texts[i])
		put16(&b, 3)
		put16(&b, 1)
		put16(&b, 0x409)
		put16(&b, id)
		put16(&b, uint16(len(text)))     //nolint:gosec // short
		put16(&b, uint16(storage.Len())) //nolint:gosec // small
		storage.Write(text)
	}
	b.Write(storage.Bytes())
	return b.Bytes()
}

func utf16BE(s string) []byte {
	var out []byte
	for _, r := range s {
		if r > 0xffff {
			r = '?'
		}
		out = binary.BigEndian.AppendUint16(out, uint16(r))
	}
	return out
}

func put8(b *bytes.Buffer, v uint8)   { b.WriteByte(v) }
func put16(b *bytes.Buffer, v uint16) { b.Write(binary.BigEndian.AppendUint16(nil, v)) }

//nolint:gosec // its three bytes
func put24(b *bytes.Buffer, v uint32) { b.Write([]byte{byte(v >> 16), byte(v >> 8), byte(v)}) }
func put32(b *bytes.Buffer, v uint32) { b.Write(binary.BigEndian.AppendUint32(nil, v)) }

func f2dot14(v float32) uint16 { return uint16(int16(math.Round(float64(v) * 16384))) } //nolint:gosec // -2 to 2

func fixed16(v float32) uint32 { return uint32(int32(math.Round(float64(v) * 65536))) } //nolint:gosec // 16.16

func sortedRunes[V any](m map[rune]V) []rune {
	out := make([]rune, 0, len(m))
	for r := range m {
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b rune) int { return int(GlyphID(a)) - int(GlyphID(b)) })
	return out
}

func cpal(palettes [][]color.NRGBA, types []uint32) []byte {
	var b bytes.Buffer
	version := uint16(0)
	if len(types) > 0 {
		version = 1
	}
	entries := len(palettes[0])
	put16(&b, version)
	put16(&b, uint16(entries))               //nolint:gosec // a few
	put16(&b, uint16(len(palettes)))         //nolint:gosec // a few
	put16(&b, uint16(entries*len(palettes))) //nolint:gosec // a few
	head := 12 + 2*len(palettes)
	if version == 1 {
		head += 12
	}
	recordsAt := head
	put32(&b, uint32(recordsAt)) //nolint:gosec // small
	for i := range palettes {
		put16(&b, uint16(i*entries)) //nolint:gosec // small
	}
	if version == 1 {
		put32(&b, uint32(recordsAt+4*entries*len(palettes))) //nolint:gosec // small
		put32(&b, 0)
		put32(&b, 0)
	}
	for _, p := range palettes {
		for _, c := range p {
			b.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}
	for _, t := range types {
		put32(&b, t)
	}
	return b.Bytes()
}

func colr(opts Options) []byte {
	var b bytes.Buffer
	v1 := len(opts.Paints) > 0
	bases := sortedRunes(opts.Layers)
	var layers []Layer
	type base struct{ gid, first, n uint16 }
	var records []base
	for _, r := range bases {
		records = append(records, base{GlyphID(r), uint16(len(layers)), uint16(len(opts.Layers[r]))}) //nolint:gosec // a few
		layers = append(layers, opts.Layers[r]...)
	}
	head := 14
	if v1 {
		head = 34
	}
	baseAt := head
	layersAt := baseAt + 6*len(records)
	listAt := layersAt + 4*len(layers)
	version := uint16(0)
	if v1 {
		version = 1
	}
	put16(&b, version)
	put16(&b, uint16(len(records))) //nolint:gosec // a few
	put32(&b, offsetOr(baseAt, len(records)))
	put32(&b, offsetOr(layersAt, len(layers)))
	put16(&b, uint16(len(layers))) //nolint:gosec // a few
	var list, layerList, clips []byte
	if v1 {
		list, layerList = paintLists(opts.Paints)
		clips = clipList(opts.Clips)
		put32(&b, uint32(listAt))           //nolint:gosec // small
		put32(&b, uint32(listAt+len(list))) //nolint:gosec // small
		put32(&b, offsetOr(listAt+len(list)+len(layerList), len(clips)))
		put32(&b, 0)
		put32(&b, 0)
	}
	for _, r := range records {
		put16(&b, r.gid)
		put16(&b, r.first)
		put16(&b, r.n)
	}
	for _, l := range layers {
		put16(&b, GlyphID(l.Shape))
		put16(&b, l.Palette)
	}
	b.Write(list)
	b.Write(layerList)
	b.Write(clips)
	return b.Bytes()
}

func offsetOr(at, n int) uint32 {
	if n == 0 {
		return 0
	}
	return uint32(at) //nolint:gosec // small
}

// paintLists are the BaseGlyphList of paints and the LayerList their
// Layers use.
func paintLists(paints map[rune]Paint) ([]byte, []byte) {
	var layers []Paint
	runes := sortedRunes(paints)
	encoded := make([][]byte, len(runes))
	for i, r := range runes {
		encoded[i] = encode(paints[r], &layers)
	}
	var b bytes.Buffer
	put32(&b, uint32(len(runes))) //nolint:gosec // a few
	at := 4 + 6*len(runes)
	for i, r := range runes {
		put16(&b, GlyphID(r))
		put32(&b, uint32(at)) //nolint:gosec // small
		at += len(encoded[i])
	}
	for _, e := range encoded {
		b.Write(e)
	}
	var l bytes.Buffer
	var layerEncoded [][]byte
	for i := 0; i < len(layers); i++ {
		layerEncoded = append(layerEncoded, encode(layers[i], &layers))
	}
	put32(&l, uint32(len(layers))) //nolint:gosec // a few
	at = 4 + 4*len(layers)
	for _, e := range layerEncoded {
		put32(&l, uint32(at)) //nolint:gosec // small
		at += len(e)
	}
	for _, e := range layerEncoded {
		l.Write(e)
	}
	return b.Bytes(), l.Bytes()
}

func clipList(clips map[rune][4]int16) []byte {
	if len(clips) == 0 {
		return nil
	}
	var b bytes.Buffer
	runes := sortedRunes(clips)
	put8(&b, 1)
	put32(&b, uint32(len(runes))) //nolint:gosec // a few
	at := 5 + 7*len(runes)
	for _, r := range runes {
		put16(&b, GlyphID(r))
		put16(&b, GlyphID(r))
		put24(&b, uint32(at)) //nolint:gosec // small
		at += 9
	}
	for _, r := range runes {
		c := clips[r]
		put8(&b, 1)
		for _, v := range c {
			put16(&b, uint16(v)) //nolint:gosec // a signed FWORD
		}
	}
	return b.Bytes()
}

// PNG is img encoded.
func PNG(img image.Image) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func sbix(strikes map[int]map[rune]Bitmap, numGlyphs int) []byte {
	ppems := make([]int, 0, len(strikes))
	for p := range strikes {
		ppems = append(ppems, p)
	}
	slices.Sort(ppems)
	var b bytes.Buffer
	put16(&b, 1)
	put16(&b, 1)
	put32(&b, uint32(len(ppems))) //nolint:gosec // a few
	var encoded [][]byte
	for _, p := range ppems {
		var s bytes.Buffer
		put16(&s, uint16(p)) //nolint:gosec // a ppem
		put16(&s, 72)
		byGlyph := map[uint16]Bitmap{}
		for r, bm := range strikes[p] {
			byGlyph[GlyphID(r)] = bm
		}
		at := 4 + 4*(numGlyphs+1)
		var data bytes.Buffer
		for g := range numGlyphs + 1 {
			put32(&s, uint32(at+data.Len())) //nolint:gosec // small
			if bm, ok := byGlyph[uint16(g)]; ok && g < numGlyphs {
				put16(&data, uint16(int16(bm.X))) //nolint:gosec // small
				put16(&data, uint16(int16(bm.Y))) //nolint:gosec // small
				data.WriteString("png ")
				data.Write(bm.PNG)
			}
		}
		s.Write(data.Bytes())
		encoded = append(encoded, s.Bytes())
	}
	at := 8 + 4*len(ppems)
	for _, e := range encoded {
		put32(&b, uint32(at)) //nolint:gosec // small
		at += len(e)
	}
	for _, e := range encoded {
		b.Write(e)
	}
	return b.Bytes()
}

// cblc is CBLC and CBDT for bitmaps at ppem: one index subtable of
// format 1 a glyph, each glyph of format 17.
func cblc(bitmaps map[rune]Bitmap, ppem int) ([]byte, []byte) {
	runes := sortedRunes(bitmaps)
	var cbdt bytes.Buffer
	put16(&cbdt, 3)
	put16(&cbdt, 0)
	type entry struct {
		gid      uint16
		at, size uint32
	}
	var entries []entry
	for _, r := range runes {
		bm := bitmaps[r]
		cfg, err := png.DecodeConfig(bytes.NewReader(bm.PNG))
		if err != nil {
			panic(err)
		}
		at := uint32(cbdt.Len())          //nolint:gosec // small
		put8(&cbdt, uint8(cfg.Height))    //nolint:gosec // under 256
		put8(&cbdt, uint8(cfg.Width))     //nolint:gosec // under 256
		put8(&cbdt, uint8(int8(bm.X)))    //nolint:gosec // small
		put8(&cbdt, uint8(int8(bm.Y)))    //nolint:gosec // small
		put8(&cbdt, uint8(ppem))          //nolint:gosec // the advance, one em
		put32(&cbdt, uint32(len(bm.PNG))) //nolint:gosec // small
		cbdt.Write(bm.PNG)
		entries = append(entries, entry{GlyphID(r), at, uint32(cbdt.Len()) - at}) //nolint:gosec // small
	}
	var b bytes.Buffer
	put16(&b, 3)
	put16(&b, 0)
	put32(&b, 1)
	arrayAt := 8 + 48
	put32(&b, uint32(arrayAt))
	arraySize := 8*len(entries) + 16*len(entries)
	put32(&b, uint32(arraySize))    //nolint:gosec // small
	put32(&b, uint32(len(entries))) //nolint:gosec // a few
	put32(&b, 0)
	//nolint:gosec // a ppem's metrics, small
	line := []byte{byte(int8(ppem * 4 / 5)), byte(int8(-ppem / 5)), byte(ppem), 1, 0, 0, 0, 0, 0, 0, 0, 0}
	b.Write(line)
	b.Write(line)
	put16(&b, entries[0].gid)
	put16(&b, entries[len(entries)-1].gid)
	put8(&b, uint8(ppem)) //nolint:gosec // a ppem
	put8(&b, uint8(ppem)) //nolint:gosec // a ppem
	put8(&b, 32)
	put8(&b, 1)
	for i, e := range entries {
		put16(&b, e.gid)
		put16(&b, e.gid)
		put32(&b, uint32(8*len(entries)+16*i)) //nolint:gosec // small
	}
	for _, e := range entries {
		put16(&b, 1)
		put16(&b, 17)
		put32(&b, e.at)
		put32(&b, 0)
		put32(&b, e.size)
	}
	return b.Bytes(), cbdt.Bytes()
}
