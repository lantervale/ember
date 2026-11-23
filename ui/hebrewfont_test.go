package ui_test

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
)

// hebrewFont is Go Regular with the Hebrew letters drawn by its Latin ones,
// א as a, ב as b and on: a font that ember and Chrome both shape alike, for
// right-to-left text whose widths they agree on.
func hebrewFont(t testing.TB) []byte {
	t.Helper()
	f, err := sfnt.Parse(goregular.TTF)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	glyphs := map[rune]uint16{}
	for r := rune(0x20); r < 0xfffe; r++ {
		if g, lookupErr := f.GlyphIndex(&buf, r); lookupErr == nil && g != 0 {
			glyphs[r] = uint16(g)
		}
	}
	for i := range rune(0x05ea - 0x05d0 + 1) {
		latin := 'a' + i
		if i == 26 {
			latin = 'A'
		}
		glyphs[0x05d0+i] = glyphs[latin]
	}
	return withCmap(t, goregular.TTF, cmap4(glyphs))
}

// cmap4 is a cmap table of one Windows Unicode BMP subtable, format 4,
// mapping each rune of glyphs.
func cmap4(glyphs map[rune]uint16) []byte {
	runes := make([]rune, 0, len(glyphs))
	for r := range glyphs {
		runes = append(runes, r)
	}
	slices.Sort(runes)
	type segment struct{ start, end rune }
	var segs []segment
	for _, r := range runes {
		if n := len(segs); n > 0 && segs[n-1].end == r-1 && glyphs[r]-glyphs[r-1] == 1 {
			segs[n-1].end = r
			continue
		}
		segs = append(segs, segment{r, r})
	}
	segs = append(segs, segment{0xffff, 0xffff})
	n := len(segs)
	sub := binary.BigEndian.AppendUint16(nil, 4)
	length := 16 + 8*n
	sub = binary.BigEndian.AppendUint16(sub, uint16(length))
	sub = binary.BigEndian.AppendUint16(sub, 0)
	sub = binary.BigEndian.AppendUint16(sub, uint16(2*n))
	search, selector := 2, 0
	for search*2 <= 2*n {
		search *= 2
		selector++
	}
	sub = binary.BigEndian.AppendUint16(sub, uint16(search))
	sub = binary.BigEndian.AppendUint16(sub, uint16(selector))
	sub = binary.BigEndian.AppendUint16(sub, uint16(2*n-search))
	for _, s := range segs {
		sub = binary.BigEndian.AppendUint16(sub, uint16(s.end))
	}
	sub = binary.BigEndian.AppendUint16(sub, 0)
	for _, s := range segs {
		sub = binary.BigEndian.AppendUint16(sub, uint16(s.start))
	}
	for _, s := range segs {
		delta := uint16(1)
		if s.start != 0xffff {
			delta = glyphs[s.start] - uint16(s.start)
		}
		sub = binary.BigEndian.AppendUint16(sub, delta)
	}
	for range segs {
		sub = binary.BigEndian.AppendUint16(sub, 0)
	}
	table := []byte{0, 0, 0, 1, 0, 3, 0, 1, 0, 0, 0, 12}
	return append(table, sub...)
}

// withCmap is the font file ttf with its cmap table replaced by cmap.
func withCmap(t testing.TB, ttf, cmap []byte) []byte {
	t.Helper()
	return withTables(t, ttf, map[string][]byte{"cmap": cmap})
}

// withTables is the font file ttf with the tables of with in place of its own of their
// tags, those it lacks added in their tags' order.
func withTables(t testing.TB, ttf []byte, with map[string][]byte) []byte {
	t.Helper()
	be := binary.BigEndian
	n := int(be.Uint16(ttf[4:]))
	type table struct {
		tag  string
		data []byte
	}
	tables := make([]table, n)
	for i := range n {
		rec := ttf[12+16*i:]
		off, size := be.Uint32(rec[8:]), be.Uint32(rec[12:])
		tables[i] = table{string(rec[:4]), ttf[off : off+size]}
		if data, ok := with[tables[i].tag]; ok {
			tables[i].data = data
		}
		if tables[i].tag == "head" {
			head := slices.Clone(tables[i].data)
			be.PutUint32(head[8:], 0)
			tables[i].data = head
		}
	}
	for tag, data := range with {
		if !slices.ContainsFunc(tables, func(tb table) bool { return tb.tag == tag }) {
			tables = append(tables, table{tag, data})
		}
	}
	slices.SortFunc(tables, func(a, b table) int { return strings.Compare(a.tag, b.tag) })
	n = len(tables)
	out := slices.Clone(ttf[:12])
	be.PutUint16(out[4:], uint16(n))
	search, selector := 1, 0
	for search*2 <= n {
		search *= 2
		selector++
	}
	be.PutUint16(out[6:], uint16(16*search))
	be.PutUint16(out[8:], uint16(selector))
	be.PutUint16(out[10:], uint16(16*(n-search)))
	off := 12 + 16*n
	var body []byte
	var headAt int
	for _, tb := range tables {
		out = append(out, tb.tag...)
		out = be.AppendUint32(out, checksum(tb.data))
		out = be.AppendUint32(out, uint32(off+len(body)))
		out = be.AppendUint32(out, uint32(len(tb.data)))
		if tb.tag == "head" {
			headAt = off + len(body)
		}
		body = append(body, tb.data...)
		for len(body)%4 != 0 {
			body = append(body, 0)
		}
	}
	out = append(out, body...)
	be.PutUint32(out[headAt+8:], 0xb1b0afba-checksum(out))
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

func TestTheHebrewFontDrawsHebrewByLatin(t *testing.T) {
	f, err := sfnt.Parse(hebrewFont(t))
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	for _, pair := range [][2]rune{{'א', 'a'}, {'ת', 'A'}, {'x', 'x'}, {'é', 'é'}} {
		got, lookupErr := f.GlyphIndex(&buf, pair[0])
		if lookupErr != nil {
			t.Fatal(lookupErr)
		}
		want, _ := f.GlyphIndex(&buf, pair[1])
		if got == 0 || got != want {
			t.Errorf("%q: glyph %d, want %q's %d", pair[0], got, pair[1], want)
		}
	}
}
