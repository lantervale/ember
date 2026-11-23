// Chrome's own masks against Chrome's: a mask placed along one axis by
// -webkit-mask-position-x or -y, and -webkit-mask-box-image, a picture
// cut in nine over the border box, stretched, repeated, rounded, with
// widths, an outset, and its middle left out.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; }
body > div { display: grid; grid-template-columns: repeat(5, 64px); gap: 14px; }
div > div { height: 64px; background: linear-gradient(135deg, #b0493c, #3c6fb0); }
.at { -webkit-mask: radial-gradient(circle, #000 0 10px, transparent 10px) no-repeat; -webkit-mask-size: 24px 24px; }
.x { -webkit-mask-position-x: right 4px; }
.y { -webkit-mask-position-y: bottom 6px; }
.xy { -webkit-mask-position: center; -webkit-mask-position-x: left 3px; }
.yx { -webkit-mask-position-x: 30%; -webkit-mask-position-y: center, 2px; }
.two { -webkit-mask-image: radial-gradient(circle, #000 0 10px, transparent 10px),
  linear-gradient(#000 0 0); -webkit-mask-size: 24px 24px, 8px 8px; -webkit-mask-position-x: 80%, right; }
.b1 { -webkit-mask-box-image: url(m.png) 8; }
.b2 { -webkit-mask-box-image: url(m.png) 8 / 12px round; }
.b3 { border: 6px solid #222; -webkit-mask-box-image: url(m.png) 8 / 2 / 4px repeat; }
.b4 { -webkit-mask-box-image-source: url(m.png); -webkit-mask-box-image-slice: 8; }
.b5 { -webkit-mask-box-image: url(m.png) 8 space stretch;
  -webkit-mask-image: radial-gradient(circle, #000 0 16px, transparent 16px); }
`

// mask is a 24-pixel picture whose alpha shows how it is cut: discs in
// its corners, stripes along its edges and a half-clear middle.
func mask() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	for y := range 24 {
		for x := range 24 {
			a := uint8(128)
			cx, cy := x/8, y/8
			switch {
			case cx != 1 && cy != 1:
				dx, dy := x%8-4, y%8-4
				if dx*dx+dy*dy <= 12 {
					a = 255
				} else {
					a = 0
				}
			case cx != 1:
				a = uint8(255 * ((y / 2) % 2))
			case cy != 1:
				a = uint8(255 * ((x / 2) % 2))
			}
			img.SetNRGBA(x, y, color.NRGBA{A: a})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}, "m.png": {Data: mask()}},
		"page.css"), ui.BrowserSheets())
	grid := ui.Div()
	for _, c := range []string{"at x", "at y", "at xy", "at yx", "two", "b1", "b2", "b3", "b4", "b5"} {
		grid.Append(ui.Div(ui.Class(strings.Fields(c)...)))
	}
	d.Body().Append(grid)
	pages.Run(d, 380, 180)
}
