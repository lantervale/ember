package colourfont

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"regexp"
	"sync/atomic"
)

// SVGReader draws an SVG document into a picture width by height
// pixels, its currentColor fg.
type SVGReader func(doc []byte, width, height int, fg color.NRGBA) (*image.RGBA, error)

// svgReader draws the SVG table's glyphs. Without one they are left to
// the font's other colour tables or its outlines, so a program links no
// SVG reader it did not import.
var svgReader atomic.Pointer[SVGReader] //nolint:gochecknoglobals // set by an import, as ui's vector formats

// RegisterSVG has read draw the SVG table's glyphs, as importing ui/svg
// does, and returns the reader it replaces; nil leaves them undrawn.
func RegisterSVG(read SVGReader) SVGReader {
	var next *SVGReader
	if read != nil {
		next = &read
	}
	if prev := svgReader.Swap(next); prev != nil {
		return *prev
	}
	return nil
}

// maxSVGDocument is the longest SVG document read, unzipped.
const maxSVGDocument = 1 << 24

// svgDoc is the document drawing glyph gid, nil for none or while no
// reader is registered.
func (f *Font) svgDoc(gid uint16) []byte {
	if f.svg == nil || svgReader.Load() == nil {
		return nil
	}
	t := f.svg
	list := int(t.u32(2))
	n := int(t.u16(list))
	lo, hi := 0, n
	for lo < hi {
		mid := (lo + hi) / 2
		rec := list + 2 + 12*mid
		switch {
		case gid < t.u16(rec):
			hi = mid
		case gid > t.u16(rec+2):
			lo = mid + 1
		default:
			at, size := list+int(t.u32(rec+4)), int(t.u32(rec+8))
			return t.bytes(at, size)
		}
	}
	return nil
}

var (
	prolog  = regexp.MustCompile(`(?s)^\s*(<\?xml.*?\?>\s*)?(<!DOCTYPE[^>]*>\s*)?`)
	rootTag = regexp.MustCompile(`(?s)<svg\b`)
)

// drawSVG draws glyph gid's element, glyphN, of its document, whose
// user space is the font's units with y down, through the registered
// reader, over the font's bounding box.
func (f *Font) drawSVG(gid uint16, req Request) (*image.RGBA, image.Point, bool) {
	read := svgReader.Load()
	doc := f.svgDoc(gid)
	if read == nil || doc == nil {
		return nil, image.Point{}, false
	}
	if len(doc) > 2 && doc[0] == 0x1f && doc[1] == 0x8b {
		z, err := gzip.NewReader(bytes.NewReader(doc))
		if err != nil {
			return nil, image.Point{}, false
		}
		doc, err = io.ReadAll(io.LimitReader(z, maxSVGDocument))
		if err != nil {
			return nil, image.Point{}, false
		}
	}
	doc = prolog.ReplaceAll(doc, nil)
	if !rootTag.Match(doc) {
		return nil, image.Point{}, false
	}
	s := float64(req.PPEM / f.upem)
	px0, py0 := math.Floor(float64(f.xMin)*s), math.Floor(-float64(f.yMax)*s)
	pw, ph := int(math.Ceil(float64(f.xMax)*s)-px0), int(math.Ceil(-float64(f.yMin)*s)-py0)
	if pw <= 0 || ph <= 0 || pw > maxSide || ph > maxSide {
		return nil, image.Point{}, false
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" `+
		`width="%d" height="%d" viewBox="%g %g %g %g" overflow="visible"><defs>`,
		pw, ph, px0/s, py0/s, float64(pw)/s, float64(ph)/s)
	b.Write(doc)
	fmt.Fprintf(&b, `</defs><use href="#glyph%d"/></svg>`, gid)
	img, err := (*read)(b.Bytes(), pw, ph, req.Foreground)
	if err != nil || img == nil {
		return nil, image.Point{}, false
	}
	return img, image.Pt(int(px0), int(py0)), true
}
