// Font smoothing against Chrome's: -webkit-font-smoothing: antialiased
// draws glyphs at their exact coverage, solid blocks and letters at 13,
// 16, 24 and 25 px, dark on light and light on dark, beside the default
// and the unsmoothed text.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; }
.case { padding: 4px 10px; width: 380px; }
.dark { background-color: #fff; color: #000; }
.light { background-color: #1e1e1e; color: #fff; }
.exact { -webkit-font-smoothing: antialiased; }
.sub { -webkit-font-smoothing: subpixel-antialiased; }
.pixel { -webkit-font-smoothing: none; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"smoothing.css": {Data: []byte(sheet)}}, "smoothing.css"))
	exact := ui.Div(ui.Class("exact"))
	for _, tone := range []string{"dark", "light"} {
		for _, size := range []int{13, 16, 24, 25} {
			exact.Append(ui.Div(ui.Class("case", tone), ui.Style(fmt.Sprintf("font-size: %dpx", size)),
				ui.Text("█ █▌■ Handgloves "), ui.Span(ui.Style("font-weight: bold"), ui.Text("bold"))))
		}
	}
	exact.Append(
		ui.Div(ui.Class("case", "dark", "sub"), ui.Text("█ ■ darkened again in a child")),
		ui.Div(ui.Class("case", "dark", "pixel"), ui.Text("█ ■ unsmoothed in a child")),
	)
	d.Body().Append(exact, ui.Div(ui.Class("case", "dark"), ui.Text("█ ■ the default, darkened")))
	pages.Run(d, 420, 520)
}
