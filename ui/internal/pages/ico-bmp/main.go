// ICO and BMP pictures against Chrome's: one 12×12 picture of four
// colours, scaled up, in each form ui/web reads. The icons: 1, 4, 8, 16
// and 24 bits a pixel cut by their AND mask, 32 bits with alpha (its
// mask saying nothing), 32 bits whose alpha is zero everywhere (cut by
// its mask), a PNG, and files of several pictures, of which the largest
// then the deepest shows. The bitmaps: 1, 4, 8, 16 (555, then 565 by
// masks), 24 (from the bottom, from the top) and 32 bits: alpha bits no
// header names (opaque), none at all, the masks of a V4 header, masks
// without alpha, the alpha mask of a V5 header without compression, and
// a V4 header's alpha all zero (nothing shows). Then pictures run-length
// encoded, in bytes and in nibbles, in an icon and in a bitmap, whose
// runs skip pixels (black in Chrome) and stop where they would leave the
// picture; two cursors; and last an icon of 12 and 24 pixels shown at 12
// points, where Chrome shows the larger, scaled down. The files are
// written here, by icotest.
package main

import (
	"fmt"
	"image/color"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/ico/icotest"
	"github.com/lantervale/ember/ui/internal/pages"
	_ "github.com/lantervale/ember/ui/web"
)

const sheet = `
html { background-color: #d8c8a0; }
body { margin: 0; padding: 12px; display: grid; grid-template-columns: repeat(8, 48px); gap: 12px; }
img { display: block; width: 48px; height: 48px; image-rendering: pixelated; }
`

const (
	side    = 12
	icoKind = "ico"

	rle8      = 1
	rle4      = 2
	bitfields = 3
	headerV4  = 108
	headerV5  = 124
)

func palette() []color.NRGBA {
	return []color.NRGBA{{200, 40, 40, 255}, {40, 160, 60, 255}, {40, 80, 200, 255}, {250, 250, 250, 255}}
}

// index is the colour of a pixel: a quarter of the picture each.
func index(x, y int) uint32 {
	i := 0
	if x >= side/2 {
		i++
	}
	if y >= side/2 {
		i += 2
	}
	return uint32(i)
}

func bit(x, y int) uint32 { return index(x, y) % 2 }

func bgr(x, y int) uint32 {
	c := palette()[index(x, y)]
	return uint32(c.B) | uint32(c.G)<<8 | uint32(c.R)<<16
}

func rgb(x, y int) uint32 {
	c := palette()[index(x, y)]
	return uint32(c.R) | uint32(c.G)<<8 | uint32(c.B)<<16
}

// faded is bgr with alpha growing to the right, none on the first
// column.
func faded(x, y int) uint32 { return bgr(x, y) | uint32(x*255/(side-1))<<24 } //nolint:gosec // 0 to 255

func packed(rbits, gbits, bbits int) func(x, y int) uint32 {
	return func(x, y int) uint32 {
		c := palette()[index(x, y)]
		r, g, b := uint32(c.R)>>(8-rbits), uint32(c.G)>>(8-gbits), uint32(c.B)>>(8-bbits)
		return r<<(gbits+bbits) | g<<bbits | b
	}
}

// corner is what an icon's AND mask cuts: the top-left corner.
func corner(x, y int) bool { return x+y < 5 }

func bitmap(bpp int, at func(x, y int) uint32) icotest.Bitmap {
	b := icotest.Bitmap{W: side, H: side, BPP: bpp, At: at, Hole: corner}
	if bpp <= 8 {
		b.Palette = palette()[:min(4, 1<<bpp)]
	}
	return b
}

// runs8 is the picture run-length encoded in bytes, a row a run each
// side and an end of line, then the end of the bitmap.
func runs8() []byte {
	var out []byte
	for i := range side {
		y := side - 1 - i
		out = append(out, side/2, byte(index(0, y)), side/2, byte(index(side-1, y)), 0, 0) //nolint:gosec // 0 to 3
	}
	return append(out, 0, 1)
}

// runs4 is the picture run-length encoded in nibbles, its top rows two
// colours striped by the runs' alternating nibbles.
func runs4() []byte {
	var out []byte
	for i := range side {
		y := side - 1 - i
		l, r := byte(index(0, y)), byte(index(side-1, y)) //nolint:gosec // 0 to 3
		if y < 2 {
			r = 3
		}
		out = append(out, side/2, l<<4|l, side/2, r<<4|l, 0, 0)
	}
	return append(out, 0, 1)
}

// skips is a bitmap in runs of bytes that leaves pixels out, which show
// nothing: one row in absolute runs (one odd, padded) ended early by an
// end of line, a move over pixels and up a row, then the end of the
// bitmap halfway up.
func skips() []byte {
	return []byte{
		0, 5, 0, 1, 2, 3, 0, 0, 0, 3, 1, 1, 1, 0, 0, 0,
		4, 2, 0, 2, 3, 1, 3, 3, 0, 0,
		side, 1, 0, 0,
		0, 2, 2, 0, side - 2, 3, 0, 0,
		0, 1,
	}
}

// nibbleSkips is skips in nibbles: an absolute run of three, an end of
// line, a move, the end of the bitmap.
func nibbleSkips() []byte {
	return []byte{
		0, 3, 0x12, 0x30, 5, 0x21, 0, 0,
		side, 0x33, 0, 2, 3, 1, 4, 0x10, 0, 0,
		0, 1,
	}
}

func coded(comp uint32, bpp int, runs []byte) icotest.Bitmap {
	b := icotest.Bitmap{W: side, H: side, BPP: bpp, Comp: comp, Palette: palette(), Runs: runs}
	return b
}

func flat(size, bpp int, c uint32) icotest.Entry {
	return icotest.Bitmap{W: size, H: size, BPP: bpp, At: func(int, int) uint32 { return c }}.Entry()
}

func icons() [][]byte {
	out := [][]byte{}
	for _, b := range []icotest.Bitmap{
		bitmap(1, bit), bitmap(4, index), bitmap(8, index), bitmap(16, packed(5, 5, 5)), bitmap(24, bgr),
		bitmap(32, faded), bitmap(32, bgr),
	} {
		out = append(out, icotest.ICO(b.Entry()))
	}
	return append(out,
		icotest.ICO(icotest.Entry{W: side, H: side, BPP: 32, Data: icotest.PNG(side, side, color.NRGBA{40, 80, 200, 128})}),
		icotest.ICO(flat(8, 32, 0xffc82828), bitmap(8, index).Entry(), flat(10, 32, 0xff2850c8)),
		icotest.ICO(flat(side, 4, 0), bitmap(32, faded).Entry(), flat(side, 24, 0x28a03c)),
		icotest.ICO(coded(rle8, 8, runs8()).Entry()),
		icotest.ICO(coded(rle4, 4, runs4()).Entry()),
		icotest.CUR(bitmap(8, index).Entry()),
		icotest.CUR(flat(8, 32, 0xffc82828), bitmap(32, faded).Entry()),
	)
}

func bitmaps() [][]byte {
	top := bitmap(24, bgr)
	top.TopDown = true
	v4 := bitmap(32, faded)
	v4.Header, v4.Comp, v4.Masks = headerV4, bitfields, []uint32{0xff0000, 0xff00, 0xff, 0xff000000}
	swapped := bitmap(32, func(x, y int) uint32 { return rgb(x, y) | 0x80000000 })
	swapped.Comp, swapped.Masks = bitfields, []uint32{0xff, 0xff00, 0xff0000}
	v5 := bitmap(32, faded)
	v5.Header, v5.Masks = headerV5, []uint32{0xff0000, 0xff00, 0xff, 0xff000000}
	blank := bitmap(32, bgr)
	blank.Header, blank.Comp, blank.Masks = headerV4, bitfields, v4.Masks
	high := bitmap(16, packed(5, 6, 5))
	high.Comp, high.Masks = bitfields, []uint32{0xf800, 0x7e0, 0x1f}
	out := [][]byte{}
	for _, b := range []icotest.Bitmap{
		bitmap(1, bit), bitmap(4, index), bitmap(8, index), bitmap(16, packed(5, 5, 5)), high, bitmap(24, bgr), top,
		bitmap(32, faded), bitmap(32, bgr), v4, swapped, v5, blank,
		coded(rle8, 8, runs8()), coded(rle4, 4, runs4()), coded(rle8, 8, skips()), coded(rle4, 4, nibbleSkips()),
	} {
		out = append(out, b.File())
	}
	return out
}

func main() {
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"))
	for kind, pictures := range map[string][][]byte{icoKind: icons(), "bmp": bitmaps()} {
		if kind == icoKind {
			n := len(pictures) - 2
			for i, data := range pictures[n:] {
				files[fmt.Sprintf("%02d.cur", i)] = &fstest.MapFile{Data: data}
			}
			pictures = pictures[:n]
		}
		for i, data := range pictures {
			files[fmt.Sprintf("%02d.%s", i, kind)] = &fstest.MapFile{Data: data}
		}
	}
	for _, kind := range []string{icoKind, "cur", "bmp"} {
		for i := 0; ; i++ {
			name := fmt.Sprintf("%02d.%s", i, kind)
			if _, ok := files[name]; !ok {
				break
			}
			d.Body().Append(ui.Img(ui.Attr("src", name)))
		}
	}
	files["small.ico"] = &fstest.MapFile{Data: icotest.ICO(bitmap(8, index).Entry(), flat(2*side, 32, 0xffc82828))}
	d.Body().Append(ui.Img(ui.Attr("src", "small.ico"), ui.Style("width: 12px; height: 12px")))
	pages.Run(d, 492, 264)
}
