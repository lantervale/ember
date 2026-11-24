//go:build chrome

package ui_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestTransformsHoldFixedAndAbsoluteBoxesAsChromes has an element moved,
// turned or scaled by transform or one of translate, rotate and scale be
// the containing block of its fixed and absolute descendants, as Chrome
// makes it, through a scroller between them and with the page scrolled.
func TestTransformsHoldFixedAndAbsoluteBoxesAsChromes(t *testing.T) {
	const sheet = `body { margin: 0; }
.t { margin: 30px 40px; width: 160px; height: 90px; padding: 5px; border: 3px solid #333; background: #fde9c8; }
.fixed { position: fixed; top: 10px; right: 15px; width: 40px; height: 20px; background: #36c; }
.abs { position: absolute; bottom: 4px; left: 6px; width: 30px; height: 10px; background: #c36; }
.list { overflow: auto; height: 60px; } .row { height: 50px; }
.tall { height: 1500px; }`
	transforms := []string{
		"transform: translate(10px, 5px)",
		"translate: 12px 7px",
		"rotate: 15deg",
		"scale: 1.2 0.8",
		"transform: rotate(-10deg) skewX(10deg)",
		"translate: none",
	}
	ids := func(i int) []string {
		return []string{fmt.Sprint("t", i), fmt.Sprint("f", i), fmt.Sprint("a", i), fmt.Sprint("lf", i), fmt.Sprint("la", i)}
	}
	for _, page := range []float32{0, 120} {
		t.Run(fmt.Sprint("the page scrolled ", page), func(t *testing.T) {
			d := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(sheet))
			var all []string
			var lists []*ui.Node
			for i, tr := range transforms {
				id := ids(i)
				all = append(all, id...)
				list := ui.Div(ui.Class("list"), ui.Div(ui.Class("row")),
					ui.Div(ui.ID(id[3]), ui.Class("fixed")), ui.Div(ui.ID(id[4]), ui.Class("abs")), ui.Div(ui.Class("row")))
				lists = append(lists, list)
				d.Body().Append(ui.Div(ui.ID(id[0]), ui.Class("t"), ui.Style(tr),
					ui.Div(ui.ID(id[1]), ui.Class("fixed")), ui.Div(ui.ID(id[2]), ui.Class("abs")), list))
			}
			d.Body().Append(ui.Div(ui.Class("tall")))
			dom.Data(d.HTML()).Scroll = geom.V2(0, page)
			for _, l := range lists {
				dom.Data(l).Scroll = geom.V2(0, 25)
			}
			size := geom.V2(500, 300)
			js, err := json.Marshal(all)
			require.NoError(t, err)
			out, err := chrome.Eval(d, size, `return `+string(js)+`.map(id => {
	const r = document.getElementById(id).getBoundingClientRect(); return [r.x, r.y, r.width, r.height]; });`)
			require.NoError(t, err)
			var want [][4]float32
			require.NoError(t, json.Unmarshal(out, &want))
			require.Len(t, want, len(all))
			d.Layout(size)
			for i, id := range all {
				got, ok := ui.TransformedBox(d, d.ByID(id))
				require.True(t, ok, id)
				w := geom.Rect{X: want[i][0], Y: want[i][1], W: want[i][2], H: want[i][3]}
				assert.InDelta(t, w.X, got.X, 0.5, "%s: ember %v, Chrome %v", id, got, w)
				assert.InDelta(t, w.Y, got.Y, 0.5, "%s: ember %v, Chrome %v", id, got, w)
				assert.InDelta(t, w.W, got.W, 0.5, "%s: ember %v, Chrome %v", id, got, w)
				assert.InDelta(t, w.H, got.H, 0.5, "%s: ember %v, Chrome %v", id, got, w)
			}
		})
	}
}

// TestTransformsGrowOverflowInLinesAndVerticalScrollersAsChromes has a
// transformed inline-block, and transformed boxes in scrollers of each
// vertical writing mode, reach as far as in Chrome's scrollers.
func TestTransformsGrowOverflowInLinesAndVerticalScrollersAsChromes(t *testing.T) {
	sheet := `body { margin: 0; } .s { width: 100px; height: 100px; overflow: auto; position: relative; }
.s > div, .s span { width: 50px; height: 50px; } span { display: inline-block; }
.v { writing-mode: vertical-rl; } .l { writing-mode: vertical-lr; } .wide { width: 200px; height: 10px; }`
	cases := []struct {
		mode, style string
		inline      bool
		text, wide  bool
	}{
		{"", "translate: 100px 80px", true, false, false},
		{"", "rotate: 45deg; margin-left: 40px", true, true, false},
		{"", "scale: 3; transform-origin: 0 0", true, false, false},
		{"v", "translate: -100px 80px", false, false, false},
		{"v", "position: relative; left: -100px; top: 80px", false, false, false},
		{"v", "margin-right: 100px; margin-top: 80px", false, false, false},
		{"v", "translate: 100px 80px", false, false, false},
		{"v", "rotate: 45deg; margin-top: 80px", false, false, false},
		{"v", "transform: scale(3)", false, false, false},
		{"v", "translate: -100px 80px; overflow: hidden", false, false, true},
		{"v", "translate: -60px 70px", true, false, false},
		{"l", "translate: 100px 80px", false, false, false},
		{"l", "translate: -100px -80px", false, false, false},
		{"l", "rotate: 30deg; scale: 2", false, false, false},
	}
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range cases {
		child := ui.Div(ui.Style(c.style))
		if c.inline {
			child = ui.Span(ui.Style(c.style))
		}
		if c.wide {
			child.Append(ui.Div(ui.Class("wide")))
		}
		s := ui.Div(ui.Class("s", c.mode))
		if c.text {
			s.Append(ui.Span(ui.Style("display: inline; width: auto; height: auto"), ui.Text("text ")))
		}
		s.Append(child)
		d.Body().Append(s)
	}
	size := geom.V2(500, 1400)
	out, err := chrome.Eval(d, size, `return [...document.querySelectorAll(".s")].map(e =>
	[e.scrollWidth, e.scrollHeight, e.clientWidth, e.clientHeight]);`)
	require.NoError(t, err)
	var want [][4]float32
	require.NoError(t, json.Unmarshal(out, &want))
	d.Layout(size)
	require.Len(t, want, len(cases))
	for i, s := range d.Body().Children() {
		got := ui.ScrollSize(d, s)
		got = geom.V2(max(got.X, want[i][2]), max(got.Y, want[i][3]))
		c := cases[i]
		assert.InDelta(t, want[i][0], got.X, 1, "%s %s: scrollWidth, ember %v, Chrome %v", c.mode, c.style, got, want[i])
		assert.InDelta(t, want[i][1], got.Y, 1, "%s %s: scrollHeight, ember %v, Chrome %v", c.mode, c.style, got, want[i])
	}
}

// TestTheRootsTransformGrowsTheViewportsOverflowAsChromes moves, turns
// and scales the root, and the body, and has the viewport scroll as far
// as Chrome's.
func TestTheRootsTransformGrowsTheViewportsOverflowAsChromes(t *testing.T) {
	for _, c := range []string{
		"html { translate: 300px 200px }",
		"html { transform: rotate(30deg) }",
		"html { scale: 2; transform-origin: 0 0 }",
		"html { translate: -100px -50px }",
		"body { translate: 300px 200px }",
		"html { translate: 100px; overflow: hidden }",
	} {
		t.Run(c, func(t *testing.T) {
			d := ui.NewDocument(ui.ScrollingViewport(), ui.CSS("body { margin: 0; } .b { width: 400px; height: 250px; } "+c))
			d.Body().Append(ui.Div(ui.Class("b")))
			size := geom.V2(500, 300)
			out, err := chrome.Eval(d, size, `const e = document.scrollingElement; return [e.scrollWidth, e.scrollHeight];`)
			require.NoError(t, err)
			var want [2]float32
			require.NoError(t, json.Unmarshal(out, &want))
			d.Layout(size)
			got := ui.ScrollSize(d, d.HTML())
			got = geom.V2(max(got.X, size.X), max(got.Y, size.Y))
			assert.InDelta(t, want[0], got.X, 1, "scrollWidth, ember %v, Chrome %v", got, want)
			assert.InDelta(t, want[1], got.Y, 1, "scrollHeight, ember %v, Chrome %v", got, want)
		})
	}
}
