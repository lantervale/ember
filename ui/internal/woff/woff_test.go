package woff //nolint:testpackage // tests unexported internals

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

func fixture(t testing.TB, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// tablesOf are an sfnt file's tables by tag.
func tablesOf(t testing.TB, ttf []byte) map[string][]byte {
	t.Helper()
	be := binary.BigEndian
	out := map[string][]byte{}
	for i := 0; i < int(be.Uint16(ttf[4:])); i++ {
		rec := ttf[12+16*i:]
		off, n := be.Uint32(rec[8:]), be.Uint32(rec[12:])
		data := ttf[off : off+n]
		if sum := checksum(data); string(rec[:4]) != "head" && sum != be.Uint32(rec[4:]) {
			t.Errorf("table %s: checksum %08x, the directory says %08x", rec[:4], sum, be.Uint32(rec[4:]))
		}
		out[string(rec[:4])] = data
	}
	return out
}

func TestAWOFFFileIsRebuiltTableByTable(t *testing.T) {
	for _, name := range []string{"go-regular-latin", "go-icons"} {
		got, err := Decode(fixture(t, name+".woff"))
		if err != nil {
			t.Fatal(err)
		}
		want := fixture(t, name+".ttf")
		if len(got) != len(want) || !bytes.Equal(got[:12], want[:12]) {
			t.Fatalf("%s: the rebuilt file is %d bytes, the font packed %d, or their headers differ",
				name, len(got), len(want))
		}
		if checksum(got) != 0xb1b0afba {
			t.Errorf("%s: the file sums to %08x: head's checksum no longer holds", name, checksum(got))
		}
		original, rebuilt := tablesOf(t, want), tablesOf(t, got)
		if len(rebuilt) != len(original) {
			t.Errorf("%s: %d tables rebuilt of %d", name, len(rebuilt), len(original))
		}
		for tag, data := range original {
			table := rebuilt[tag]
			if tag == "head" {
				// The tables stand in the WOFF file's order, so the
				// directory's offsets, and the checksum over them, are new.
				table = withAdjustment(table, data)
			}
			if !bytes.Equal(table, data) {
				t.Errorf("%s: table %s differs from the font packed", name, tag)
			}
		}
	}
}

// withAdjustment is the head table with from's checkSumAdjustment.
func withAdjustment(head, from []byte) []byte {
	return append(append(append([]byte{}, head[:8]...), from[8:12]...), head[12:]...)
}

func TestAWOFF2FileIsRebuiltTableByTable(t *testing.T) {
	for _, packed := range []string{"go-regular-latin.woff2", "go-icons.woff2", "go-icons.hmtx.woff2"} {
		name, _, _ := strings.Cut(packed, ".")
		t.Run(packed, func(t *testing.T) {
			got, err := Decode2(fixture(t, packed))
			if err != nil {
				t.Fatal(err)
			}
			if checksum(got) != 0xb1b0afba {
				t.Errorf("the file sums to %08x, not to the checksum head adjusts it to", checksum(got))
			}
			original := tablesOf(t, fixture(t, name+".ttf"))
			rebuilt := tablesOf(t, got)
			if len(rebuilt) != len(original) {
				t.Errorf("%d tables rebuilt of %d", len(rebuilt), len(original))
			}
			for tag, want := range original {
				switch tag {
				case "head":
					// woff2_compress sets bit 11 of flags, and the checksum is the new file's.
					if !bytes.Equal(rebuilt[tag][12:16], want[12:16]) || !bytes.Equal(rebuilt[tag][18:], want[18:]) {
						t.Errorf("head differs past its checksum and flags")
					}
				case "glyf", "loca":
					// Compared by outlines below: the encoder packs flags its own way.
				default:
					if !bytes.Equal(rebuilt[tag], want) {
						t.Errorf("table %s differs from the font packed", tag)
					}
				}
			}
			sameGlyphs(t, got, fixture(t, name+".ttf"))
		})
	}
}

// TestAWOFF2FileIsRebuiltAsTheReferenceDecoderDoes compares every
// table with what woff2_decompress writes for the same file.
func TestAWOFF2FileIsRebuiltAsTheReferenceDecoderDoes(t *testing.T) {
	for _, name := range []string{"go-regular-latin", "go-icons"} {
		got, err := Decode2(fixture(t, name+".woff2"))
		if err != nil {
			t.Fatal(err)
		}
		reference := tablesOf(t, fixture(t, name+".roundtrip.ttf"))
		rebuilt := tablesOf(t, got)
		for tag, want := range reference {
			if tag == "head" {
				want = withAdjustment(want, rebuilt[tag])
			}
			if !bytes.Equal(rebuilt[tag], want) {
				t.Errorf("%s: table %s is %d bytes, differing from the reference decoder's %d",
					name, tag, len(rebuilt[tag]), len(want))
			}
		}
	}
}

// sameGlyphs checks two font files draw every glyph alike, with the
// same advance.
func sameGlyphs(t *testing.T, got, want []byte) {
	t.Helper()
	g, err := sfnt.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	w, err := sfnt.Parse(want)
	if err != nil {
		t.Fatal(err)
	}
	if g.NumGlyphs() != w.NumGlyphs() {
		t.Fatalf("%d glyphs, want %d", g.NumGlyphs(), w.NumGlyphs())
	}
	var gb, wb sfnt.Buffer
	ppem := fixed.I(int(w.UnitsPerEm()))
	for i := 0; i < w.NumGlyphs(); i++ {
		idx := sfnt.GlyphIndex(i)
		gs, gerr := g.LoadGlyph(&gb, idx, ppem, nil)
		ws, werr := w.LoadGlyph(&wb, idx, ppem, nil)
		if (gerr == nil) != (werr == nil) || len(gs) != len(ws) {
			t.Fatalf("glyph %d: %d segments (%v), want %d (%v)", i, len(gs), gerr, len(ws), werr)
		}
		for j := range ws {
			if gs[j] != ws[j] {
				t.Fatalf("glyph %d: segment %d is %v, want %v", i, j, gs[j], ws[j])
			}
		}
		ga, _ := g.GlyphAdvance(&gb, idx, ppem, font.HintingNone)
		wa, _ := w.GlyphAdvance(&wb, idx, ppem, font.HintingNone)
		if ga != wa {
			t.Fatalf("glyph %d: advance %v, want %v", i, ga, wa)
		}
	}
}

// pack2 writes tables as a WOFF2 file with no table transformed, for
// the paths woff2_compress never takes.
func pack2(ttf []byte, edit func(header, directory []byte)) []byte {
	be := binary.BigEndian
	n := int(be.Uint16(ttf[4:]))
	var dir, stream []byte
	for i := 0; i < n; i++ {
		rec := ttf[12+16*i:]
		tag := string(rec[:4])
		off, size := be.Uint32(rec[8:]), be.Uint32(rec[12:])
		flags := byte(63)
		for k, known := range knownTags {
			if known == tag {
				flags = byte(k)
			}
		}
		if tag == "glyf" || tag == "loca" {
			flags |= 3 << 6
		}
		dir = append(dir, flags)
		if flags&0x3f == 63 {
			dir = append(dir, tag...)
		}
		dir = appendBase128(dir, size)
		stream = append(stream, ttf[off:off+size]...)
	}
	var packed bytes.Buffer
	w := brotli.NewWriter(&packed)
	_, _ = w.Write(stream)
	_ = w.Close()
	header := make([]byte, 48)
	be.PutUint32(header, sigWOFF2)
	copy(header[4:], ttf[:4])
	be.PutUint16(header[12:], uint16(n))
	be.PutUint32(header[16:], uint32(len(ttf)))
	be.PutUint32(header[20:], uint32(packed.Len()))
	be.PutUint32(header[8:], uint32(48+len(dir)+packed.Len()))
	if edit != nil {
		edit(header, dir)
	}
	return slices.Concat(header, dir, packed.Bytes())
}

func appendBase128(b []byte, v uint32) []byte {
	started := false
	for shift := 28; shift >= 0; shift -= 7 {
		d := byte(v>>shift) & 0x7f
		if d == 0 && !started && shift > 0 {
			continue
		}
		started = true
		if shift > 0 {
			d |= 0x80
		}
		b = append(b, d)
	}
	return b
}

func TestAWOFF2FileWithNoTransformKeepsEveryTable(t *testing.T) {
	ttf := fixture(t, "go-regular-latin.ttf")
	got, err := Decode2(pack2(ttf, nil))
	if err != nil {
		t.Fatal(err)
	}
	original := tablesOf(t, ttf)
	for tag, data := range tablesOf(t, got) {
		if tag == "head" {
			data = withAdjustment(data, original[tag])
		}
		if !bytes.Equal(data, original[tag]) {
			t.Errorf("table %s differs", tag)
		}
	}
}

func TestTheFixturesUseTheTransforms(t *testing.T) {
	data := fixture(t, "go-icons.hmtx.woff2")
	entries, _, err := directory2(data[48:], int(binary.BigEndian.Uint16(data[12:])), MaxSize)
	if err != nil {
		t.Fatal(err)
	}
	transformed := map[uint32]bool{}
	for _, e := range entries {
		transformed[e.tag] = e.transformed
	}
	for _, tag := range []uint32{tagGlyf, tagLoca, tagHmtx} {
		if !transformed[tag] {
			t.Errorf("table %08x of the fixture is not transformed: its reader goes untested", tag)
		}
	}
}

func TestSfntUnwrapsWhatItIsGiven(t *testing.T) {
	ttf := fixture(t, "go-regular-latin.ttf")
	for name, want := range map[string]string{
		"go-regular-latin.ttf": "truetype", "go-regular-latin.woff": "woff", "go-regular-latin.woff2": "woff2",
	} {
		data := fixture(t, name)
		if got := Format(data); got != want {
			t.Errorf("Format(%s) = %q, want %q", name, got, want)
		}
		out, err := Sfnt(data)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if _, err = sfnt.Parse(out); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if out, _ := Sfnt(ttf); &out[0] != &ttf[0] {
		t.Error("an sfnt file is copied, not handed back")
	}
	if _, err := Sfnt([]byte("<html>not a font</html>")); !errors.Is(err, errNoFont) {
		t.Errorf("a page of HTML: %v, want %v", err, errNoFont)
	}
	if _, err := Sfnt([]byte("ttcf\x00\x01\x00\x00")); !errors.Is(err, errCollection) {
		t.Errorf("a collection: %v, want %v", err, errCollection)
	}
}

func TestEveryTruncatedFileIsAnError(t *testing.T) {
	for _, name := range []string{
		"go-regular-latin.woff", "go-icons.woff", "go-regular-latin.woff2", "go-icons.woff2", "go-icons.hmtx.woff2",
	} {
		data := fixture(t, name)
		for n := 0; n < len(data); n++ {
			if _, err := Sfnt(data[:n]); err == nil {
				t.Fatalf("%s cut to %d bytes of %d is read", name, n, len(data))
			}
		}
	}
}

func TestAFileSayingItIsLargerThanTheCapIsRefused(t *testing.T) {
	be := binary.BigEndian
	woff := append([]byte{}, fixture(t, "go-regular-latin.woff")...)
	be.PutUint32(woff[16:], MaxSize+4)
	if _, err := Decode(woff); !errors.Is(err, errTooLarge) {
		t.Errorf("WOFF: %v, want %v", err, errTooLarge)
	}
	woff2 := append([]byte{}, fixture(t, "go-regular-latin.woff2")...)
	be.PutUint32(woff2[16:], MaxSize+4)
	if _, err := Decode2(woff2); !errors.Is(err, errTooLarge) {
		t.Errorf("WOFF2: %v, want %v", err, errTooLarge)
	}
}

func TestATableLongerThanTheFontSaysIsRefused(t *testing.T) {
	ttf := fixture(t, "go-regular-latin.ttf")
	small := pack2(ttf, func(header, _ []byte) { binary.BigEndian.PutUint32(header[16:], 64) })
	if _, err := Decode2(small); !errors.Is(err, errSize) {
		t.Errorf("WOFF2 whose tables are longer than its totalSfntSize: %v, want %v", err, errSize)
	}
	woff := append([]byte{}, fixture(t, "go-regular-latin.woff")...)
	binary.BigEndian.PutUint32(woff[16:], binary.BigEndian.Uint32(woff[16:])-4)
	if _, err := Decode(woff); !errors.Is(err, errSize) {
		t.Errorf("WOFF whose tables are longer than its totalSfntSize: %v, want %v", err, errSize)
	}
}

// TestAStreamUnpackingToMoreThanItsTablesIsRefused is the Brotli bomb:
// a megabyte of zeros behind a directory that says a few bytes.
func TestAStreamUnpackingToMoreThanItsTablesIsRefused(t *testing.T) {
	var packed bytes.Buffer
	w := brotli.NewWriter(&packed)
	_, _ = w.Write(make([]byte, 1<<20))
	_ = w.Close()
	be := binary.BigEndian
	header := make([]byte, 48)
	be.PutUint32(header, sigWOFF2)
	be.PutUint32(header[4:], sigTTF)
	be.PutUint16(header[12:], 1)
	be.PutUint32(header[16:], 1024)
	dir := appendBase128([]byte{0}, 16)
	be.PutUint32(header[20:], uint32(packed.Len()))
	be.PutUint32(header[8:], uint32(48+len(dir)+packed.Len()))
	file := slices.Concat(header, dir, packed.Bytes())
	if _, err := Decode2(file); !errors.Is(err, errCompressed) {
		t.Errorf("got %v, want %v", err, errCompressed)
	}
}

func TestBase128RefusesLeadingZerosAndOverflow(t *testing.T) {
	for _, tc := range []struct {
		in   []byte
		want uint32
		bad  bool
	}{
		{[]byte{0x3f}, 63, false},
		{[]byte{0x81, 0x00}, 128, false},
		{[]byte{0x8f, 0xff, 0xff, 0xff, 0x7f}, 0xffffffff, false},
		{[]byte{0x80, 0x01}, 0, true},
		{[]byte{0x9f, 0xff, 0xff, 0xff, 0x7f}, 0, true},
		{[]byte{0x81, 0x81, 0x81, 0x81, 0x81, 0x01}, 0, true},
		{[]byte{0x81}, 0, true},
	} {
		r := &reader{b: tc.in}
		if got := r.base128(); got != tc.want || r.bad != tc.bad {
			t.Errorf("base128(% x) = %d, bad %v; want %d, bad %v", tc.in, got, r.bad, tc.want, tc.bad)
		}
	}
}

func TestShort255(t *testing.T) {
	for _, tc := range []struct {
		in   []byte
		want int
	}{
		{[]byte{252}, 252}, {[]byte{255, 0}, 253}, {[]byte{254, 0}, 506}, {[]byte{254, 255}, 761},
		{[]byte{253, 0x02, 0xfa}, 762}, {[]byte{253, 0xff, 0xff}, 65535},
	} {
		r := &reader{b: tc.in}
		if got := r.short255(); got != tc.want || r.bad {
			t.Errorf("short255(% x) = %d, bad %v; want %d", tc.in, got, r.bad, tc.want)
		}
	}
}

// read is what the fuzzers ask of a reader: an error or a font no
// larger than the cap, which the font code then reads without a panic.
func read(t *testing.T, decode func([]byte) ([]byte, error), data []byte) {
	out, err := decode(data)
	if err != nil {
		return
	}
	if len(out) > MaxSize {
		t.Fatalf("a font of %d bytes, over the cap", len(out))
	}
	f, err := sfnt.Parse(out)
	if err != nil {
		return
	}
	var buf sfnt.Buffer
	ppem := fixed.I(int(f.UnitsPerEm()))
	for _, r := range "Aag0" {
		if g, gerr := f.GlyphIndex(&buf, r); gerr == nil {
			_, _ = f.LoadGlyph(&buf, g, ppem, nil)
			_, _ = f.GlyphAdvance(&buf, g, ppem, font.HintingNone)
		}
	}
}

func FuzzDecode(f *testing.F) {
	f.Add(fixture(f, "go-icons.woff"))
	f.Add(fixture(f, "go-regular-latin.woff"))
	f.Fuzz(func(t *testing.T, data []byte) { read(t, Decode, data) })
}

func FuzzDecode2(f *testing.F) {
	f.Add(fixture(f, "go-regular-latin.woff2"))
	f.Add(fixture(f, "go-icons.woff2"))
	f.Add(fixture(f, "go-icons.hmtx.woff2"))
	f.Add(pack2(fixture(f, "go-icons.ttf"), nil))
	f.Fuzz(func(t *testing.T, data []byte) { read(t, Decode2, data) })
}

// FuzzUntransformGlyf feeds the glyf reader what Brotli would hand it,
// which a fuzzer of whole files seldom gets past the compression to.
func FuzzUntransformGlyf(f *testing.F) {
	for _, name := range []string{"go-regular-latin.woff2", "go-icons.woff2", "go-icons.hmtx.woff2"} {
		data := fixture(f, name)
		n := int(binary.BigEndian.Uint16(data[12:]))
		entries, rest, err := directory2(data[48:], n, MaxSize)
		if err != nil {
			f.Fatal(err)
		}
		var want uint64
		for _, e := range entries {
			want += e.stream
		}
		stream, err := unbrotli(rest[:binary.BigEndian.Uint32(data[20:])], want)
		if err != nil {
			f.Fatal(err)
		}
		for _, e := range entries {
			if e.tag == tagGlyf {
				f.Add(stream[:e.stream], uint16(4))
			}
			stream = stream[e.stream:]
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, numHMetrics uint16) {
		g, err := untransformGlyf(data)
		if err != nil {
			return
		}
		if len(g.glyf) > MaxSize || len(g.loca) > 4*(1<<16+1) {
			t.Fatalf("glyf of %d bytes, loca of %d", len(g.glyf), len(g.loca))
		}
		_, _ = untransformHmtx(data, int(numHMetrics), g.xMins)
	})
}
