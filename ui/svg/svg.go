// Package svg lets a ui Document show SVG pictures: importing it
// registers ember's svg reader as a vector format of ui, so an img's
// src and a url() background naming an SVG file draw it, at the size it
// takes on the screen.
//
//	import _ "github.com/lantervale/ember/ui/svg"
//
// A file is one by its name ending in .svg or by its bytes starting, past
// an XML prolog, a doctype and comments, with an svg element. What a file
// uses that the reader doesn't draw is a warning of the Document.
//
// It also draws the SVG glyphs of colour fonts in twod's texts, which
// without it are left to the font's other colour tables or its outlines;
// ui leaves them to their outlines either way, as Chrome.
package svg

import (
	"bytes"
	"image"
	"math"
	"strings"

	"github.com/lantervale/ember/internal/colourfont"
	"github.com/lantervale/ember/internal/colourfont/svgglyph"
	"github.com/lantervale/ember/svg"
	"github.com/lantervale/ember/ui"
)

const mime = "image/svg+xml"

//nolint:gochecknoinits // registers the format on import, as audio's decoders do
func init() {
	ui.RegisterVectorFormat(mime, is, parse)
	colourfont.RegisterSVG(svgglyph.Read)
}

// is reports whether the file name with these bytes is an SVG picture.
func is(name string, data []byte) bool {
	if i := strings.IndexAny(name, "?#"); i >= 0 {
		name = name[:i]
	}
	return len(name) >= 4 && strings.EqualFold(name[len(name)-4:], ".svg") || sniff(data)
}

// sniffed is how far into a file its svg element is looked for.
const sniffed = 4096

// sniff reports whether data starts with an svg element, after a byte
// order mark, white space, an XML declaration, processing instructions,
// comments and a doctype, its internal subset included.
func sniff(data []byte) bool {
	data = narrow(data[:min(len(data), 2*sniffed)])
	data = bytes.TrimPrefix(data[:min(len(data), sniffed)], []byte("\xef\xbb\xbf"))
	for {
		data = bytes.TrimLeft(data, " \t\r\n")
		var end string
		switch {
		case bytes.HasPrefix(data, []byte("<?")):
			end = "?>"
		case bytes.HasPrefix(data, []byte("<!--")):
			end = "-->"
		case len(data) >= 9 && bytes.EqualFold(data[:9], []byte("<!DOCTYPE")):
			end = ">"
			if open, shut := bytes.IndexByte(data, '['), bytes.IndexByte(data, '>'); open >= 0 && open < shut {
				end = "]"
			}
		default:
			rest, ok := bytes.CutPrefix(data, []byte("<svg"))
			return ok && len(rest) > 0 && strings.IndexByte(" \t\r\n>/", rest[0]) >= 0
		}
		i := bytes.Index(data, []byte(end))
		if i < 0 {
			return false
		}
		data = data[i+len(end):]
		if end == "]" {
			if i = bytes.IndexByte(data, '>'); i < 0 {
				return false
			}
			data = data[i+1:]
		}
	}
}

// narrow is UTF-16 data, known by its byte order mark or its first '<',
// a byte a character, those past ASCII as 0xff; other data as it is.
func narrow(data []byte) []byte {
	low := 1
	switch {
	case bytes.HasPrefix(data, []byte{0xfe, 0xff}), bytes.HasPrefix(data, []byte{0, '<'}):
	case bytes.HasPrefix(data, []byte{0xff, 0xfe}), bytes.HasPrefix(data, []byte{'<', 0}):
		low = 0
	default:
		return data
	}
	if data[0] > 0x7f {
		data = data[2:]
	}
	out := make([]byte, len(data)/2)
	for i := range out {
		out[i] = data[2*i+low]
		if data[2*i+1-low] != 0 || out[i] > 0x7f {
			out[i] = 0xff
		}
	}
	return out
}

func parse(data []byte) (ui.VectorImage, error) {
	p, err := svg.NewPicture(data)
	if err != nil {
		return nil, err
	}
	return picture{p}, nil
}

// picture is an svg.Picture drawn within the reader's limits.
type picture struct{ *svg.Picture }

// Rasterise draws the picture w×h pixels large for a screen of scale
// pixels a CSS pixel, or the largest size of that shape within
// svg.MaxSide and svg.MaxPixels, its scale as much smaller.
func (p picture) Rasterise(w, h int, scale float64) (*image.RGBA, error) {
	w, h, scale = fit(w, h, scale)
	return p.RasteriseScaled(w, h, scale)
}

func fit(w, h int, scale float64) (int, int, float64) {
	w, h = max(w, 1), max(h, 1)
	k := min(1, svg.MaxSide/float64(w), svg.MaxSide/float64(h), math.Sqrt(svg.MaxPixels/(float64(w)*float64(h))))
	if k < 1 {
		w, h, scale = max(int(float64(w)*k), 1), max(int(float64(h)*k), 1), scale*k
	}
	return w, h, scale
}
