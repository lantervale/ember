// A page wider and taller than its window, whose viewport scrolls, its
// bars styled by body's ::-webkit-scrollbar rules over the root's, as
// Chrome takes them: thicknesses, a round thumb, a track with margins,
// a button at its end and the corner; scrolled part of the way. Its
// vertical thumb turns under the pointer
// (TestScrollbarPartsUnderTheMouseAsChromes).
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { margin: 0; padding: 8px; }
.page { width: 600px; height: 800px; background-color: #3a332c; border: 6px solid #ad9773; }
html::-webkit-scrollbar { width: 30px; height: 30px; background-color: #ff0000; }
body::-webkit-scrollbar { width: 14px; height: 10px; background-color: #5a1f1f; }
body::-webkit-scrollbar-thumb { background-color: #4f7cff; border-radius: 5px; }
body::-webkit-scrollbar-thumb:horizontal { background-color: #e0d4c0; }
body::-webkit-scrollbar-thumb:vertical:hover { background-color: #ff7c4f; }
body::-webkit-scrollbar-track:vertical { margin: 6px 0; background-color: #2f8f4f; }
body::-webkit-scrollbar-button:vertical:end:increment { display: block; height: 12px; background-color: #e040e0; }
body::-webkit-scrollbar-corner { background-color: #40e0e0; }
`

func main() {
	d := ui.NewDocument(ui.ScrollingViewport(),
		ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(ui.Div(ui.Class("page")))
	dom.Data(d.HTML()).Scroll = geom.V2(120, 200)
	pages.Run(d, 320, 260)
}
