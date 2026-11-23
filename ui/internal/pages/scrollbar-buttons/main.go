// Scroll containers whose styled bars have buttons, two at each end, of
// set, auto and percentage lengths, one way or both, some scrolled or
// right to left; buttons too long for their bar; and resizers styled by
// ::-webkit-resizer, in the corner of two bars and beside one alone.
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
.s { display: inline-block; vertical-align: top; width: 100px; height: 100px; margin: 4px; overflow: scroll;
  background-color: #3a332c; }
.s > div { width: 300px; height: 300px; background-color: #ad9773; border-bottom: 40px solid #6b8f5a; }
.s::-webkit-scrollbar { width: 12px; height: 12px; background-color: #5a1f1f; }
.s::-webkit-scrollbar-thumb { background-color: #4f7cff; border-radius: 4px; }
.s::-webkit-scrollbar-track-piece:start { background-color: #2f8f4f; }
.s::-webkit-scrollbar-track-piece:end { background-color: #8f8f2f; }
.s::-webkit-scrollbar-corner { background-color: #c040c0; }
.s::-webkit-scrollbar-button { display: block; width: 10px; height: 10px; }
.s::-webkit-scrollbar-button:start:decrement { background-color: #e04040; }
.s::-webkit-scrollbar-button:start:increment { background-color: #e040e0; }
.s::-webkit-scrollbar-button:end:decrement { background-color: #40e0e0; }
.s::-webkit-scrollbar-button:end:increment { background-color: #e0e0e0; border: 2px solid #222; }
.wide::-webkit-scrollbar-button:horizontal:start:decrement { width: 30px; }
.tall::-webkit-scrollbar-button:vertical { height: 20%; }
.auto::-webkit-scrollbar-button { width: auto; height: auto; }
.thick::-webkit-scrollbar { width: 20px; }
.long::-webkit-scrollbar-button:vertical { height: 50px; }
.y { overflow-x: hidden; }
.rtl { direction: rtl; }
.r { resize: both; }
.r::-webkit-resizer { background-color: #f0a030; border-radius: 3px; }
.r::-webkit-scrollbar-button { display: none; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	scrolled := map[int]geom.Vec2{1: geom.V2(100, 120), 5: geom.V2(0, 260)}
	for i, c := range [][]string{{}, {}, {"wide"}, {"tall"}, {"auto", "thick"}, {"y"}, {"long"}, {"rtl"},
		{"r"}, {"r", "y"}, {"r", "y", "rtl"}} {
		s := ui.Div(ui.Class(append([]string{"s"}, c...)...), ui.Div())
		d.Body().Append(s)
		if at, ok := scrolled[i]; ok {
			dom.Data(s).Scroll = at
		}
	}
	pages.Run(d, 560, 250)
}
