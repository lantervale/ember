// Scroll containers whose bars their sheets style through
// ::-webkit-scrollbar and its parts, as real pages do: thicknesses, round
// thumbs, tracks with margins and borders, track pieces, a corner, a bar
// hidden, a thumb's least length, and a stable gutter; some scrolled.
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
.s { display: inline-block; vertical-align: top; width: 100px; height: 80px; margin: 4px; overflow: scroll;
  background-color: #3a332c; }
.s > div { width: 200px; height: 90px; background-color: #ad9773; border-bottom: 40px solid #6b8f5a; }
.round::-webkit-scrollbar { width: 12px; height: 8px; background-color: #5a1f1f; }
.round::-webkit-scrollbar-thumb { background-color: #4f7cff; border-radius: 4px; }
.track::-webkit-scrollbar { width: 10px; height: 10px; }
.track::-webkit-scrollbar-track { background-color: #2f8f4f; border-radius: 5px; }
.track::-webkit-scrollbar-thumb { background-color: #111; border: 2px solid #2f8f4f; }
.track::-webkit-scrollbar-corner { background-color: #c040c0; }
.pieces::-webkit-scrollbar { width: 12px; height: 12px; background-color: #ccc; }
.pieces::-webkit-scrollbar-thumb { background-color: #224; }
.pieces::-webkit-scrollbar-track-piece { background-color: #0c0; }
.pieces::-webkit-scrollbar-track-piece:end { background-color: #c0c; }
.least::-webkit-scrollbar { width: 12px; height: 12px; background-color: #ccc; }
.least::-webkit-scrollbar-thumb { background-color: #00f; min-height: 30px; }
.least > div { height: 3000px; }
.inset::-webkit-scrollbar { width: 14px; height: 14px; background-color: #ccc; }
.inset::-webkit-scrollbar-thumb { background-color: #00f; border: 3px solid transparent; background-clip: padding-box;
  border-radius: 7px; }
.inset::-webkit-scrollbar-thumb:horizontal { background-color: #f60; }
.margins::-webkit-scrollbar { width: 12px; height: 12px; background-color: #ccc; }
.margins::-webkit-scrollbar-thumb { background-color: #00f; }
.margins::-webkit-scrollbar-track { background-color: #fc0; margin: 6px 4px; border: 2px solid #000; }
.none::-webkit-scrollbar { display: none; }
.stable { overflow: auto; scrollbar-gutter: stable; }
.stable > div { width: 50px; height: 40px; border-bottom: 0; }
.stable::-webkit-scrollbar { width: 12px; background-color: #ccc; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	scrolled := map[string]geom.Vec2{"track": geom.V2(30, 30), "pieces": geom.V2(100, 30), "margins": geom.V2(0, 260)}
	for _, c := range []string{"round", "track", "pieces", "least", "inset", "margins", "none", "stable"} {
		s := ui.Div(ui.Class("s", c), ui.Div())
		d.Body().Append(s)
		if at, ok := scrolled[c]; ok {
			dom.Data(s).Scroll = at
		}
	}
	pages.Run(d, 480, 230)
}
