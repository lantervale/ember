// ICO and BMP pictures against Chrome's: one 12×12 picture of four
// colours, scaled up, in each form ui/web reads. The icons: 1, 4, 8, 16
// and 24 bits a pixel cut by their AND mask, 32 bits with alpha (its
// mask saying nothing), 32 bits whose alpha is zero everywhere (cut by
// its mask), a PNG, and files of several pictures, of which the largest
// then the deepest shows. The bitmaps: 1, 4, 8, 16 (555, then 565 by
// masks), 24 (from the bottom, from the top) and 32 bits: alpha bits no
// header names (opaque), none at all, the masks of a V4 header, masks
// without alpha, the alpha mask of a V5 header without compression, and
// a V4 header's alpha all zero (nothing shows). The files are written
// here, by icotest.
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
	side = 12

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
	} {
		out = append(out, b.File())
	}
	return out
}

func main() {
	files := fstest.MapFS{"page.css": {Data: []byte(sheet)}}
	d := ui.NewDocument(ui.Stylesheet(files, "page.css"))
	for kind, pictures := range map[string][][]byte{"ico": icons(), "bmp": bitmaps()} {
		for i, data := range pictures {
			files[fmt.Sprintf("%02d.%s", i, kind)] = &fstest.MapFile{Data: data}
		}
	}
	for _, kind := range []string{"ico", "bmp"} {
		for i := 0; ; i++ {
			name := fmt.Sprintf("%02d.%s", i, kind)
			if _, ok := files[name]; !ok {
				break
			}
			d.Body().Append(ui.Img(ui.Attr("src", name)))
		}
	}
	pages.Run(d, 492, 204)
}
