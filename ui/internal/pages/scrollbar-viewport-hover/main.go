// A page whose viewport scrolls, its bars styled by body's rules and
// restyled by body's :hover (body:hover::-webkit-scrollbar-thumb…),
// which Chrome applies to every part at once, where a box's bars wait
// for the pointer to change a part's state; hovering .own changes
// body's own style too. The tests move the mouse through it
// (TestViewportBarsRestyledByBodysHoverAsChromes).
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { margin: 0; height: 60px; background-color: #3a332c; }
.own { width: 60px; height: 40px; background-color: #6b8f5a; }
.page { position: absolute; top: 100px; left: 0; width: 600px; height: 800px; background-color: #ad9773; }
body:has(.own:hover) { background-color: #5a3a6b; }
body::-webkit-scrollbar { width: 12px; height: 12px; background-color: #5a1f1f; }
body:hover::-webkit-scrollbar { background-color: #1f5a1f; }
body::-webkit-scrollbar-track-piece { background-color: #303060; }
body:hover::-webkit-scrollbar-track-piece { background-color: #c0c040; }
body::-webkit-scrollbar-thumb { background-color: #4f7cff; }
body:hover::-webkit-scrollbar-thumb { background-color: #ff7c4f; }
body::-webkit-scrollbar-corner { background-color: #202020; }
body:hover::-webkit-scrollbar-corner { background-color: #e040e0; }
`

func main() {
	d := ui.NewDocument(ui.ScrollingViewport(),
		ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(ui.Div(ui.Class("own")), ui.Div(ui.Class("page")))
	pages.Run(d, 300, 220)
}
