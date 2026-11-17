package pages

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
)

// The viewport shows no bar: headless Chrome hides a window's and a
// frame's own in a picture.
const tallSheet = `
html { scrollbar-width: none; }
body { margin: 0; padding: 12px; background: #1b1714 url(leaf.png) no-repeat bottom right; color: #f4ede4; }
.head { position: sticky; top: 0; height: 24px; padding: 0 8px; line-height: 24px; background-color: #5b4636; }
.badge { position: fixed; top: 40px; right: 24px; width: 60px; height: 24px; background-color: #ad9773; }
.row { height: 24px; padding: 0 8px; line-height: 24px; border-bottom: 1px solid #3a332c; }
`

// TallPage is a Document whose viewport scrolls, some 1,100 points high:
// a sticky head, a fixed badge, forty rows, and a picture body places at
// its bottom right, which the canvas takes. The viewport-top and
// viewport-end pages show its two ends.
func TallPage() *ui.Document {
	d := ui.NewDocument(ui.ScrollingViewport(),
		ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(tallSheet)}, "leaf.png": {Data: leaf()}}, "page.css"))
	d.Body().Append(ui.Div(ui.Class("row"), ui.Text("above the head")),
		ui.Div(ui.Class("head"), ui.Text("head")), ui.Div(ui.Class("badge")))
	for i := range 40 {
		d.Body().Append(ui.Div(ui.Class("row"), ui.Text(fmt.Sprintf("row %d", i))))
	}
	return d
}

// leaf is a 48×48 picture of two flat colours, a square in a square.
func leaf() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	for y := range 48 {
		for x := range 48 {
			c := color.NRGBA{R: 0x4f, G: 0x8a, B: 0x5b, A: 0xff}
			if x >= 12 && x < 36 && y >= 12 && y < 36 {
				c = color.NRGBA{R: 0xe0, G: 0xb0, B: 0x60, A: 0xff}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}
