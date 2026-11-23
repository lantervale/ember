// Right to left against Chrome: a paragraph of direction rtl aligned to
// its right, text reordered by bdo and bidi-override drawn from its end,
// mirrored brackets, list markers on the right (bullets, numbers, pictures), a table's columns, a
// flex row and a grid from the right, and padded spans in a mirrored line.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

// dot is a 6×6 red picture, a list item's marker.
const dot = "data:image/png;base64," +
	"iVBORw0KGgoAAAANSUhEUgAAAAYAAAAGCAYAAADgzO9IAAAAEklEQVR4nGM44W77HxtmGEgJACYcUo3S" +
	"NAkeAAAAAElFTkSuQmCC"

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 16px; }
p, ul, ol, table { margin: 0 0 6px; }
.box { width: 360px; border: 1px solid #c8473d; }
.pad { padding: 0 8px 0 2px; background-color: #f0c040; }
.row { display: flex; gap: 6px; width: 360px; margin-bottom: 6px; }
.row div, .grid div { height: 14px; background-color: #3a6fd8; }
.grid { display: grid; grid-template-columns: 40px 1fr 60px; gap: 4px; width: 360px; margin-bottom: 6px; }
td { border: 1px solid #2b2722; padding: 2px 6px; }
.dots { list-style-image: url(` + dot + `); }
.roman { list-style-type: lower-roman; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"direction.css": {Data: []byte(sheet)}}, "direction.css"))
	rtl := ui.Attr("dir", "rtl")
	d.Body().Append(
		ui.P(rtl, ui.Class("box"), ui.Text("Words of a paragraph from the right, "),
			ui.Span(ui.Class("pad"), ui.Text("a padded span")), ui.Text(" and the rest (in brackets).")),
		ui.P(ui.Class("box"), ui.Text("Ahead "), ui.Bdo(rtl, ui.Text("reversed (text) here")), ui.Text(" behind.")),
		ui.P(ui.Class("box"), ui.Span(ui.Style("direction: rtl; unicode-bidi: bidi-override"), ui.Text("drawn [backwards]")),
			ui.Text(" then forwards.")),
		ui.Ul(rtl, ui.Class("box"), ui.Li(ui.Text("first")), ui.Li(ui.Text("second"))),
		ui.Ol(rtl, ui.Class("box"), ui.Li(ui.Text("first")), ui.Li(ui.Text("second"))),
		ui.Ul(rtl, ui.Class("box"), ui.Class("dots"), ui.Li(ui.Text("pictured"))),
		ui.Ol(rtl, ui.Class("box"), ui.Class("roman"), ui.Attr("start", "4"), ui.Li(ui.Text("fourth"))),
		ui.Table(rtl, ui.Tr(ui.Td(ui.Text("one")), ui.Td(ui.Text("two")), ui.Td(ui.Text("three")))),
		ui.Div(rtl, ui.Class("row"), ui.Div(ui.Style("width: 40px")), ui.Div(ui.Style("width: 80px")),
			ui.Div(ui.Style("flex: 1"))),
		ui.Div(rtl, ui.Class("grid"), ui.Div(), ui.Div(), ui.Div()),
	)
	pages.Run(d, 400, 400)
}
