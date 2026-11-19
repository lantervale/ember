// Where a background shows and where its picture is placed: the box
// background-clip names, the box background-origin names, a picture fixed
// in the viewport or scrolling with its scroller's content, the shorthand
// with its parts in any order, and an inline over several lines, whose
// background is one box sliced by its lines, as a link of the web has its
// icon after its last word.
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
body { padding: 10px; display: flex; flex-wrap: wrap; gap: 12px; align-items: flex-start; font-size: 13px; }
.box { width: 90px; height: 50px; padding: 8px; border: 6px dashed #345; background: #fde9c8 url(mark.png); }
.clip-padding { background-clip: padding-box; } .clip-content { background-clip: content-box; }
.origin-border { background-origin: border-box; background-repeat: no-repeat; }
.origin-content { background-origin: content-box; background-repeat: no-repeat; }
.short { background: content-box padding-box url(mark.png) no-repeat right bottom #cde; }
.axes { background: url(mark.png) repeat no-repeat 0 50%; }
.fixed { background: fixed url(mark.png) 20px 30px no-repeat #dfe; width: 180px; }
.local, .scroll { height: 50px; width: 110px; overflow: hidden; border: 1px solid #345; }
.local { background: url(mark.png) local no-repeat 0 30px; }
.scroll { background: url(mark.png) scroll no-repeat 0 30px; }
.tall { height: 200px; }
.layers { background: url(mark.png) no-repeat center, url(other.png) fixed, #edf; }
p { width: 190px; margin: 0; line-height: 20px; }
a { color: #36c; }
.external { background: url(mark.png) no-repeat right center; padding-right: 20px; }
.marked { background: #fde9c8 url(mark.png) no-repeat left center; padding: 0 4px 0 18px; border: 1px solid #c93; }
.striped { background: linear-gradient(to right, #cde, #fde9c8); padding: 0 6px; }
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
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"background-boxes.css": {Data: []byte(sheet)},
		"mark.png": {Data: mark()}}, "background-boxes.css"))
	box := func(class ...string) *ui.Node { return ui.Div(ui.Class(append([]string{"box"}, class...)...)) }
	scroller := func(class string) *ui.Node {
		s := ui.Div(ui.Class(class), ui.Div(ui.Class("tall")))
		return s
	}
	d.Body().Append(
		box(), box("clip-padding"), box("clip-content"), box("origin-border"), box("origin-content"),
		box("short"), box("axes"), box("fixed"), box("layers"),
		scroller("local"), scroller("scroll"),
		ui.P(ui.Text("A link "), ui.A(ui.Class("external"), ui.Text("that runs over more than one line of its paragraph")),
			ui.Text(" has its icon after its last word.")),
		ui.P(ui.Text("One "), ui.Span(ui.Class("marked"), ui.Text("marked at its start, with a border round all its lines")),
			ui.Text(" and one "), ui.Span(ui.Class("striped"), ui.Text("with a gradient from its first letter to its last")),
			ui.Text(".")),
	)
	pages.Run(d, 640, 330)
}
