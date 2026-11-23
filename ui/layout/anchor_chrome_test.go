//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestAnchorPositioningMatchesChrome(t *testing.T) {
	const page = `body { margin: 0; position: relative; height: 560px }
.a { position: absolute; left: 120px; top: 200px; width: 80px; height: 40px; background: #ccc; anchor-name: --a }
.p { position: absolute; position-anchor: --a; width: 50px; height: 30px; background: #c00 }
.box { position: relative; margin: 40px; width: 500px; height: 400px; border: 3px solid; padding: 7px }`
	el := func(class, style string, kids ...*ui.Node) *ui.Node {
		args := []ui.Arg{ui.Class(class), ui.Style(style)}
		for _, k := range kids {
			args = append(args, k)
		}
		return ui.Div(args...)
	}
	anchor := func(style string) *ui.Node { return el("a", style) }
	cases := map[string]*ui.Document{
		"anchor() sides on each inset": styled(page, anchor(""),
			el("p", "top: anchor(bottom); left: anchor(right)"),
			el("p", "bottom: anchor(top); right: anchor(left)"),
			el("p", "top: anchor(center); left: anchor(50%)"),
			el("p", "top: anchor(start); left: anchor(end)"),
			el("p", "bottom: anchor(inside); right: anchor(outside)"),
			el("p", "top: calc(anchor(bottom) + 10px); left: anchor(25%)")),
		"anchor-size() in sizes, insets and margins": styled(page, anchor(""),
			el("p", "top: anchor(bottom); left: anchor(left); width: anchor-size(width); height: anchor-size()"),
			el("p", "top: anchor-size(height); left: 0; width: calc(anchor-size(width) * 2)"),
			el("p", "top: 0; left: 0; margin-left: anchor-size(width); min-height: anchor-size(height)"),
			el("p", "top: 0; left: 0; max-width: anchor-size(height); width: 100px")),
		"a missing anchor takes the fallback or auto": styled(page, anchor(""),
			el("p", "top: anchor(--none bottom, 15px); left: anchor(--none left)"),
			el("p", "position-anchor: --none; top: anchor(bottom, 50%); left: anchor(left, 3px)")),
		"named anchors and the last of a name": styled(page, anchor(""),
			el("a", "left: 300px; top: 20px; anchor-name: --b"),
			el("a", "left: 400px; top: 60px; anchor-name: --b"),
			el("p", "top: anchor(--b bottom); left: anchor(--a right)")),
		"in a containing block with a border and padding": styled(page,
			el("box", "", anchor("left: 30px; top: 50px"),
				el("p", "top: anchor(bottom); left: anchor(right)"),
				el("p", "position: fixed; top: anchor(top); right: anchor(left)"))),
		"an anchor deeper in the tree": styled(page,
			el("box", "", ui.Div(ui.Style("margin: 20px; padding: 10px"), ui.Div(ui.Class("x"),
				ui.Style("anchor-name: --deep; width: 60px; height: 10px; margin-left: 33px"))),
				el("p", "top: anchor(--deep bottom); left: anchor(--deep right)"))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
