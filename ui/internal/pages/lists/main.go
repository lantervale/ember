// Lists against Chrome's: markers written by @counter-style rules of each
// system and by predefined styles, string markers, the type attribute of
// ol and li, list-style-image, a ::before that is a list item with its
// ::before::marker, an element content: url() replaces, and checkboxes
// with an ::after.
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 13px; line-height: 17px; color: #2b2722; display: grid;
	grid-template-columns: 1fr 1fr 1fr; gap: 4px 10px; }
ol, ul { margin: 0; padding-left: 34px; background-color: #f4f1ec; }
@counter-style paren { system: extends lower-alpha; prefix: "("; suffix: ") "; }
@counter-style tally { system: additive; additive-symbols: 5 "V", 1 "I"; suffix: ": "; }
@counter-style base2 { system: numeric; symbols: "0" "1"; pad: 3 "0"; }
@counter-style stars { system: symbolic; symbols: "*"; suffix: " "; }
@counter-style shapes { system: cyclic; symbols: "+" "-" "="; suffix: " "; }
@counter-style low { system: fixed 2; symbols: x y; fallback: upper-roman; }
.paren { list-style-type: paren; }
.tally { list-style-type: tally; }
.base2 { list-style-type: base2; color: #3c6fb0; }
.stars { list-style-type: stars; }
.shapes { list-style-type: shapes; }
.low { list-style-type: low; }
.greek { list-style-type: lower-greek; }
.str { list-style-type: "-> "; }
.str li:nth-child(2) { list-style-type: "\2014  "; color: #b0493c; }
.img { list-style-image: url(dot.png); }
.img li:nth-child(2) { font-size: 22px; line-height: 26px; }
.gen { padding-left: 34px; background-color: #f4f1ec; }
.gen div::before { content: "a ::before"; display: list-item; list-style-type: upper-roman; }
.gen div:nth-child(2)::before { list-style-type: disc; }
.gen div:nth-child(3)::before { list-style-type: "~ "; }
.gen div:nth-child(3)::before::marker { color: #b0493c; font-weight: 700; }
.pic { content: url(pic.png); border: 2px solid #3c8a4f; }
input::after { content: "off"; margin-left: 4px; color: #b0493c; }
input:checked::after { content: "on"; color: #3c8a4f; }
`

func picture(w, h int, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"lists.css": {Data: []byte(sheet)}}, "lists.css"),
		ui.Images(fstest.MapFS{
			"dot.png": {Data: picture(8, 8, color.NRGBA{R: 0x3c, G: 0x8a, B: 0x4f, A: 0xff})},
			"pic.png": {Data: picture(40, 24, color.NRGBA{R: 0x7a, G: 0x4f, B: 0xb0, A: 0xff})},
		}))
	list := func(tag func(...ui.Arg) *ui.Node, n int, args ...ui.Arg) *ui.Node {
		for range n {
			args = append(args, ui.Li(ui.Text("item")))
		}
		return tag(args...)
	}
	d.Body().Append(
		list(ui.Ol, 3, ui.Class("paren"), ui.Attr("start", "25")),
		list(ui.Ol, 3, ui.Class("tally"), ui.Attr("start", "4")),
		list(ui.Ol, 3, ui.Class("base2")),
		list(ui.Ol, 3, ui.Class("stars")),
		list(ui.Ol, 3, ui.Class("shapes")),
		list(ui.Ol, 3, ui.Class("low"), ui.Attr("start", "2")),
		list(ui.Ol, 3, ui.Class("greek")),
		list(ui.Ul, 2, ui.Class("str")),
		list(ui.Ul, 2, ui.Class("img")),
		list(ui.Ol, 3, ui.Attr("type", "A"), ui.Attr("start", "26")),
		ui.Ol(ui.Attr("type", "i"), ui.Li(ui.Text("item")), ui.Li(ui.Attr("type", "a"), ui.Text("item")),
			ui.Li(ui.Text("item"))),
		ui.Ol(ui.Attr("reversed", ""), ui.Li(ui.Text("item")), ui.Li(ui.Text("item")), ui.Li(ui.Text("item"))),
		ui.Div(ui.Class("gen"), ui.Div(), ui.Div(), ui.Div()),
		ui.Div(ui.Div(ui.Class("pic"), ui.Text("replaced")), ui.Text("after")),
		ui.Div(ui.Input(ui.Attr("type", "checkbox")), ui.Input(ui.Attr("type", "checkbox"), ui.Attr("checked", ""))),
	)
	pages.Run(d, 520, 300)
}
