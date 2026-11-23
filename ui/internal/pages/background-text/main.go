// Backgrounds shown through text: a gradient and a picture clipped to the
// glyphs of an element and of those it holds, the -webkit- spelling with
// -webkit-text-fill-color, a layer clipped to text over a colour clipped
// to the border box, glyphs filled apart from their underline; a picture
// fixed under a transform, placed in its element; the canvas's picture
// placed in html's box; and a border-image of shapes drawn at the
// screen's density, its middle drawn only when filled; markers shadowed.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
	_ "github.com/lantervale/ember/ui/svg"
)

const sheet = `
html { margin: 14px 0 0 24px; background: #fff url(mark.png) no-repeat 2px 2px; }
body { margin: 0; padding: 12px 10px; display: flex; flex-wrap: wrap; gap: 10px 16px; align-items: flex-start;
  font-weight: bold; font-size: 40px; }
.gradient { background: linear-gradient(90deg, #c33, #36c 70%, #2a8); background-clip: text; color: transparent; }
.picture { background: url(mark.png) text; color: transparent; font-size: 52px; }
.webkit { background: linear-gradient(#f0b545, #345); -webkit-background-clip: text;
  -webkit-text-fill-color: transparent; color: #c33; text-decoration: underline; }
.held { background: linear-gradient(#36c, #c33); background-clip: text; color: rgba(255, 255, 255, 0.3);
  padding: 4px 8px; border: 3px dashed #345; }
.held span { color: transparent; font-size: 28px; }
.layers { background: linear-gradient(135deg, #c33, #f0b545) text, #cde; color: transparent; padding: 2px 10px; }
.fill { -webkit-text-fill-color: #36c; color: #c33; text-decoration: underline; font-size: 32px; }
.turned { transform: translate(6px, 2px); }
.fixed { width: 120px; height: 56px; border: 2px solid #345;
  background: url(mark.png) fixed 10px 12px no-repeat #dfe; }
.frame { width: 90px; height: 40px; border: 12px solid; border-image: url(frame.svg) 12 round; }
.filled { border-image: url(frame.svg) 12 fill round; }
ol { margin: 0; padding-left: 40px; font-size: 20px; font-weight: normal; }
li::marker { color: #36c; text-shadow: 2px 2px #f0b545, -1px 0 1px #c33; }
`

const frame = `<svg xmlns="http://www.w3.org/2000/svg" width="36" height="36">
<rect x="1.5" y="1.5" width="33" height="33" rx="9" fill="#fde9c8" stroke="#345" stroke-width="3"/>
<circle cx="18" cy="18" r="4.5" fill="#c33"/></svg>`

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
	d := ui.NewDocument(ui.BrowserSheets(), ui.Stylesheet(fstest.MapFS{
		"background-text.css": {Data: []byte(sheet)}, "mark.png": {Data: mark()},
		"frame.svg": {Data: []byte(frame)},
	}, "background-text.css"))
	div := func(class, text string) *ui.Node { return ui.Div(ui.Class(class), ui.Text(text)) }
	d.Body().Append(
		div("gradient", "Lantern"),
		div("picture", "Vale"),
		div("webkit", "Ember"),
		ui.Div(ui.Class("held"), ui.Text("Held "), ui.Span(ui.Text("and its child"))),
		div("layers", "Layers"),
		div("fill", "Filled apart"),
		ui.Div(ui.Class("turned"), ui.Div(ui.Class("fixed"))),
		ui.Div(ui.Class("frame")), ui.Div(ui.Class("frame", "filled")),
		ui.Ol(ui.Li(ui.Text("a shadowed marker")), ui.Li(ui.Text("and another"))),
	)
	pages.Run(d, 640, 300)
}
