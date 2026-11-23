// Selects shown as listboxes, none focused: rows of their options, the
// selected ones grey, a group's label bold before its options indented,
// a disabled option and a disabled listbox greyed, rows sized by a
// larger font, and a listbox with more options than rows.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #ffffff; }
body { padding: 12px; margin: 0; }
.row { display: flex; gap: 12px; align-items: flex-start; }
.big { font-size: 20px; }
`

func opt(text string, args ...ui.Arg) *ui.Node {
	return ui.Option(append([]ui.Arg{ui.Text(text)}, args...)...)
}

func main() {
	multiple := ui.Attr("multiple", "")
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(
		ui.Div(ui.Class("row"),
			ui.Select(multiple, opt("Sword"), opt("Bow", ui.Selected()), opt("Wand", ui.Selected()), opt("Staff")),
			ui.Select(multiple, opt("Fists"),
				ui.Optgroup("Blades", opt("Sword", ui.Selected()), opt("Dagger", ui.Disabled())),
				ui.Optgroup("Cursed", ui.Disabled(), opt("Skull"))),
			ui.Select(ui.Attr("size", "3"), opt("One"), opt("Two", ui.Selected()), opt("Three"), opt("Four"),
				opt("Five")),
			ui.Select(multiple, ui.Disabled(), opt("Locked", ui.Selected()), opt("Shut")),
			ui.Select(multiple, ui.Class("big"), opt("Aa", ui.Selected()), opt("Bb")),
		),
	)
	pages.Run(d, 520, 140)
}
