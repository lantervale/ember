// Vertical writing modes against Chrome: lines of vertical-rl stacking from the right
// with their glyphs turned a quarter clockwise, vertical-lr stacking from the left,
// sideways-lr's lines running up, underlined and padded inline elements on vertical
// lines, a flex row running down a box, a horizontal block in a vertical one, and
// letters standing upright (text-orientation: upright).
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 16px; display: flex; gap: 10px; align-items: flex-start; }
p { margin: 0; }
.box { height: 200px; border: 1px solid #c8473d; padding: 4px; }
.rl { writing-mode: vertical-rl; }
.lr { writing-mode: vertical-lr; }
.up { writing-mode: sideways-lr; }
.pad { padding: 2px 4px; background-color: #f0c040; }
u { text-decoration-color: #3a6fd8; }
.row { display: flex; gap: 6px; }
.row div { inline-size: 30px; background-color: #3a6fd8; }
.row .grow { flex: 1; }
.upright { text-orientation: upright; }
.across { writing-mode: horizontal-tb; width: 90px; background-color: #e8e2d6; margin-block: 4px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"writing.css": {Data: []byte(sheet)}}, "writing.css"))
	d.Body().Append(
		ui.Div(ui.Class("box", "rl"), ui.P(ui.Text("Lines from the right, "),
			ui.Span(ui.Class("pad"), ui.Text("padded")), ui.Text(" and "), ui.U(ui.Text("underlined")),
			ui.Text(" words going down."))),
		ui.Div(ui.Class("box", "lr"), ui.P(ui.Text("Lines from the left, wrapping as they go down the box."))),
		ui.Div(ui.Class("box", "up"), ui.P(ui.Text("Sideways lines that run up from the bottom."))),
		ui.Div(ui.Class("box", "rl", "row"), ui.Div(), ui.Div(ui.Class("grow")), ui.Div()),
		ui.Div(ui.Class("box", "rl"), ui.P(ui.Text("Before")),
			ui.Div(ui.Class("across"), ui.Text("A horizontal block inside.")), ui.P(ui.Text("After"))),
		ui.Div(ui.Class("box", "rl", "upright"), ui.P(ui.Text("Upright Hi"))),
	)
	pages.Run(d, 440, 240)
}
