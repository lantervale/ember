// Text at sizes between two whole ones against Chrome's: each drawn from
// glyphs rasterized at its own size, beside whole sizes, in the regular,
// bold and mono faces, with spaces Go lacks.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { padding: 8px; color: #1d1b18; }
p { margin: 0 0 4px; white-space: nowrap; }
.mono { font-family: Go Mono; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"sizes.css": {Data: []byte(sheet)}}, "sizes.css"))
	for _, size := range []string{"13px", "13.44px", "13.3333px", "10.5625px", "14.4px", "17.6px", "21.75px"} {
		d.Body().Append(ui.P(ui.Style("font-size: "+size),
			ui.Text(fmt.Sprintf("%s Handgloves quick jumps ", size)), ui.B(ui.Text("bold")),
			ui.Span(ui.Class("mono"), ui.Text(" code()"))))
	}
	d.Body().Append(ui.P(ui.Style("font-size: 15px"), ui.Text("em\u2003thin\u2009en\u2002six\u2006hair\u200aend")))
	pages.Run(d, 420, 240)
}
