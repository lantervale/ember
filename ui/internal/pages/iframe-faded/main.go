// Iframes faded as Chrome fades them: under an ancestor of opacity 0.5,
// with an opacity of their own, and under two faded ancestors, each over
// a blue backdrop that shows through.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; display: flex; gap: 12px; }
.cell { padding: 6px; background-color: #2b6cb0; }
.half { opacity: 0.5; }
.more { opacity: 0.6; }
iframe { display: block; width: 140px; height: 90px; }
`

const inner = `
html { background-color: #ffffff; }
body { margin: 8px; }
.bar { height: 24px; margin-bottom: 8px; background-color: #e03c31; }
.bar + .bar { background-color: #1b1714; width: 60%; }
`

func framed() *ui.Document {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(inner)}}, "page.css"))
	d.Body().Append(ui.Div(ui.Class("bar")), ui.Div(ui.Class("bar")))
	return d
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(
		ui.Div(ui.Class("cell"), ui.Div(ui.Class("half"), ui.Iframe(framed()))),
		ui.Div(ui.Class("cell"), ui.Iframe(framed(), ui.Class("half"))),
		ui.Div(ui.Class("cell"), ui.Div(ui.Class("half"), ui.Div(ui.Class("more"), ui.Iframe(framed())))),
		ui.Div(ui.Class("cell"), ui.Iframe(framed())))
	pages.Run(d, 640, 140)
}
