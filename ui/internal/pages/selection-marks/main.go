// A selection's marks against Chrome's: empty lines a br holds and those
// of preformatted text marked a space wide, ::selection's text-shadow in
// place of the text's own, its text-decoration over the text's, and
// currentColor in them the selection's colour. Every ::selection colour
// is the page's own, so no machine's highlight colour shows.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; user-select: text; }
body { margin: 0; padding: 12px; font-size: 16px; line-height: 22px; color: #2b2722; }
::selection { background-color: #ffd54a; color: #1a1a1a; }
p { margin: 0 0 10px; }
.pre { white-space: pre; }
.centred { text-align: center; }
.shadowed { text-shadow: 2px 2px #8a8378; font-size: 22px; line-height: 30px; }
.shadowed::selection { text-shadow: 3px 3px #3c6fb0; }
.lined::selection { text-decoration: underline #b0493c 2px; }
.through { text-decoration: line-through; }
.through::selection { text-decoration: overline dashed; color: #3c6fb0; }
.ink::selection { color: #ffffff; background-color: #3c6fb0; text-shadow: 1px 1px currentColor; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"marks.css": {Data: []byte(sheet)}}, "marks.css"))
	first := ui.P(ui.Text("Before the selection, then it begins"), ui.Br(), ui.Br(), ui.Text("after an empty line"),
		ui.Br(), ui.Br(), ui.Br(), ui.Text("and two"))
	last := ui.P(ui.Class("ink"), ui.Text("The last lines, drawn white on blue with a shadow"))
	d.Body().Append(
		first,
		ui.P(ui.Class("pre"), ui.Text("pre text\n\nwith empty lines\n\n\nbetween")),
		ui.P(ui.Class("centred"), ui.Text("centred"), ui.Br(), ui.Br(), ui.Text("lines")),
		ui.P(ui.Class("shadowed"), ui.Text("Shadowed text, its shadow blue")),
		ui.P(ui.Class("lined"), ui.Text("Underlined where selected, "), ui.Span(ui.Class("through"),
			ui.Text("over a line through")), ui.Text(" and on")),
		last,
		ui.P(ui.Text("After the selection.")),
	)
	d.Select(ui.Position{Node: first.Children()[0], Offset: 22}, ui.Position{Node: last.Children()[0], Offset: 30})
	pages.Run(d, 420, 560)
}
