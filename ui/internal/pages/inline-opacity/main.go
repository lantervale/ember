// Faded inline elements holding blocks: a span at half opacity whose
// lines and the bordered block between them fade together, and a faded
// span inside another, its block faded by both. The blocks draw nothing
// over what they draw, which ui fades apart where Chrome fades it whole.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 14px; line-height: 20px; color: #2b2722; }
.f { opacity: 0.5; background-color: #f6dfdf; }
.b { margin: 6px 0; padding: 6px; color: #3c5a8a; border: 3px solid #b0493c; }
.g { opacity: 0.4; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"inline-opacity.css": {Data: []byte(sheet)}}, "inline-opacity.css"))
	d.Body().Append(
		ui.Div(ui.Text("Before "), ui.Span(ui.Class("f"), ui.Text("a faded span"),
			ui.Div(ui.Class("b"), ui.Text("a block inside it")), ui.Text("and its last line")), ui.Text(" after.")),
		ui.Div(ui.Span(ui.Class("f"), ui.Text("outer "), ui.Span(ui.Class("g"), ui.Text("inner"),
			ui.Div(ui.Class("b"), ui.Text("faded twice")), ui.Text("inner end")), ui.Text(" outer end"))),
	)
	pages.Run(d, 420, 240)
}
