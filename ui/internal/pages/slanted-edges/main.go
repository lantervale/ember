// Edges at a slant: an img, a picture stretched over its box and one tiled
// in it, square borders of one colour, of four colours, of unequal widths
// and double, a border-image, and gradients with hard stops, each turned,
// skewed and scaled with a turn; the gradients again unturned, their stops
// at an angle. Chrome anti-aliases the edges of the pictures, borders and
// boxes, and leaves the hard stops stepped.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 20px; display: grid; grid-template-columns: repeat(8, 56px); gap: 34px 22px; }
.b { height: 40px; }
img.b { width: 56px; display: block; }
.turn { rotate: 30deg; }
.skew { transform: skewX(-20deg); }
.scale { transform: scale(0.8, 1.3) rotate(-15deg); }
.stretched { background: url(sky.png) 0 0 / 100% 100% no-repeat; }
.tiled { background: url(sky.png) 0 0 / 28px 20px; }
.solid { box-sizing: border-box; border: 4px solid #345; background: #fde9c8; }
.colours { box-sizing: border-box; border: 6px solid; border-color: #c33 #2a8 #36c #f0b545; }
.widths { box-sizing: border-box; border: solid #345; border-width: 2px 8px 5px 3px; background: #cde; }
.double { box-sizing: border-box; border: 9px double #345; }
.stop { background: linear-gradient(45deg, #36c 50%, #f0b545 50%); }
.framed { box-sizing: border-box; border: 8px solid; border-image: url(sky.png) 6 fill / 8px stretch; }
.stops { background: linear-gradient(110deg, #c33 33%, #2a8 33% 66%, #36c 66%); }
`

// sky is a 28×20 picture: a sky fading to a sea, a sun, a band of sand.
func sky() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 28, 20))
	for y := range 20 {
		for x := range 28 {
			c := color.RGBA{uint8(90 + 6*y), uint8(150 + 4*y), 230, 255}
			switch {
			case y >= 15:
				c = color.RGBA{232, 200, 130, 255}
			case y >= 10:
				c = color.RGBA{40, uint8(100 + 8*(y-10)), 170, 255}
			case (x-20)*(x-20)+(y-4)*(y-4) <= 6:
				c = color.RGBA{250, 230, 90, 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"t.css":   {Data: []byte(sheet)},
		"sky.png": {Data: sky()},
	}, "t.css"))
	turns := []string{"turn", "skew", "scale"}
	kinds := []string{"img", "stretched", "tiled", "solid", "colours", "widths", "double", "stop"}
	for _, turn := range turns {
		for _, k := range kinds {
			if k == "img" {
				d.Body().Append(ui.Img(ui.Attr("src", "sky.png"), ui.Class("b", turn)))
				continue
			}
			d.Body().Append(ui.Div(ui.Class("b", k, turn)))
		}
	}
	for _, k := range []string{"stop", "stops"} {
		d.Body().Append(ui.Div(ui.Class("b", k)))
	}
	for _, turn := range turns {
		d.Body().Append(ui.Div(ui.Class("b", "stops", turn)))
	}
	for _, turn := range turns {
		d.Body().Append(ui.Div(ui.Class("b", "framed", turn)))
	}
	pages.Run(d, 660, 380)
}
