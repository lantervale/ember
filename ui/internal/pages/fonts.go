package pages

import (
	"encoding/binary"
	"slices"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// WithTables is the font file ttf with the tables of with in place of its own of their
// tags, those it lacks added, its checksums made again.
func WithTables(ttf []byte, with map[string][]byte) []byte {
	be := binary.BigEndian
	type table struct {
		tag  string
		data []byte
	}
	var tables []table
	for i := range int(be.Uint16(ttf[4:])) {
		rec := ttf[12+16*i:]
		off, size := be.Uint32(rec[8:]), be.Uint32(rec[12:])
		tb := table{string(rec[:4]), ttf[off : off+size]}
		if data, ok := with[tb.tag]; ok {
			tb.data = data
		}
		if tb.tag == "head" {
			tb.data = slices.Clone(tb.data)
			be.PutUint32(tb.data[8:], 0)
		}
		tables = append(tables, tb)
	}
	for tag, data := range with {
		if !slices.ContainsFunc(tables, func(tb table) bool { return tb.tag == tag }) {
			tables = append(tables, table{tag, data})
		}
	}
	slices.SortFunc(tables, func(a, b table) int { return strings.Compare(a.tag, b.tag) })
	n := len(tables)
	out := slices.Clone(ttf[:12])
	be.PutUint16(out[4:], uint16(n)) //nolint:gosec // a font's tables
	search, selector := 1, 0
	for search*2 <= n {
		search *= 2
		selector++
	}
	be.PutUint16(out[6:], uint16(16*search))
	be.PutUint16(out[8:], uint16(selector))
	be.PutUint16(out[10:], uint16(16*(n-search))) //nolint:gosec // a font's tables
	off := 12 + 16*n
	var body []byte
	headAt := -1
	for _, tb := range tables {
		out = append(out, tb.tag...)
		out = be.AppendUint32(out, checksum(tb.data))
		out = be.AppendUint32(out, uint32(off+len(body))) //nolint:gosec // a font's size
		out = be.AppendUint32(out, uint32(len(tb.data)))  //nolint:gosec // a font's size
		if tb.tag == "head" {
			headAt = off + len(body)
		}
		body = append(body, tb.data...)
		for len(body)%4 != 0 {
			body = append(body, 0)
		}
	}
	out = append(out, body...)
	if headAt >= 0 {
		be.PutUint32(out[headAt+8:], 0xb1b0afba-checksum(out))
	}
	return out
}

func checksum(b []byte) uint32 {
	var sum uint32
	for i := 0; i < len(b); i += 4 {
		var word [4]byte
		copy(word[:], b[i:])
		sum += binary.BigEndian.Uint32(word[:])
	}
	return sum
}

// Tall is ttf, a TrueType font, of ascent and descent, in its units, in hhea's and OS/2's
// typographic and Windows metrics alike.
func Tall(ttf []byte, ascent, descent int16) []byte {
	be := binary.BigEndian
	tables := map[string][]byte{}
	for i := range int(be.Uint16(ttf[4:])) {
		rec := ttf[12+16*i:]
		if tag := string(rec[:4]); tag == "hhea" || tag == "OS/2" {
			off, size := be.Uint32(rec[8:]), be.Uint32(rec[12:])
			tables[tag] = slices.Clone(ttf[off : off+size])
		}
	}
	hhea, os2 := tables["hhea"], tables["OS/2"]
	if len(hhea) < 8 || len(os2) < 78 {
		return ttf
	}
	be.PutUint16(hhea[4:], uint16(ascent))   //nolint:gosec // a metric in font units
	be.PutUint16(hhea[6:], uint16(-descent)) //nolint:gosec // a metric in font units
	be.PutUint16(os2[68:], uint16(ascent))   //nolint:gosec // a metric in font units
	be.PutUint16(os2[70:], uint16(-descent)) //nolint:gosec // a metric in font units
	be.PutUint16(os2[74:], uint16(ascent))   //nolint:gosec // a metric in font units
	be.PutUint16(os2[76:], uint16(descent))  //nolint:gosec // a metric in font units
	return WithTables(ttf, tables)
}

// VerticalAdvance is glyph g's advance down a vertical line in a VerticalFont, in its
// units: 1,500 to 2,199, unlike a Go face's ascent and descent, 2,288 of its 2,048 an em.
func VerticalAdvance(g uint16) uint16 { return 1500 + g*37%700 }

// VerticalFont is ttf, a TrueType font, with vertical metrics: each glyph's advance down a
// vertical line is VerticalAdvance's (vhea, vmtx), its top side bearing 100 to 340 units;
// vert replaces the glyph of each key of alternates by its value's, and with origin above
// 0 VORG stands every glyph's baseline origin units below the top of its room.
func VerticalFont(ttf []byte, alternates map[rune]rune, origin uint16) ([]byte, error) {
	f, err := sfnt.Parse(ttf)
	if err != nil {
		return nil, err
	}
	var buf sfnt.Buffer
	subs := map[uint16]uint16{}
	for from, to := range alternates {
		a, aErr := f.GlyphIndex(&buf, from)
		b, bErr := f.GlyphIndex(&buf, to)
		if aErr != nil || bErr != nil || a == 0 || b == 0 {
			continue
		}
		subs[uint16(a)] = uint16(b)
	}
	be := binary.BigEndian
	n := f.NumGlyphs()
	vhea := make([]byte, 36)
	be.PutUint32(vhea, 0x00011000)
	be.PutUint16(vhea[4:], 1024)
	be.PutUint16(vhea[6:], 0xfc00)
	be.PutUint16(vhea[34:], uint16(n)) //nolint:gosec // a font's glyphs
	var vmtx []byte
	for g := range uint16(n) { //nolint:gosec // a font's glyphs
		vmtx = be.AppendUint16(vmtx, VerticalAdvance(g))
		vmtx = be.AppendUint16(vmtx, 100+g%7*40)
	}
	tables := map[string][]byte{"vhea": vhea, "vmtx": vmtx, "GSUB": vertSubstitutions(subs)}
	if origin > 0 {
		vorg := []byte{0, 1, 0, 0}
		vorg = be.AppendUint16(vorg, origin)
		tables["VORG"] = be.AppendUint16(vorg, 0)
	}
	return WithTables(ttf, tables), nil
}

// vertSubstitutions is a GSUB table whose one feature, vert, of the default script,
// replaces each glyph of subs by its value.
func vertSubstitutions(subs map[uint16]uint16) []byte {
	be := binary.BigEndian
	from := make([]uint16, 0, len(subs))
	for g := range subs {
		from = append(from, g)
	}
	slices.Sort(from)
	gsub := []byte{0, 1, 0, 0, 0, 10, 0, 30, 0, 44}
	// the script list: DFLT, its default language system with feature 0
	gsub = append(gsub, 0, 1, 'D', 'F', 'L', 'T', 0, 8, 0, 4, 0, 0, 0, 0, 0xff, 0xff, 0, 1, 0, 0)
	// the feature list: vert, of lookup 0
	gsub = append(gsub, 0, 1, 'v', 'e', 'r', 't', 0, 8, 0, 0, 0, 1, 0, 0)
	// the lookup list: one single substitution, format 2, its coverage after it
	gsub = append(gsub, 0, 1, 0, 4, 0, 1, 0, 0, 0, 1, 0, 8)
	k := uint16(len(from)) //nolint:gosec // a font's glyphs
	gsub = be.AppendUint16(gsub, 2)
	gsub = be.AppendUint16(gsub, 6+2*k)
	gsub = be.AppendUint16(gsub, k)
	for _, g := range from {
		gsub = be.AppendUint16(gsub, subs[g])
	}
	gsub = be.AppendUint16(gsub, 1)
	gsub = be.AppendUint16(gsub, k)
	for _, g := range from {
		gsub = be.AppendUint16(gsub, g)
	}
	return gsub
}
