// Preserved spaces against Chrome's: break-spaces lines whose spaces take
// room at their end, under an underline and spans' backgrounds, aligned
// right, centred and justified, beside pre-wrap's hanging spaces; carriage
// returns kept by pre; break-all round punctuation and between elements.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #2b2722; font-size: 16px; }
p { margin: 0 0 8px; width: 170px; background-color: #f3efe6; }
.bs { white-space: break-spaces; }
.pw { white-space: pre-wrap; }
.under { text-decoration: underline #3a6fd8; }
.right { text-align: right; }
.center { text-align: center; }
.just { text-align: justify; }
.mark { background-color: #e8c66a; }
.pre { white-space: pre; width: auto; }
.all { word-break: break-all; width: 120px; }
b { color: #c8473d; }
`

const kept = "The lantern   swings over  the gate,    all night long. "

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"break-spaces.css": {Data: []byte(sheet)}}, "break-spaces.css"))
	d.Body().Append(
		ui.P(ui.Class("bs", "under"), ui.Text(kept)),
		ui.P(ui.Class("pw", "under"), ui.Text(kept)),
		ui.P(ui.Class("bs", "right"), ui.Text(kept)),
		ui.P(ui.Class("bs", "center"), ui.Text(kept)),
		ui.P(ui.Class("bs", "just"), ui.Text(kept)),
		ui.P(ui.Class("bs"), ui.Text("Brass "), ui.Span(ui.Class("mark"), ui.Text("lanterns   ")),
			ui.Text("and "), ui.Span(ui.Class("mark", "under"), ui.Text("  iron ones  ")), ui.Text("glow.")),
		ui.P(ui.Class("pre"), ui.Text("Kept\rreturns\r\rdraw nothing.")),
		ui.P(ui.Class("all"), ui.Text("Open assets/(dungeon_walls).tsj, 12.5% off “now”! "),
			ui.B(ui.Text("lantern")), ui.Text("making")),
	)
	pages.Run(d, 420, 520)
}
