// Scroll containers whose scrollbar-gutter keeps room for a bar: stable
// both-edges, at each edge, either direction and with the bar showing or
// not; content shown under a stable gutter no bar stands in, and not
// under one where it does; and bars scrollbar-color turns into the
// system's, which lie over the content, scrolled or not.
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
.s { display: inline-block; vertical-align: top; width: 100px; height: 80px; margin: 4px; overflow: auto;
  background-color: #3a332c; position: relative; }
.s > div { height: 30px; background-color: #ad9773; border-left: 6px solid #6b8f5a; border-right: 6px solid #c86b3c; }
.s > .under { position: absolute; top: 40px; left: -10px; right: -10px; height: 20px; background-color: #4f7cff; }
.s::-webkit-scrollbar { width: 12px; height: 12px; background-color: #5a1f1f; }
.s::-webkit-scrollbar-thumb { background-color: #e0d4c0; border-radius: 6px; }
.both { scrollbar-gutter: stable both-edges; }
.stable { scrollbar-gutter: stable; overflow-x: hidden; }
.stable > div { width: 130px; }
.tall > div:first-child { height: 200px; }
.scroll { overflow-y: scroll; }
.rtl { direction: rtl; }
.hidden { overflow: hidden; }
.c { scrollbar-color: #e05050 #303060; }
.c > div { width: 200px; height: 300px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	scrolled := map[int]geom.Vec2{9: geom.V2(40, 100)}
	const both, stable, tall = "both", "stable", "tall"
	for i, c := range [][]string{{both}, {both, tall}, {both, "rtl"}, {both, "rtl", tall},
		{stable}, {stable, tall}, {stable, "scroll"}, {stable, "hidden"}, {"c"}, {"c"}} {
		s := ui.Div(ui.Class(append([]string{"s"}, c...)...), ui.Div(), ui.Div(ui.Class("under")))
		d.Body().Append(s)
		if at, ok := scrolled[i]; ok {
			dom.Data(s).Scroll = at
		}
	}
	pages.Run(d, 560, 210)
}
