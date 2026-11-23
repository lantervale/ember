// A focused field's selection against Chrome's: its ::selection's
// background-color behind the characters selected and its color on them,
// and no caret beside them.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; padding: 12px; font-size: 16px; color: #2b2722; }
input { width: 220px; outline: none; caret-color: #c8473d; }
input::selection { background-color: #1a1a1a; color: #ffd54a; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"field.css": {Data: []byte(sheet)}}, "field.css"))
	field := ui.Input(ui.Attr("value", "the selected words in yellow"), ui.Attr("autofocus", ""))
	d.Body().Append(field)
	d.SetSelectionRange(field, 4, 18)
	pages.Run(d, 320, 60)
}
