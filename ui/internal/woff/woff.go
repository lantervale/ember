// Package woff unwraps the font files of the web: a WOFF or a WOFF2
// file becomes the sfnt (TrueType or OpenType) file it packs, for the
// code that reads those. The bytes come from the network, so every
// length, offset and count is checked against what is there, nothing
// larger than MaxSize is built, and a file that says anything else is an
// error, never a panic.
package woff

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"sort"
)

// MaxSize is the largest font file a reader builds, and so the most a
// file may say it unpacks to.
const MaxSize = 32 << 20

const (
	sigWOFF  = 0x774f4646
	sigWOFF2 = 0x774f4632
	sigTTCF  = 0x74746366
	sigOTTO  = 0x4f54544f
	sigTrue  = 0x74727565
	sigTTF   = 0x00010000

	tagGlyf = 0x676c7966
	tagLoca = 0x6c6f6361
	tagHmtx = 0x686d7478
	tagHhea = 0x68686561
	tagHead = 0x68656164
)

var (
	errShort      = errors.New("woff: the file ends early")
	errTooLarge   = fmt.Errorf("woff: the font is over %d bytes", MaxSize)
	errLength     = errors.New("woff: the file is not as long as it says")
	errDirectory  = errors.New("woff: the table directory is wrong")
	errSize       = errors.New("woff: the tables are not the size the file says")
	errCompressed = errors.New("woff: a table does not unpack to its length")
	errCollection = errors.New("woff: font collections are not read")
	errTransform  = errors.New("woff: a transformed table is wrong")
	errNoFont     = errors.New("woff: not a font file")
)

// Format is the kind of font file data is by its first bytes: "woff",
// "woff2", "truetype", "opentype" (CFF outlines), "collection", or ""
// for none of them.
func Format(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	switch binary.BigEndian.Uint32(data) {
	case sigWOFF:
		return "woff"
	case sigWOFF2:
		return "woff2"
	case sigTTF, sigTrue:
		return "truetype"
	case sigOTTO:
		return "opentype"
	case sigTTCF:
		return "collection"
	}
	return ""
}

// Sfnt is the sfnt file data holds: data itself when it is one, the
// file a WOFF or a WOFF2 packs, rebuilt, otherwise.
func Sfnt(data []byte) ([]byte, error) {
	switch Format(data) {
	case "woff":
		return Decode(data)
	case "woff2":
		return Decode2(data)
	case "truetype", "opentype":
		return data, nil
	case "collection":
		return nil, errCollection
	}
	return nil, errNoFont
}

// table is one table of the font being rebuilt.
type table struct {
	tag  uint32
	data []byte
}

func pad4(n uint64) uint64 { return (n + 3) &^ 3 }

// checksum is the sum of data's big-endian 32-bit words, the last one
// padded with zeros.
func checksum(data []byte) uint32 {
	var sum uint32
	for len(data) >= 4 {
		sum += binary.BigEndian.Uint32(data)
		data = data[4:]
	}
	if len(data) > 0 {
		var last [4]byte
		copy(last[:], data)
		sum += binary.BigEndian.Uint32(last[:])
	}
	return sum
}

// build writes the sfnt file of tables: their data in the order given,
// each on a 4-byte boundary, under a directory sorted by tag. With
// adjust it sets head's checkSumAdjustment for the file as written.
func build(flavor uint32, tables []table, adjust bool) ([]byte, error) {
	n := len(tables)
	if n == 0 || n > 0xffff {
		return nil, errDirectory
	}
	size := uint64(12 + 16*n)
	for _, t := range tables {
		size += pad4(uint64(len(t.data)))
	}
	if size > MaxSize {
		return nil, errTooLarge
	}
	out := make([]byte, size)
	be := binary.BigEndian
	be.PutUint32(out, flavor)
	entrySelector := bits.Len(uint(n)) - 1
	searchRange := 16 << entrySelector
	be.PutUint16(out[4:], uint16(n))
	be.PutUint16(out[6:], uint16(searchRange))       //nolint:gosec // at most 16 × 0x8000, truncated as the format has it
	be.PutUint16(out[8:], uint16(entrySelector))     //nolint:gosec // under 16
	be.PutUint16(out[10:], uint16(16*n-searchRange)) //nolint:gosec // as above
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return tables[order[a]].tag < tables[order[b]].tag })
	offsets := make([]int, n)
	at := 12 + 16*n
	head := -1
	for i, t := range tables {
		offsets[i] = at
		copy(out[at:], t.data)
		if t.tag == tagHead && len(t.data) >= 12 {
			head = at
			if adjust {
				be.PutUint32(out[at+8:], 0)
			}
		}
		at += int(pad4(uint64(len(t.data)))) //nolint:gosec // under MaxSize
	}
	for slot, i := range order {
		t := tables[i]
		rec := out[12+16*slot:]
		end := offsets[i] + len(t.data)
		sum := checksum(out[offsets[i]:end])
		if offsets[i] == head && !adjust {
			sum -= be.Uint32(out[head+8:])
		}
		be.PutUint32(rec, t.tag)
		be.PutUint32(rec[4:], sum)
		be.PutUint32(rec[8:], uint32(offsets[i]))   //nolint:gosec // under MaxSize
		be.PutUint32(rec[12:], uint32(len(t.data))) //nolint:gosec // under MaxSize
	}
	if adjust && head >= 0 {
		be.PutUint32(out[head+8:], 0xb1b0afba-checksum(out))
	}
	return out, nil
}

// Decode is the sfnt file the WOFF file data packs: each table as it
// was, stored or unpacked with zlib, in the order the file holds them.
func Decode(data []byte) ([]byte, error) {
	const header, record = 44, 20
	if len(data) < header {
		return nil, errShort
	}
	be := binary.BigEndian
	if be.Uint32(data) != sigWOFF {
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
	if total > MaxSize {
		return nil, errTooLarge
	}
	dirEnd := header + record*n
	if n == 0 || dirEnd > len(data) {
		return nil, errDirectory
	}
	type entry struct {
		tag               uint32
		offset, comp, len uint64
	}
	entries := make([]entry, n)
	seen := make(map[uint32]bool, n)
	want := uint64(12 + 16*n)
	for i := range entries {
		rec := data[header+record*i:]
		e := entry{be.Uint32(rec), uint64(be.Uint32(rec[4:])), uint64(be.Uint32(rec[8:])), uint64(be.Uint32(rec[12:]))}
		if seen[e.tag] || e.offset < uint64(dirEnd) || e.offset+e.comp > uint64(len(data)) || e.comp > e.len {
			return nil, errDirectory
		}
		seen[e.tag] = true
		if want += pad4(e.len); want > total {
			return nil, errSize
		}
		entries[i] = e
	}
	if want != total {
		return nil, errSize
	}
	sort.SliceStable(entries, func(a, b int) bool { return entries[a].offset < entries[b].offset })
	tables := make([]table, n)
	end := uint64(dirEnd)
	for i, e := range entries {
		if e.offset < end {
			return nil, errDirectory
		}
		end = e.offset + e.comp
		packed := data[e.offset:end]
		tables[i].tag = e.tag
		if e.comp == e.len {
			tables[i].data = packed
			continue
		}
		unpacked, err := inflate(packed, int(e.len)) //nolint:gosec // under MaxSize
		if err != nil {
			return nil, err
		}
		tables[i].data = unpacked
	}
	return build(flavor, tables, false)
}

// inflate is the n bytes packed unpacks to with zlib, and an error when
// it holds fewer or more.
func inflate(packed []byte, n int) ([]byte, error) {
	z, err := zlib.NewReader(bytes.NewReader(packed))
	if err != nil {
		return nil, errCompressed
	}
	defer z.Close()
	var out bytes.Buffer
	if _, err = io.Copy(&out, io.LimitReader(z, int64(n)+1)); err != nil || out.Len() != n {
		return nil, errCompressed
	}
	return out.Bytes(), nil
}
