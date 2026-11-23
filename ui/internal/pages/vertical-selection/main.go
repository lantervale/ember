// A selection drawn on vertical lines against Chrome's: down the turned Latin of
// vertical-rl and vertical-lr, up the lines of sideways-lr, over the symbols UAX #50
// stands upright, from inside one line across the next ones and into the next
// paragraphs, the ::selection's colours the page's own.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; user-select: text; }
body { margin: 0; padding: 12px; font-size: 16px; line-height: 22px; color: #2b2722; display: flex; gap: 14px;
	align-items: flex-start; }
::selection { background-color: #ffd54a; color: #1a1a1a; }
p { margin: 0; block-size: 92px; inline-size: 190px; border: 1px solid #c8473d; }
.rl { writing-mode: vertical-rl; }
.lr { writing-mode: vertical-lr; }
.slr { writing-mode: sideways-lr; }
.ink::selection { background-color: #3c6fb0; color: #ffffff; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"vertical.css": {Data: []byte(sheet)}}, "vertical.css"))
	first := ui.P(ui.Class("rl"), ui.Text("Lantern light over the crypt stairs, © § ± signs standing"))
	last := ui.P(ui.Class("slr"), ui.Text("Every step rang hollow below the hall"))
	d.Body().Append(
		first,
		ui.P(ui.Class("lr", "ink"), ui.Text("Left to right the blocks go on, ± × in between")),
		last,
		ui.P(ui.Class("rl"), ui.Text("After the selection.")),
	)
	d.Select(ui.Position{Node: first.Children()[0], Offset: 9}, ui.Position{Node: last.Children()[0], Offset: 24})
	pages.Run(d, 420, 240)
}
