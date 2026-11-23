// -webkit-mask-composite's compositing operators against Chrome's: two
// mask layers, a disc over a band, joined by each operator; and
// -webkit-mask with a position of three values.
package main

import (
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; }
body > div { display: grid; grid-template-columns: repeat(6, 64px); gap: 10px; }
div > div { height: 64px; background: linear-gradient(135deg, #b0493c, #3c6fb0);
  -webkit-mask-image: radial-gradient(circle at 40% 40%, #000 0 18px, rgba(0, 0, 0, 0.5) 18px 24px, transparent 24px),
    linear-gradient(transparent 20%, #000 20% 60%, rgba(0, 0, 0, 0.4) 60% 80%, transparent 80%); }
.pos { -webkit-mask: radial-gradient(circle, #000 0 12px, transparent 12px) right 6px top / 32px 32px no-repeat; }
.pos2 { -webkit-mask: radial-gradient(circle, #000 0 12px, transparent 12px) center bottom 4px / 32px 32px no-repeat; }
`

func main() {
	d := ui.NewDocument(ui.CSS(sheet), ui.BrowserSheets())
	grid := ui.Div()
	for _, op := range []string{"source-over", "source-in", "source-out", "source-atop", "destination-over",
		"destination-in", "destination-out", "destination-atop", "xor", "copy", "clear", "plus-lighter"} {
		grid.Append(ui.Div(ui.Style("-webkit-mask-composite: " + op)))
	}
	grid.Append(ui.Div(ui.Class("pos")), ui.Div(ui.Class("pos2")))
	d.Body().Append(grid)
	pages.Run(d, 460, 260)
}
