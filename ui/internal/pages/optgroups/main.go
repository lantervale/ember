// Selects whose options stand in optgroups, closed: each as wide as its
// options, those of a group counted behind four spaces, never the
// labels, showing the option chosen in a group, the first enabled one
// past a disabled group, or a placeholder. An open list is not compared:
// Chrome on macOS shows it as a native menu no screenshot takes.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #ffffff; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 12px; align-items: center; margin-bottom: 12px; }
.big { font-size: 20px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(
		ui.Div(ui.Class("row"),
			ui.Select(ui.Option(ui.Text("Fists")),
				ui.Optgroup("Blades", ui.Option(ui.Text("Sword")), ui.Option(ui.Text("Dagger"), ui.Selected()))),
			ui.Select(ui.Option(ui.Text("Sword")),
				ui.Optgroup("A label wider than its options", ui.Option(ui.Text("Wand")))),
			ui.Select(ui.Optgroup("Cursed", ui.Disabled(), ui.Option(ui.Text("Skull"))), ui.Option(ui.Text("Bow"))),
		),
		ui.Div(ui.Class("row"),
			ui.Select(ui.Class("big"), ui.Optgroup("Magic", ui.Option(ui.Text("Wand")), ui.Option(ui.Text("Staff")))),
			ui.Select(ui.Optgroup("Empty"), ui.Optgroup("Long", ui.Option(ui.Text("Bone flail of the crypt")))),
			ui.Select(ui.Placeholder("Pick one"), ui.Optgroup("Ranged", ui.Option(ui.Text("Bow")))),
		),
	)
	pages.Run(d, 520, 100)
}
