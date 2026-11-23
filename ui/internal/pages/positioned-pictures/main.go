// Pictures across writing modes against Chrome: pictures absolutely positioned out of
// vertical-rl, vertical-lr and sideways-lr boxes into the page's horizontal one, of
// their natural size, a max-height, an aspect-ratio alone and an object-fit; a
// horizontal picture standing on a vertical line; a picture in a horizontal box placed
// by a vertical one; a canvas leaving a vertical box; and ranges of appearance
// slider-vertical whose height is auto.
package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; margin: 0; display: flex; gap: 10px; align-items: flex-start; flex-wrap: wrap; }
.v { inline-size: 90px; block-size: 56px; border: 1px solid #c8473d; padding: 3px; }
.v > p { margin: 0; inline-size: 24px; block-size: 16px; background-color: #e8e2d6; }
.rl { writing-mode: vertical-rl; }
.lr { writing-mode: vertical-lr; }
.up { writing-mode: sideways-lr; }
.a { position: absolute; margin: 2px; }
.max { max-height: 14px; }
.square { aspect-ratio: 1; }
.fit { width: 36px; height: 36px; object-fit: contain; background-color: #3a6fd8; }
.rel { position: relative; }
.across { writing-mode: horizontal-tb; width: 60px; height: 50px; background-color: #f0c040; }
canvas.a { width: 30px; background-color: #3a6fd8; }
input { margin: 2px; }
.thin { width: 24px; }
.flat { writing-mode: horizontal-tb; }
`

// stripes is a 40×16 picture of three bands, its top-left corner marked: which way it
// stands shows.
func stripes() string {
	img := image.NewNRGBA(image.Rect(0, 0, 40, 16))
	bands := []color.NRGBA{{200, 71, 61, 255}, {58, 111, 216, 255}, {43, 39, 34, 255}}
	for y := range 16 {
		for x := range 40 {
			img.Set(x, y, bands[x*3/40])
			if x < 6 && y < 6 {
				img.Set(x, y, color.NRGBA{240, 192, 64, 255})
			}
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

func main() {
	src := stripes()
	pic := func(class ...string) *ui.Node { return ui.Img(ui.Class(class...), ui.Attr("src", src)) }
	box := func(mode string, kid *ui.Node) *ui.Node { return ui.Div(ui.Class("v", mode), ui.P(), kid, ui.P()) }
	d := ui.NewDocument(ui.CSS(sheet))
	d.Body().Append(
		box("rl", pic("a")),
		box("lr", pic("a", "max")),
		box("up", pic("a", "square")),
		box("rl", pic("a", "fit")),
		box("rl", pic("flat")),
		ui.Div(ui.Class("v", "rl", "rel"), ui.P(), ui.Div(ui.Class("across"), pic("a")), ui.P()),
		box("lr", ui.Canvas(nil, ui.Class("a"), ui.Attr("width", "40"), ui.Attr("height", "16"))),
		ui.Div(ui.Input(ui.Type("range"), ui.Style("appearance: slider-vertical")),
			ui.Input(ui.Type("range"), ui.Class("thin"), ui.Style("appearance: slider-vertical"))),
	)
	pages.Run(d, 560, 260)
}
