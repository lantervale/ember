// Pixel-art pictures under a fractional Document zoom (EMBER_UI_ZOOM, 1.5
// by default), as a game whose art zoom is whole has on a 2× screen: a
// 16-pixel checker as a pixelated img at its own size, at 24 and at 40
// points, at places off the point grid, and the same as pixelated
// backgrounds below. At 24 and 40 points a texel is a whole and a half
// device pixels wide, so every other line between texels crosses a
// pixel's centre: Chrome gives that pixel the upper left texel.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #ffffff; }
body { margin: 0; }
img, .bg { position: absolute; width: 16px; height: 16px; top: 8.25px; image-rendering: pixelated; }
.bg { top: 56.5px; background-image: url(checker.png); background-size: 100% 100%; }
.a { left: 8.25px; }
.b { left: 40.5px; width: 24px; height: 24px; }
.c { left: 72.75px; width: 24px; height: 24px; }
.d { left: 104.4px; width: 40px; height: 40px; }
`

// checker is a 16-pixel picture whose pixels alternate red and blue.
func checker() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			c := color.RGBA{255, 0, 0, 255}
			if (x+y)%2 == 1 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	zoom := 1.5
	if z, err := strconv.ParseFloat(os.Getenv("EMBER_UI_ZOOM"), 32); err == nil && z > 0 {
		zoom = z
	}
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "checker.png": {Data: checker()}},
		"page.css"), ui.Zoom(float32(zoom)))
	for _, at := range []string{"a", "b", "c", "d"} {
		d.Body().Append(ui.Img(ui.Attr("src", "checker.png"), ui.Class(at)), ui.Div(ui.Class("bg", at)))
	}
	pages.Run(d, int(160*zoom), int(104*zoom))
}
