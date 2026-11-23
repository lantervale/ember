// Boxes the user may resize, their resizer unstyled: Chrome's grip in
// the corner, on a box that clips with no bars, right to left, beside
// styled bars, beside a lone styled bar, on light and dark backgrounds.
package main

import (
	"testing/fstest"

	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/pages"
)

const sheet = `
html { background-color: #f4efe6; }
body { margin: 0; padding: 8px; display: flex; flex-wrap: wrap; gap: 8px; }
.r { width: 90px; height: 60px; overflow: hidden; resize: both; background-color: #ffffff; flex: none; }
.r > div { height: 20px; background-color: #c9b48f; }
.dark { background-color: #2a2522; }
.rtl { direction: rtl; }
.v { resize: vertical; }
.styled { overflow: scroll; }
.styled > div { width: 200px; height: 200px; }
.lone { overflow: auto; }
.lone > div { height: 200px; }
.styled::-webkit-scrollbar, .lone::-webkit-scrollbar { width: 12px; height: 12px; background-color: #5a1f1f; }
.styled::-webkit-scrollbar-thumb, .lone::-webkit-scrollbar-thumb { background-color: #4f7cff; }
.big { width: 140px; height: 90px; }
`

func main() {
	d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
	for _, c := range [][]string{{}, {"dark"}, {"rtl"}, {"v"}, {"styled"}, {"lone"}, {"big", "dark"}} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"r"}, c...)...), ui.Div()))
	}
	pages.Run(d, 480, 160)
}
