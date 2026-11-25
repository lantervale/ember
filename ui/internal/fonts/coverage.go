package fonts

import (
	"encoding/binary"
	"os"
	"slices"
	"sort"
	"sync"
)

// coverage is the characters an installed font's cmap maps to a glyph,
// in order, and whether ember can draw its glyphs: outlines, glyf or
// CFF, and no colour bitmaps standing for them.
type coverage struct {
	ranges   []Range
	drawable bool
}

// has reports whether the font maps r to a glyph.
func (c *coverage) has(r rune) bool {
	i := sort.Search(len(c.ranges), func(i int) bool { return c.ranges[i].Hi >= r })
	return i < len(c.ranges) && c.ranges[i].Lo <= r
}

//nolint:gochecknoglobals // the machine's fonts' coverage, read once for every Document and the scan
var (
	coveredMu sync.Mutex
	covered   = map[*installedFace]*coverage{}
)

// coverage is what f's cmap covers, read the first time it is asked
// for, the frame's or the scan's, and kept: none for a file that can't
// be read.
func (f *installedFace) coverage() *coverage {
	coveredMu.Lock()
	c, ok := covered[f]
	coveredMu.Unlock()
	if ok {
		return c
	}
	c = f.readCoverage()
	coveredMu.Lock()
	covered[f] = c
	coveredMu.Unlock()
	return c
}

const (
	tagCmap = 0x636d6170
	tagGlyf = 0x676c7966
	tagCFF  = 0x43464620
	tagCFF2 = 0x43464632
	tagSbix = 0x73626978
	tagCBDT = 0x43424454
)

// maxCmap is the longest cmap read: longer is no real font's.
const maxCmap = 16 << 20

func (f *installedFace) readCoverage() *coverage {
	out := &coverage{}
	file, err := os.Open(f.path)
	if err != nil {
		return out
	}
	defer file.Close()
	_, tables, err := tableDirectory(file, f.offset)
	if err != nil {
		return out
	}
	var cmap tableRecord
	outlines, bitmaps := false, false
	for _, t := range tables {
		switch t.tag {
		case tagCmap:
			cmap = t
		case tagGlyf, tagCFF, tagCFF2:
			outlines = true
		case tagSbix, tagCBDT:
			bitmaps = true
		}
	}
	if cmap.length == 0 || cmap.length > maxCmap {
		return out
	}
	buf := make([]byte, cmap.length)
	if _, err = file.ReadAt(buf, int64(cmap.offset)); err != nil {
		return out
	}
	out.ranges = cmapRanges(buf)
	out.drawable = outlines && !bitmaps && len(out.ranges) > 0
	return out
}

// cmapRanges are the characters a cmap table maps to a glyph, from its
// best Unicode subtable: a full one (format 12) over a BMP one (format
// 4); a symbol font's maps none.
func cmapRanges(cmap []byte) []Range {
	if len(cmap) < 4 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(cmap[2:]))
	best, rank := -1, 0
	for i := range n {
		rec := 4 + 8*i
		if rec+8 > len(cmap) {
			break
		}
		platform, encoding := binary.BigEndian.Uint16(cmap[rec:]), binary.BigEndian.Uint16(cmap[rec+2:])
		off := int(binary.BigEndian.Uint32(cmap[rec+4:]))
		if off+2 > len(cmap) {
			continue
		}
		format := binary.BigEndian.Uint16(cmap[off:])
		const unicode, windows = 0, 3
		r := 0
		switch {
		case format == 12 && (platform == unicode || platform == windows && encoding == 10):
			r = 2
		case format == 4 && (platform == unicode || platform == windows && encoding == 1):
			r = 1
		}
		if r > rank {
			best, rank = off, r
		}
	}
	switch rank {
	case 2:
		return format12(cmap[best:])
	case 1:
		return format4(cmap[best:])
	}
	return nil
}

func format12(t []byte) []Range {
	if len(t) < 16 {
		return nil
	}
	n := int(binary.BigEndian.Uint32(t[12:]))
	var out []Range
	for i := range n {
		g := t[16+12*i:]
		if len(g) < 12 {
			break
		}
		//nolint:gosec // a cmap's characters, which a rune holds
		lo, hi := rune(binary.BigEndian.Uint32(g)), rune(binary.BigEndian.Uint32(g[4:]))
		if binary.BigEndian.Uint32(g[8:]) == 0 {
			lo++
		}
		if lo <= hi {
			out = append(out, Range{Lo: lo, Hi: hi})
		}
	}
	return merged(out)
}

func format4(t []byte) []Range {
	if len(t) < 14 {
		return nil
	}
	segs := int(binary.BigEndian.Uint16(t[6:])) / 2
	if 16+8*segs > len(t) {
		return nil
	}
	ends, starts := t[14:], t[16+2*segs:]
	deltas, offsets := t[16+4*segs:], t[16+6*segs:]
	var out []Range
	add := func(r rune) {
		if k := len(out) - 1; k >= 0 && out[k].Hi == r-1 {
			out[k].Hi = r
			return
		}
		out = append(out, Range{Lo: r, Hi: r})
	}
	for i := range segs {
		end, start := int(binary.BigEndian.Uint16(ends[2*i:])), int(binary.BigEndian.Uint16(starts[2*i:]))
		delta, ro := int(binary.BigEndian.Uint16(deltas[2*i:])), int(binary.BigEndian.Uint16(offsets[2*i:]))
		for c := start; c <= end && c < 0xFFFF; c++ {
			var g int
			if ro == 0 {
				g = (c + delta) & 0xFFFF
			} else {
				at := 16 + 6*segs + 2*i + ro + 2*(c-start)
				if at+2 > len(t) {
					continue
				}
				if g = int(binary.BigEndian.Uint16(t[at:])); g != 0 {
					g = (g + delta) & 0xFFFF
				}
			}
			if g != 0 {
				add(rune(c)) //nolint:gosec // a character of the BMP
			}
		}
	}
	return merged(out)
}

// merged is ranges sorted, those that touch made one.
func merged(ranges []Range) []Range {
	slices.SortFunc(ranges, func(a, b Range) int { return int(a.Lo - b.Lo) })
	var out []Range
	for _, r := range ranges {
		if k := len(out) - 1; k >= 0 && r.Lo <= out[k].Hi+1 {
			out[k].Hi = max(out[k].Hi, r.Hi)
			continue
		}
		out = append(out, r)
	}
	return out
}
