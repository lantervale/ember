// ::first-letter and ::first-line against Chrome's: a floated drop cap,
// an initial in a box of its own colours, quotes taken with the letter,
// a letter from ::before, and first lines in their own colour, size,
// weight, case, spacing and underline, their background behind them.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 10px; font-size: 12px; line-height: 16px; color: #2b2722;
	display: grid; grid-template-columns: 262px 262px; gap: 10px 16px; }
p { margin: 0; }
.drop::first-letter { float: left; font-size: 46px; line-height: 38px; margin: 2px 6px 0 0; color: #b0493c;
	font-weight: 700; }
.boxed::first-letter { font-size: 22px; padding: 1px 5px; margin-right: 3px; color: #fff; background-color: #3c6fb0;
	border: 2px solid #23466f; border-radius: 4px; }
.quoted::first-letter { font-size: 24px; color: #3c8a4f; }
.told::before { content: "Once "; color: #7a4fb0; }
.told::first-letter { font-size: 26px; text-decoration: underline; }
.lead::first-line { color: #b0493c; font-weight: 700; letter-spacing: 1px; }
.loud::first-line { font-size: 17px; text-transform: uppercase; color: #3c6fb0; }
.lined::first-line { text-decoration: underline; word-spacing: 4px; }
.shaded::first-line { background-color: #e6dcc8; color: #23466f; }
.both::first-line { color: #3c8a4f; font-style: italic; }
.both::first-letter { float: left; font-size: 40px; line-height: 34px; margin-right: 4px; color: #7a4fb0; }
.both em { color: #b0493c; }
`

const story = "The lantern swung over the crypt stairs, and every step below it rang hollow under the boots of " +
	"the three who went down into the dark."

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"first.css": {Data: []byte(sheet)}}, "first.css"))
	d.Body().Append(
		ui.P(ui.Class("drop"), ui.Text(story)),
		ui.P(ui.Class("boxed"), ui.Text(story)),
		ui.P(ui.Class("quoted"), ui.Text("“Well,” said the warden, “go on then, and mind the third step.”")),
		ui.P(ui.Class("told"), ui.Text("upon a time there was a crypt beneath the hill, and a lantern in it.")),
		ui.P(ui.Class("lead"), ui.Text(story)),
		ui.P(ui.Class("loud"), ui.Text(story)),
		ui.P(ui.Class("lined"), ui.Text(story)),
		ui.P(ui.Class("shaded"), ui.Text(story)),
		ui.P(ui.Class("both"), ui.Text("Every "), ui.Em(ui.Text("lantern")), ui.Text(" "+story)),
	)
	pages.Run(d, 560, 420)
}
