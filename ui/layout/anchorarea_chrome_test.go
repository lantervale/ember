//go:build chrome

package layout_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestPositionAreaAndFallbacksMatchChrome(t *testing.T) {
	const page = `body { margin: 0; position: relative; height: 560px }
.a { position: absolute; left: 300px; top: 200px; width: 80px; height: 40px; background: #ccc; anchor-name: --a }
.p { position: absolute; position-anchor: --a; background: #c00 }
.s { width: 50px; height: 30px }
@position-try --below { top: anchor(bottom); left: anchor(left); bottom: auto; right: auto }
@position-try --wide { width: 300px; position-area: top }`
	el := func(class, style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class(strings.Fields(class)...), ui.Style(style)}, kids...)...)
	}
	anchor := func(style string) *ui.Node { return el("a", style) }
	areas := func(values ...string) *ui.Document {
		nodes := []*ui.Node{anchor("")}
		for _, v := range values {
			nodes = append(nodes, el("p s", "position-area: "+v))
		}
		return styled(page, nodes...)
	}
	cases := map[string]*ui.Document{
		"position-area one keyword": areas("top", "bottom", "left", "right", "center", "span-all", "start", "end"),
		"position-area two keywords": areas("top left", "bottom right", "top center", "center right",
			"span-left bottom", "span-top right", "span-all top", "block-start inline-end"),
		"position-area spans and logical": areas("span-right top", "x-start y-end", "span-x-end span-y-start",
			"self-block-end", "span-inline-start", "end start"),
		"position-area logical in vertical writing modes": func() *ui.Document {
			var nodes []*ui.Node
			for _, mode := range []string{"vertical-rl", "vertical-lr; direction: rtl", "sideways-lr"} {
				box := el("", "position: relative; width: 780px; height: 180px; writing-mode: "+mode, anchor(""))
				for _, v := range []string{"block-start", "inline-end", "start end", "x-start y-end",
					"span-inline-start block-end", "self-block-end"} {
					box.Append(el("p s", "position-area: "+v))
				}
				box.Append(el("p s", "top: anchor(bottom); left: anchor(right); width: anchor-size(height)"),
					el("p s", "right: anchor(left); bottom: anchor(top)"))
				nodes = append(nodes, box)
			}
			return styled(page+`
.a { left: 300px; top: 60px; }`, nodes...)
		}(),
		"position-area sizes its content": styled(page, anchor(""),
			el("p", "position-area: bottom", ui.Text("some words in the area")),
			el("p", "position-area: left span-all", ui.Text("some words in it")),
			el("p", "position-area: top; width: 100%; margin: 4px")),
		"position-area with insets, margins and alignment": styled(page, anchor(""),
			el("p s", "position-area: bottom; top: 10px; left: 5px"),
			el("p s", "position-area: right; justify-self: end; align-self: start"),
			el("p s", "position-area: top left; margin: 6px")),
		"anchor-center": styled(page, anchor(""),
			el("p s", "top: anchor(bottom); justify-self: anchor-center"),
			el("p s", "left: anchor(right); align-self: anchor-center"),
			el("p", "top: anchor(bottom); justify-self: anchor-center; width: 700px; height: 10px"),
			el("p s", "top: anchor(bottom); left: 0; right: 100px; justify-self: anchor-center")),
		"flips when the box overflows": styled(page, anchor("top: 520px"),
			el("p s", "top: anchor(bottom); left: anchor(left); position-try-fallbacks: flip-block"),
			el("p s", "position-area: bottom; position-try-fallbacks: flip-block"),
			el("p s", "position-area: bottom right; position-try-fallbacks: flip-inline, flip-block"),
			el("p s", "top: anchor(bottom); left: anchor(right); position-try-fallbacks: flip-start"),
			el("p s", "top: anchor(bottom); left: anchor(left); position-try-fallbacks: flip-block flip-inline")),
		"fallbacks: rules, areas and none fitting": styled(page, anchor("left: 760px; top: 520px"),
			el("p s", "position-area: right; position-try-fallbacks: --below, left"),
			el("p s", "position-area: bottom; position-try-fallbacks: top"),
			el("p s", "position-area: right; position-try-fallbacks: bottom, right"),
			el("p s", "position-area: bottom; position-try: --wide, top")),
		"position-try-order": styled(page, anchor("left: 100px; top: 150px"),
			el("p", "position-area: bottom; position-try: most-height top, bottom", ui.Text("x")),
			el("p", "position-area: right; position-try: most-width left, right", ui.Text("x")),
			el("p", "position-area: right; position-try-order: most-width; position-try-fallbacks: left",
				ui.Text("y"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
