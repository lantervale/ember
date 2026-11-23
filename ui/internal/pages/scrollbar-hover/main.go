// A scroll container whose styled bars change under the pointer: the
// bar, its track, a piece, the thumb and the buttons each with its own
// :hover, the thumb its :active; the tests put the mouse on one part at
// a time (TestScrollbarPartsUnderTheMouseAsChromes).
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
.s { width: 100px; height: 100px; margin: 4px; overflow: scroll; background-color: #3a332c; }
.s > div { width: 200px; height: 300px; background-color: #ad9773; border-bottom: 40px solid #6b8f5a; }
.s::-webkit-scrollbar { width: 12px; height: 12px; background-color: #5a1f1f; }
.s::-webkit-scrollbar:hover { background-color: #1f5a1f; }
.s::-webkit-scrollbar-track { margin: 0; }
.s::-webkit-scrollbar-track:hover { background-color: #303060; }
.s::-webkit-scrollbar-track-piece:start:hover { background-color: #c0c040; }
.s::-webkit-scrollbar-thumb { background-color: #4f7cff; border-radius: 4px; }
.s::-webkit-scrollbar-thumb:hover { background-color: #ff7c4f; }
.s::-webkit-scrollbar-thumb:active { background-color: #ffffff; }
.s::-webkit-scrollbar-button { display: block; width: 10px; height: 10px; background-color: #808080; }
.s::-webkit-scrollbar-button:hover { background-color: #e040e0; }
.s::-webkit-scrollbar-corner { background-color: #202020; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	s := ui.Div(ui.Class("s"), ui.Div())
	d.Body().Append(s)
	dom.Data(s).Scroll = geom.V2(100, 0)
	pages.Run(d, 200, 140)
}
