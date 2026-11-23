//go:build chrome

package layout_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestScopedAndChainedAnchorsMatchChrome(t *testing.T) {
	const page = `body { margin: 0; position: relative; height: 560px }
.a { width: 80px; height: 40px; margin-left: 90px; background: #ccc; anchor-name: --a }
.p { position: absolute; position-anchor: --a; width: 50px; height: 30px; background: #c00 }
.c { position: relative; width: 200px; height: 100px; margin: 10px; anchor-scope: --a }
.c .a { margin-left: 20px }`
	el := func(class, style string, kids ...ui.Arg) *ui.Node {
		return ui.Div(append([]ui.Arg{ui.Class(strings.Fields(class)...), ui.Style(style)}, kids...)...)
	}
	cases := map[string]*ui.Document{
		"anchor-scope keeps a name to a subtree": styled(page,
			el("a", "margin-top: 300px"),
			el("c", "", el("a", ""), el("p", "top: anchor(bottom); left: anchor(left)")),
			el("c", "", el("a", "margin-left: 60px"), el("p", "top: anchor(bottom); left: anchor(left)")),
			el("p", "top: anchor(bottom); left: anchor(right)")),
		"an anchored box anchors another": styled(page,
			el("a", "margin-top: 100px"),
			el("p", "top: anchor(bottom); left: anchor(right); anchor-name: --b"),
			el("p", "top: anchor(--b bottom); left: anchor(--b right); anchor-name: --c"),
			el("p", "top: anchor(--c bottom); left: anchor(--c right)")),
		"an anchor after its box in the same containing block is no anchor": styled(page,
			el("p", "top: anchor(--late bottom, 7px); left: anchor(--late left, 9px)"),
			el("p", "anchor-name: --late; top: 200px; left: 200px")),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

