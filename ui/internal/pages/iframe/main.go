// Iframes as Chrome draws them: one in its default 300 by 150 box and
// inset border, one sized by CSS with frameborder="0", each showing a
// Document of its own laid out in that box, clipped to it, between the
// words of the line it stands on.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4ede4; }
body { padding: 12px; margin: 0; font-size: 16px; }
.bare { width: 160px; height: 90px; margin-left: 12px; }
.over { position: absolute; left: 250px; top: 90px; width: 120px; height: 40px; background-color: #2b6cb0; }
`

const inner = `
html { background-color: #ffffff; }
body { margin: 8px; font-size: 14px; }
p { margin: 0; }
.bar { height: 24px; margin-top: 8px; background-color: #e03c31; }
.wide { width: 60%; }
@media (max-width: 200px) { html { background-color: #fefcbf; } }
`

func framed(text string) *ui.Document {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(inner)}}, "page.css"))
	d.Body().Append(ui.P(ui.Text(text)), ui.Div(ui.Class("bar"), ui.Class("wide")))
	return d
}

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(
		ui.P(ui.Text("A page, then "),
			ui.Iframe(framed("A Document of its own, laid out in the frame's box."), ui.Attr("height", "120")),
			ui.Iframe(framed("A narrow one."), ui.Class("bare"), ui.Attr("frameborder", "0")),
			ui.Text(" its words.")),
		ui.Div(ui.Class("over")))
	pages.Run(d, 600, 200)
}
