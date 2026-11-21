// Several background layers, the first on top: a gradient over a
// picture, gradients stacked, lists of longhands repeated over the
// pictures, the colour in the last layer's clip box, each layer's own
// origin, attachment and repeat, and pictures tiled by space and round.
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
body { margin: 0; padding: 10px; display: grid; grid-template-columns: repeat(5, 112px); gap: 12px; }
div { height: 80px; padding: 6px; border: 4px dashed #345; background-color: #fde9c8; }
.over { background: linear-gradient(rgba(255, 255, 255, 0), rgba(20, 30, 60, 0.85)),
  url(mark.png) center / cover #cde; }
.stack { background: linear-gradient(90deg, rgba(200, 40, 40, 0.6), transparent 60%),
  linear-gradient(0deg, rgba(40, 90, 200, 0.7), transparent),
  radial-gradient(circle at 30% 40%, #f0b545 20%, transparent 22%), #eee; }
.lists { background-image: url(mark.png), url(mark.png), url(mark.png);
  background-position: left top, right bottom, center; background-repeat: no-repeat; }
.cycled { background-image: url(mark.png), linear-gradient(#36c, #cde), url(mark.png);
  background-repeat: no-repeat, repeat-x; background-size: auto, 20px 30px;
  background-position: right 4px top 4px, left bottom; }
.clips { background: url(mark.png) no-repeat border-box, #9bd content-box; }
.origins { background-image: url(mark.png), url(mark.png); background-origin: border-box, content-box;
  background-repeat: no-repeat; background-position: right bottom; }
.space { background: url(mark.png) space #fde9c8; }
.space-x { background: url(mark.png) space no-repeat 0 50% #fde9c8; }
.round { background: url(mark.png) round #fde9c8; }
.round-x { background: url(mark.png) round no-repeat 0 50% #fde9c8; }
.space-round { background: url(mark.png) space round #fde9c8; }
.lone { width: 34px; background: url(mark.png) space no-repeat right center #fde9c8; }
.fixed { background: url(mark.png) fixed 30px 40px no-repeat, linear-gradient(#dfe, #fde9c8); }
.rounded { border-radius: 16px; border-style: solid;
  background: radial-gradient(circle, transparent 40%, rgba(52, 68, 85, 0.5) 41%), url(mark.png) round; }
.sized { background: url(mark.png) 0 0 / 30px 20px round, linear-gradient(45deg, #345, #cde); }
`

// mark is a picture 14 by 10: a dark frame round a red middle.
func mark() []byte {
	m := image.NewNRGBA(image.Rect(0, 0, 14, 10))
	for y := range 10 {
		for x := range 14 {
			c := color.NRGBA{R: 0xc3, G: 0x33, B: 0x33, A: 0xff}
			if x == 0 || y == 0 || x == 13 || y == 9 {
				c = color.NRGBA{R: 0x22, G: 0x33, B: 0x55, A: 0xff}
			}
			m.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"background-layers.css": {Data: []byte(sheet)},
		"mark.png": {Data: mark()}}, "background-layers.css"))
	for _, class := range []string{
		"over", "stack", "lists", "cycled", "clips", "origins", "space", "space-x", "round", "round-x",
		"space-round", "lone", "fixed", "rounded", "sized",
	} {
		d.Body().Append(ui.Div(ui.Class(strings.Fields(class)...)))
	}
	pages.Run(d, 640, 330)
}
