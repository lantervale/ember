// ::first-line and ::first-letter against Chrome's, further: a first
// line's box-shadow drawn behind its background, a first line hidden by
// visibility with an element on it shown, opacity and vertical-align on
// a first line, which Chrome draws nothing by, and first letters taking
// the spaces white-space keeps before them.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 12px; line-height: 16px; color: #2b2722;
	display: grid; grid-template-columns: 262px 262px; gap: 12px 16px; }
p { margin: 0; }
.shadow::first-line { background-color: #e6dcc8; box-shadow: 3px 3px 0 #3c6fb0; }
.bare::first-line { box-shadow: 0 2px 0 #b0493c; }
.hidden::first-line { visibility: hidden; background-color: #e6dcc8; }
.hidden b { visibility: visible; color: #b0493c; }
.faded::first-line { opacity: 0.3; color: #3c8a4f; background-color: #dce8d0; }
.raised::first-line { vertical-align: 6px; color: #7a4fb0; }
.raised b { background-color: #f0e0a8; }
.pre { white-space: pre; }
.pre::first-letter { font-size: 26px; color: #fff; background-color: #3c6fb0; }
.wrap { white-space: pre-wrap; }
.wrap::first-letter { font-size: 22px; border: 2px solid #b0493c; padding: 0 3px; }
`

const story = "The lantern swung over the crypt stairs, and every step below it rang hollow under the boots of " +
	"the three who went down into the dark."

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"first.css": {Data: []byte(sheet)}}, "first.css"))
	d.Body().Append(
		ui.P(ui.Class("shadow"), ui.Text(story)),
		ui.P(ui.Class("bare"), ui.Text(story)),
		ui.P(ui.Class("hidden"), ui.Text("Hidden but "), ui.B(ui.Text("this word")), ui.Text(" "+story)),
		ui.P(ui.Class("faded"), ui.Text(story)),
		ui.P(ui.Class("raised"), ui.Text("Raised "), ui.B(ui.Text("nothing")), ui.Text(" "+story)),
		ui.P(ui.Class("pre"), ui.Text("   Lantern light\n  and its stairs")),
		ui.P(ui.Class("wrap"), ui.Text("  “Every” step below it rang hollow under the boots of the three.")),
	)
	pages.Run(d, 560, 420)
}
