// Stroked text against Chrome's: -webkit-text-stroke at a few widths and
// sizes over filled glyphs, outlined text whose -webkit-text-fill-color is
// transparent, a fill colour apart from color, the decoration in the fill,
// the stroke with a text-shadow, and a selection whose ::selection
// strokes the selected glyphs its own way and fills them in its color,
// its -webkit-text-fill-color left unread as Chrome leaves it.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; user-select: text; }
body { margin: 0; padding: 10px; font-size: 16px; color: #222; }
p { margin: 0 0 8px; white-space: nowrap; }
::selection { background-color: #ffd54a; color: #1a1a1a; }
.thin { -webkit-text-stroke: 1px #c0392b; }
.wide { font-size: 28px; -webkit-text-stroke: 2px #1f5fa8; -webkit-text-fill-color: #ffd54a; }
.outlined { font-size: 40px; -webkit-text-stroke: 1.5px #1f5fa8; -webkit-text-fill-color: transparent; }
.mitres { font-size: 36px; -webkit-text-stroke: 4px #2b2722; color: #fff; }
.fill { color: #c0392b; -webkit-text-fill-color: #2e7d32; text-decoration: underline; }
.shadowed { font-size: 24px; -webkit-text-stroke: 1px #000; color: #ffd54a; text-shadow: 3px 3px #8a8378; }
.current { color: #3c6fb0; -webkit-text-stroke: thick; font-size: 24px; -webkit-text-fill-color: #fff; }
.sel { font-size: 24px; -webkit-text-stroke: 1px #c0392b; }
.sel::selection { background-color: #3c6fb0; color: #fff; -webkit-text-fill-color: #ffd54a;
  -webkit-text-stroke-color: #000; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"stroke.css": {Data: []byte(sheet)}}, "stroke.css"))
	sel := ui.P(ui.Class("sel"), ui.Text("Selected in the middle"))
	d.Body().Append(
		ui.P(ui.Class("thin"), ui.Text("A one pixel red stroke on dark text")),
		ui.P(ui.Class("wide"), ui.Text("Wide stroke, yellow fill")),
		ui.P(ui.Class("outlined"), ui.Text("Outlined Ego")),
		ui.P(ui.Class("mitres"), ui.Text("MAW VIA 47")),
		ui.P(ui.Class("fill"), ui.Text("Green fill and underline, red color")),
		ui.P(ui.Class("shadowed"), ui.Text("Stroked and shadowed")),
		ui.P(ui.Class("current"), ui.Text("Thick, in currentColor")),
		sel,
	)
	d.Select(ui.Position{Node: sel.Children()[0], Offset: 4}, ui.Position{Node: sel.Children()[0], Offset: 15})
	pages.Run(d, 480, 420)
}
