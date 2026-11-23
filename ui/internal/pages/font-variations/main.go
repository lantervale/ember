// A variable font's instances against Chrome's: one file drawn at
// weights from 100 to 900 and widths from 50% to 150%, by font-weight,
// font-stretch and font-variation-settings, at several sizes, and a face
// fixed at 400 made bold as Chrome makes it up.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/lantervale/ember/internal/varfont/varfonttest"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
@font-face { font-family: Weighty; src: url(weighty.ttf); }
@font-face { font-family: Fixed; src: url(weighty.ttf); font-weight: 400; }
html { background-color: #fff; }
body { padding: 8px; font-family: Weighty; color: #000; }
div { white-space: nowrap; }
.fixed { font-family: Fixed; font-weight: 900; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{
		"faces.css":   {Data: []byte(sheet)},
		"weighty.ttf": {Data: varfonttest.Weighty()},
	}, "faces.css"))
	for _, size := range []int{13, 20, 32} {
		row := ui.Div(ui.Style(fmt.Sprintf("font-size: %dpx", size)))
		for _, w := range []int{100, 400, 650, 900} {
			row.Append(ui.Span(ui.Style(fmt.Sprintf("font-weight: %d", w)), ui.Text(fmt.Sprintf("Aé%d ", w))))
		}
		d.Body().Append(row)
	}
	for _, s := range []string{"font-stretch: 50%", "font-stretch: 150%", `font-variation-settings: "wght" 750, "wdth" 70`} {
		d.Body().Append(ui.Div(ui.Style("font-size: 22px; "+s), ui.Text("Handgloves AVOH")))
	}
	d.Body().Append(ui.Div(ui.Class("fixed"), ui.Style("font-size: 22px"), ui.Text("Fixed at 400, made bold")))
	pages.Run(d, 520, 320)
}
