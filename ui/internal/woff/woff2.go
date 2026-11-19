package woff

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/andybalholm/brotli"
)

// knownTags are the table tags a WOFF2 directory names by their index.
//
//nolint:gochecknoglobals // the format's table
var knownTags = [63]string{
	"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post", "cvt ", "fpgm", "glyf", "loca", "prep",
	"CFF ", "VORG", "EBDT", "EBLC", "gasp", "hdmx", "kern", "LTSH", "PCLT", "VDMX", "vhea", "vmtx", "BASE",
	"GDEF", "GPOS", "GSUB", "EBSC", "JSTF", "MATH", "CBDT", "CBLC", "COLR", "CPAL", "SVG ", "sbix", "acnt",
	"avar", "bdat", "bloc", "bsln", "cvar", "fdsc", "feat", "fmtx", "fvar", "gvar", "hsty", "just", "lcar",
	"mort", "morx", "opbd", "prop", "trak", "Zapf", "Silf", "Glat", "Gloc", "Feat", "Sill",
}

// reader reads a stream from its start; a read past its end fails and
// every read after it does.
type reader struct {
	b   []byte
	bad bool
}

func (r *reader) take(n int) []byte {
	if r.bad || n < 0 || n > len(r.b) {
		r.bad = true
		return nil
	}
	out := r.b[:n:n]
	r.b = r.b[n:]
	return out
}

func (r *reader) u8() uint8 {
	if b := r.take(1); b != nil {
		return b[0]
	}
	return 0
}

func (r *reader) u16() uint16 {
	if b := r.take(2); b != nil {
		return binary.BigEndian.Uint16(b)
	}
	return 0
}

func (r *reader) u32() uint32 {
	if b := r.take(4); b != nil {
		return binary.BigEndian.Uint32(b)
	}
	return 0
}

// base128 reads a UIntBase128: up to five bytes of seven bits, the most
// significant first, with no leading zeros and no overflow.
func (r *reader) base128() uint32 {
	var v uint32
	for i := 0; i < 5; i++ {
		b := r.u8()
		if r.bad || (i == 0 && b == 0x80) || v&0xfe000000 != 0 {
			r.bad = true
			return 0
		}
		v = v<<7 | uint32(b&0x7f)
		if b&0x80 == 0 {
			return v
		}
	}
	r.bad = true
	return 0
}

// short255 reads a 255UInt16.
func (r *reader) short255() int {
	switch code := r.u8(); code {
	case 253:
		return int(r.u16())
	case 255:
		return int(r.u8()) + 253
	case 254:
		return int(r.u8()) + 506
	default:
		return int(code)
	}
}

// entry2 is a WOFF2 directory entry: stream is how many bytes of the
// unpacked data are the table's, and transformed whether those are a
// transformed table to rebuild.
type entry2 struct {
	tag         uint32
	transformed bool
	stream      uint64
	data        []byte
}

// Decode2 is the sfnt file the WOFF2 file data packs: its tables
// unpacked with Brotli, the transformed glyf, loca and hmtx tables
// rebuilt as the format says, and head's checksum set for the file
// written. A collection is refused.
func Decode2(data []byte) ([]byte, error) {
	const header = 48
	if len(data) < header {
		return nil, errShort
	}
	be := binary.BigEndian
	if be.Uint32(data) != sigWOFF2 {
		return nil, errNoFont
	}
	flavor := be.Uint32(data[4:])
	if flavor == sigTTCF {
		return nil, errCollection
	}
	if uint64(be.Uint32(data[8:])) != uint64(len(data)) {
		return nil, errLength
	}
	n := int(be.Uint16(data[12:]))
	total := uint64(be.Uint32(data[16:]))
	packedSize := uint64(be.Uint32(data[20:]))
	if total > MaxSize {
		return nil, errTooLarge
	}
	if n == 0 {
		return nil, errDirectory
	}
	entries, dir, err := directory2(data[header:], n, total)
	if err != nil {
		return nil, err
	}
	if packedSize > uint64(len(dir)) {
		return nil, errLength
	}
	var want uint64
	for _, e := range entries {
		want += e.stream
	}
	unpacked, err := unbrotli(dir[:packedSize], want)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].data = unpacked[:entries[i].stream:entries[i].stream]
		unpacked = unpacked[entries[i].stream:]
	}
	tables, err := rebuild(entries)
	if err != nil {
		return nil, err
	}
	return build(flavor, tables, true)
}

// directory2 reads the n entries of a WOFF2 table directory at the
// start of b, and returns them with what follows it. No table says it
// is longer than total, the font's size, and the streams together stay
// under MaxSize.
func directory2(b []byte, n int, total uint64) ([]entry2, []byte, error) {
	r := &reader{b: b}
	entries := make([]entry2, 0, min(n, len(b)))
	seen := map[uint32]bool{}
	var sum uint64
	for i := 0; i < n; i++ {
		flags := r.u8()
		var e entry2
		if idx := flags & 0x3f; idx == 63 {
			e.tag = r.u32()
		} else {
			e.tag = binary.BigEndian.Uint32([]byte(knownTags[idx]))
		}
		version := flags >> 6
		orig := uint64(r.base128())
		e.stream = orig
		switch e.tag {
		case tagGlyf, tagLoca:
			if version != 0 && version != 3 {
				return nil, nil, errTransform
			}
			e.transformed = version == 0
		case tagHmtx:
			if version > 1 {
				return nil, nil, errTransform
			}
			e.transformed = version == 1
		default:
			if version != 0 {
				return nil, nil, errTransform
			}
		}
		if e.transformed {
			e.stream = uint64(r.base128())
			if e.tag == tagLoca && e.stream != 0 {
				return nil, nil, errTransform
			}
		}
		if r.bad {
			return nil, nil, errShort
		}
		if seen[e.tag] {
			return nil, nil, errDirectory
		}
		seen[e.tag] = true
		if orig > total {
			return nil, nil, errSize
		}
		if sum += e.stream; sum > MaxSize {
			return nil, nil, errTooLarge
		}
		entries = append(entries, e)
	}
	return entries, r.b, nil
}

// unbrotli is the n bytes packed unpacks to, and an error when it holds
// fewer or more.
func unbrotli(packed []byte, n uint64) ([]byte, error) {
	var out bytes.Buffer
	unpacked := io.LimitReader(brotli.NewReader(bytes.NewReader(packed)), int64(n)+1) //nolint:gosec // n ≤ MaxSize
	_, err := io.Copy(&out, unpacked)
	if err != nil || uint64(out.Len()) != n { //nolint:gosec // a buffer's length
		return nil, errCompressed
	}
	return out.Bytes(), nil
}

// rebuild is the font's tables in the directory's order, the
// transformed ones as they were before.
func rebuild(entries []entry2) ([]table, error) {
	find := func(tag uint32) *entry2 {
		for i := range entries {
			if entries[i].tag == tag {
				return &entries[i]
			}
		}
		return nil
	}
	glyf, loca, hmtx := find(tagGlyf), find(tagLoca), find(tagHmtx)
	if (glyf == nil) != (loca == nil) || (glyf != nil && glyf.transformed != loca.transformed) {
		return nil, errTransform
	}
	var xMins []int16
	if glyf != nil && glyf.transformed {
		g, err := untransformGlyf(glyf.data)
		if err != nil {
			return nil, err
		}
		glyf.data, loca.data, xMins = g.glyf, g.loca, g.xMins
	}
	if hmtx != nil && hmtx.transformed {
		hhea := find(tagHhea)
		if hhea == nil || len(hhea.data) < 36 || xMins == nil {
			return nil, errTransform
		}
		var err error
		if hmtx.data, err = untransformHmtx(hmtx.data, int(binary.BigEndian.Uint16(hhea.data[34:])), xMins); err != nil {
			return nil, err
		}
	}
	tables := make([]table, len(entries))
	for i, e := range entries {
		tables[i] = table{e.tag, e.data}
	}
	return tables, nil
}

// untransformHmtx is the hmtx table of a font of len(xMins) glyphs,
// numHMetrics of them with an advance of their own, from its
// transformed form: the left side bearings it leaves out are the
// glyphs' xMin.
func untransformHmtx(data []byte, numHMetrics int, xMins []int16) ([]byte, error) {
	r := &reader{b: data}
	flags := r.u8()
	numGlyphs := len(xMins)
	if r.bad || flags&0xfc != 0 || flags&3 == 0 || numHMetrics < 1 || numHMetrics > numGlyphs {
		return nil, errTransform
	}
	advances := r.take(2 * numHMetrics)
	var lsbs, rest []byte
	if flags&1 == 0 {
		lsbs = r.take(2 * numHMetrics)
	}
	if flags&2 == 0 {
		rest = r.take(2 * (numGlyphs - numHMetrics))
	}
	if r.bad {
		return nil, errTransform
	}
	be := binary.BigEndian
	out := make([]byte, 0, 2*numHMetrics+2*numGlyphs)
	for i := 0; i < numGlyphs; i++ {
		if i < numHMetrics {
			out = append(out, advances[2*i], advances[2*i+1])
		}
		switch {
		case i < numHMetrics && lsbs != nil:
			out = append(out, lsbs[2*i], lsbs[2*i+1])
		case i >= numHMetrics && rest != nil:
			j := 2 * (i - numHMetrics)
			out = append(out, rest[j], rest[j+1])
		default:
			out = be.AppendUint16(out, uint16(xMins[i])) //nolint:gosec // the bits of an int16
		}
	}
	return out, nil
}
