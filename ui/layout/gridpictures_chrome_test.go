//go:build chrome

package layout_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func squarePNG(side int) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, side, side))); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// The cases of WPT's css/css-grid/grid-items/grid-minimum-size-grid-items-021.html: a
// picture of 50×50 or 500×500 alone or over a 100×25 item in an auto track, the grid
// shrunk to fit, 200px wide or constrained to 10×10.
func TestPicturesSizingAutoGridTracksMatchChrome(t *testing.T) {
	const sheet = `
.grid { display: inline-grid; grid-template-rows: auto; grid-template-columns: auto; font: 25px/1 sans-serif }
.rows2 { grid-template-rows: auto auto }
.constrained { width: 10px; height: 10px }
.item { background: black; width: 100px; height: 25px }
.start { justify-content: start }
.w200 { width: 200px }
.full { width: 100% }`
	grids := [][2][]string{
		{{"grid"}, {"w200"}},
		{{"grid", "constrained"}, {"w200"}},
		{{"grid", "w200"}, {"full"}},
		{{"grid", "w200", "constrained"}, {"full"}},
		{{"grid", "w200", "start"}, {"full"}},
		{{"grid", "w200", "constrained", "start"}, {"full"}},
	}
	cases := map[string][]*ui.Node{}
	for _, pic := range []string{"small.png", "big.png"} {
		for _, rows := range []bool{false, true} {
			var nodes []*ui.Node
			for _, g := range grids {
				classes := g[0]
				var kids []*ui.Node
				kids = append(kids, ui.Img(ui.Attr("src", pic), ui.Class(g[1]...)))
				if rows {
					classes = append(append([]string{}, classes...), "rows2")
					kids = append(kids, ui.Span(ui.Class("item")))
				}
				nodes = append(nodes, ui.Div(append([]ui.Arg{ui.Class(classes...)}, toArgs(kids)...)...))
			}
			name := pic + " alone"
			if rows {
				name = pic + " over an item"
			}
			cases[name] = nodes
		}
	}
	for name, nodes := range cases {
		t.Run(name, func(t *testing.T) {
			d := ui.NewDocument(ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)},
				"small.png": {Data: squarePNG(50)}, "big.png": {Data: squarePNG(500)}}, "page.css"))
			d.Body().Append(nodes...)
			r := chrome.Match(t, d, geom.V2(1400, 1200))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

func toArgs(nodes []*ui.Node) []ui.Arg {
	args := make([]ui.Arg, len(nodes))
	for i, n := range nodes {
		args[i] = n
	}
	return args
}
