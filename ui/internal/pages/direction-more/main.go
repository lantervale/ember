// More of right to left against Chrome: every mirrored character of text
// shown right to left, a selection inside it, an ellipsis cutting a line
// at its left, columns and their rules from the right, a table's collapsed
// borders resolved from its start, fields, a select and its arrow, and a
// scroller started at its right end and one scrolled left.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; user-select: text; }
body { padding: 8px; color: #2b2722; font-size: 16px; }
::selection { background-color: #ffd54a; color: #1a1a1a; }
p { margin: 0 0 6px; }
.box { width: 360px; border: 1px solid #c8473d; }
.cut { width: 200px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cols { width: 360px; column-count: 3; column-gap: 16px; column-rule: 2px solid #3a6fd8; margin-bottom: 6px; }
table { border-collapse: collapse; margin-bottom: 6px; }
td { border: 1px solid #2b2722; padding: 2px 6px; }
.start { border-inline-start: 5px solid #c8473d; }
input, select { width: 160px; margin: 0 0 6px; display: block; }
.scroller { width: 200px; overflow-x: auto; border: 1px solid #2b2722; margin-bottom: 6px; }
.wide { width: 320px; height: 14px; background: linear-gradient(to right, #3a6fd8, #f0c040); }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"direction.css": {Data: []byte(sheet)}}, "direction.css"))
	rtl := ui.Attr("dir", "rtl")
	mirrored := ui.Bdo(rtl, ui.Text("a < b ≤ c « d » [e] {f}"))
	scrolled := ui.Div(rtl, ui.Class("scroller"), ui.Div(ui.Class("wide")))
	d.Body().Append(
		ui.P(ui.Class("box"), mirrored),
		ui.P(rtl, ui.Class("box", "cut"), ui.Text("A line far too long for its box, cut at its end.")),
		ui.Div(rtl, ui.Class("cols"), ui.Text("Words from the right in three columns, each one ruled from "+
			"its neighbour, the first of them on the right and the last on the left.")),
		ui.Table(rtl, ui.Tr(ui.Td(ui.Class("start"), ui.Text("one")), ui.Td(ui.Text("two")),
			ui.Td(ui.Class("start"), ui.Text("three")))),
		ui.Input(rtl, ui.Attr("value", "a value from the right")),
		ui.Select(rtl, ui.Option(ui.Text("an option"))),
		ui.Div(rtl, ui.Class("scroller"), ui.Div(ui.Class("wide"))),
		scrolled,
	)
	d.SetScroll(scrolled, geom.V2(-60, 0))
	text := mirrored.Children()[0]
	d.Select(ui.Position{Node: text, Offset: 2}, ui.Position{Node: text, Offset: 9})
	pages.Run(d, 400, 420)
}
