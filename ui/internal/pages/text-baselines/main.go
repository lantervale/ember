// Text whose lines start off the device grid against Chrome's: each
// line's baseline rounded to the nearest device pixel, its glyphs at
// their exact coverage so a block of █ shows where they stand.
package main

import (
	"fmt"
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #fff; }
body { margin: 0; -webkit-font-smoothing: antialiased; }
div { position: absolute; left: 8px; white-space: nowrap; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"baselines.css": {Data: []byte(sheet)}}, "baselines.css"))
	top := float32(8)
	for _, size := range []int{13, 16, 20} {
		for _, frac := range []float32{0, 0.25, 0.3, 0.5, 0.6, 0.75} {
			d.Body().Append(ui.Div(ui.Style(fmt.Sprintf("top: %gpx; font-size: %dpx", top+frac, size)),
				ui.Text(fmt.Sprintf("█▀▄ Handgloves %g", frac))))
			top += float32(size) + 6
		}
	}
	pages.Run(d, 300, 440)
}
