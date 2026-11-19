// Vertical alignment against Chrome's: each value on a span, an
// inline-block and a picture, boxes on the top and the bottom of their
// line, nested alignments, decorations and backgrounds moving with their
// boxes, and the cells of a table.
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
body { padding: 8px; color: #2b2722; font-size: 16px; }
p { margin: 0 0 6px; background-color: #f1ece2; }
p span { background-color: #c9dcf5; font-size: 11px; }
b { display: inline-block; width: 12px; height: 26px; background-color: #c8473d; }
i { display: inline-block; width: 12px; height: 8px; background-color: #3a6fd8; }
.sub { vertical-align: sub; } .super { vertical-align: super; }
.text-top { vertical-align: text-top; } .text-bottom { vertical-align: text-bottom; }
.middle { vertical-align: middle; } .top { vertical-align: top; } .bottom { vertical-align: bottom; }
.up { vertical-align: 7px; } .down { vertical-align: -40%; }
.tall { line-height: 44px; }
.under { text-decoration: underline #3a6fd8; }
.boxed { border: 1px solid #2b2722; padding: 1px 3px; }
td { width: 60px; height: 44px; padding: 2px; border: 1px solid #2b2722; }
`

func line(class string) *ui.Node {
	return ui.P(ui.Text("Hxg "+class+" "), ui.Span(ui.Class(class), ui.Text("span")), ui.Text(" "), ui.B(ui.Class(class)),
		ui.Text(" "), ui.I(ui.Class(class)), ui.Text(" "), ui.Img(ui.Attr("src", "tile.png"), ui.Class(class)),
		ui.Text(" end"))
}

// tile is a green picture, 14 by 22.
func tile() []byte {
	m := image.NewNRGBA(image.Rect(0, 0, 14, 22))
	for y := range 22 {
		for x := range 14 {
			m.Set(x, y, color.NRGBA{R: 0x4a, G: 0x8f, B: 0x5c, A: 0xff})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"vertical-align.css": {Data: []byte(sheet)},
		"tile.png": {Data: tile()}}, "vertical-align.css"))
	for _, class := range []string{"baseline", "sub", "super", "text-top", "text-bottom", "middle", "top", "bottom",
		"up", "down"} {
		d.Body().Append(line(class))
	}
	d.Body().Append(
		ui.P(ui.Class("tall"), ui.Text("A tall line: "), ui.Span(ui.Class("top"), ui.Text("top")), ui.Text(" "),
			ui.Span(ui.Class("middle"), ui.Text("middle")), ui.Text(" "), ui.Span(ui.Class("bottom"), ui.Text("bottom")),
			ui.Text(" "), ui.I(ui.Class("top")), ui.I(ui.Class("middle")), ui.I(ui.Class("bottom"))),
		ui.P(ui.Text("Nested: "), ui.Span(ui.Class("super", "under"), ui.Text("raised and underlined "),
			ui.B(ui.Class("middle")), ui.Span(ui.Class("sub", "boxed"), ui.Text("then lowered"))), ui.Text(" back.")),
		ui.P(ui.Text("Top "), ui.B(ui.Class("top"), ui.Style("height: 40px")), ui.Text(" and bottom "),
			ui.B(ui.Class("bottom"), ui.Style("height: 30px")), ui.Text(" boxes, "),
			ui.Span(ui.Class("top"), ui.Text("a top span holding "), ui.I(ui.Style("height: 20px"))), ui.Text(".")),
		ui.Table(ui.Tr(ui.Td(ui.Text("plain")), ui.Td(ui.Class("top"), ui.Text("top")),
			ui.Td(ui.Class("middle"), ui.Text("middle")), ui.Td(ui.Class("bottom"), ui.Text("bottom")),
			ui.Td(ui.Attr("valign", "bottom"), ui.B()))),
	)
	pages.Run(d, 420, 620)
}
