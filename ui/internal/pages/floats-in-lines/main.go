// Floats written after text in a paragraph against Chrome's: on the line
// they fall on when they fit beside what stands before them there, below
// it when they don't, several on a line, after a line break, and right to
// left.
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
p { margin: 0; background-color: #f4efe4; }
.l { float: left; } .r { float: right; }
.tag { padding: 2px 6px; margin: 0 6px 4px 0; background-color: #3c6fb0; color: #fff; }
.r.tag { margin: 0 0 4px 6px; background-color: #b0493c; }
.seal { width: 44px; height: 30px; margin: 2px 6px; background-color: #3c8a4f; }
.wide { width: 120px; height: 24px; background-color: #7a4fb0; }
.rtl { direction: rtl; }
`

const story = "The lantern swung over the crypt stairs, and every step below it rang hollow under the boots of " +
	"the three who went down into the dark."

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"floats.css": {Data: []byte(sheet)}}, "floats.css"))
	d.Body().Append(
		ui.P(ui.Text("The lantern "), ui.Span(ui.Class("l", "tag"), ui.Text("Crypt")), ui.Text(" swung over "+story)),
		ui.P(ui.Text("Every step "), ui.Span(ui.Class("r", "tag"), ui.Text("Stairs")), ui.Text(" rang hollow "+story)),
		ui.P(ui.Text("The lantern swung over the crypt "), ui.Span(ui.Class("l", "wide")),
			ui.Text(" stairs, "+story)),
		ui.P(ui.Text("One "), ui.Span(ui.Class("l", "seal")), ui.Text(" two "), ui.Span(ui.Class("r", "tag"),
			ui.Text("Three")), ui.Text(" "+story)),
		ui.P(ui.Text("Down the stairs"), ui.Br(), ui.Span(ui.Class("l", "seal")), ui.Text(story)),
		ui.P(ui.Text(story+" At last "), ui.Span(ui.Class("r", "tag"), ui.Text("End"))),
		ui.P(ui.Class("rtl"), ui.Text("The lantern "), ui.Span(ui.Class("r", "tag"), ui.Text("Crypt")),
			ui.Text(" swung over "+story)),
		ui.P(ui.Class("rtl"), ui.Text("The lantern swung over the crypt "), ui.Span(ui.Class("l", "wide")),
			ui.Text(" stairs, "+story)),
	)
	pages.Run(d, 560, 420)
}
