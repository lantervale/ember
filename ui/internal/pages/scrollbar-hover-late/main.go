// Two scroll containers whose bars their own :hover restyles
// (.s:hover::-webkit-scrollbar-thumb…), which Chrome applies to a part
// only once the pointer changes that part's state, the second's border
// changing with its hover too, which restyles all its parts at once; the
// tests move the mouse through them
// (TestBarsRestyledByTheirBoxsHoverAsChromes).
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #1b1714; }
body { margin: 0; padding: 8px; display: flex; }
.s { width: 100px; height: 100px; margin: 4px; overflow: scroll; background-color: #3a332c; flex: none; }
.s > div { width: 200px; height: 300px; background-color: #ad9773; border-bottom: 40px solid #6b8f5a; }
.s::-webkit-scrollbar { width: 12px; height: 12px; background-color: #5a1f1f; }
.s:hover::-webkit-scrollbar { background-color: #1f5a1f; }
.s::-webkit-scrollbar-track-piece { background-color: #303060; }
.s:hover::-webkit-scrollbar-track-piece { background-color: #c0c040; }
.s::-webkit-scrollbar-thumb { background-color: #4f7cff; }
.s:hover::-webkit-scrollbar-thumb { background-color: #ff7c4f; }
.s::-webkit-scrollbar-corner { background-color: #202020; }
.s:hover::-webkit-scrollbar-corner { background-color: #e040e0; }
.t { border: 2px solid #444; }
.t:hover { border-color: #ddd; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	d.Body().Append(ui.Div(ui.Class("s"), ui.Div()), ui.Div(ui.Class("s", "t"), ui.Div()))
	pages.Run(d, 260, 140)
}
