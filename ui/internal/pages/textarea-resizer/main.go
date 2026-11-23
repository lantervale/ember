// Textareas of a page read as a browser's (BrowserSheets) that no sheet
// gives a resize: each takes Chrome's resize: auto and draws its grip,
// right to left, scrolled beside its bar, on a dark field; one whose
// sheet says resize: none draws none.
package main

import (
	"strings"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4efe6; }
body { margin: 0; padding: 8px; display: flex; flex-wrap: wrap; gap: 8px; align-items: flex-start; }
textarea { font: 14px/18px Go; width: 120px; height: 60px; margin: 0; color: #2b2722; }
.dark { background-color: #2a2522; color: #f4efe6; }
.none { resize: none; }
`

func main() {
	d := ui.NewDocument(ui.BrowserSheets(), ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(
		ui.Textarea(ui.Text("a page's textarea")),
		ui.Textarea(ui.Attr("dir", "rtl"), ui.Text("right to left")),
		ui.Textarea(ui.Text(strings.Repeat("scrolled\n", 8))),
		ui.Textarea(ui.Class("dark"), ui.Text("dark")),
		ui.Textarea(ui.Class("none"), ui.Text("no grip")),
	)
	pages.Run(d, 420, 170)
}
