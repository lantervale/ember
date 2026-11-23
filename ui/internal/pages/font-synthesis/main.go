// Faces made up against Chrome's: a family with one regular face drawn
// bold, italic and both at 12, 16, 24 and 40 px, as Chrome makes them
// up, beside the same text with font-synthesis: none, a face scaled by
// size-adjust and one unkerned by font-feature-settings.
package main

import (
	"fmt"
	"testing/fstest"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
@font-face { font-family: Plain; src: url(plain.ttf); }
@font-face { font-family: Scaled; src: url(plain.ttf); size-adjust: 130%; }
@font-face { font-family: Unkerned; src: url(plain.ttf); font-feature-settings: "kern" 0; }
html { background-color: #fff; }
body { padding: 8px; font-family: Plain; color: #000; }
div { white-space: nowrap; }
.b { font-weight: bold; }
.i { font-style: italic; }
.none { font-synthesis: none; }
.scaled { font-family: Scaled; }
.unkerned { font-family: Unkerned; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"faces.css": {Data: []byte(sheet)},
		"plain.ttf": {Data: goregular.TTF},
	}, "faces.css"))
	for _, size := range []int{12, 16, 24, 40} {
		d.Body().Append(ui.Div(ui.Style(fmt.Sprintf("font-size: %dpx", size)),
			ui.Span(ui.Text("Regular ")), ui.Span(ui.Class("b"), ui.Text("Bold ")),
			ui.Span(ui.Class("i"), ui.Text("Italic ")), ui.Span(ui.Class("b", "i"), ui.Text("Both"))))
	}
	d.Body().Append(
		ui.Div(ui.Class("b", "i", "none"), ui.Style("font-size: 20px"), ui.Text("Neither made up: AVATAR")),
		ui.Div(ui.Class("scaled"), ui.Style("font-size: 20px"), ui.Text("Scaled by 130%: AVATAR")),
		ui.Div(ui.Class("unkerned"), ui.Style("font-size: 20px"), ui.Text("Unkerned: AVATAR")),
	)
	pages.Run(d, 520, 300)
}
