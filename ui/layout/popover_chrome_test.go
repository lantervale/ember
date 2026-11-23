//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestPopoversMatchChrome(t *testing.T) {
	const page = `body { margin: 0 }
.b { position: absolute; width: 60px; height: 24px }
.under { position-area: bottom }
.beside { position-anchor: auto; left: anchor(right); top: anchor(top); margin: 0 }
.tip { position-area: top span-right; width: 120px }`
	button := func(target, style string) *ui.Node {
		return ui.Button(ui.Class("b"), ui.Attr("popovertarget", target), ui.Style(style), ui.Text("open"))
	}
	pop := func(id, kind, class string, text string) *ui.Node {
		return ui.Div(ui.ID(id), ui.Attr("popover", kind), ui.Class(class), ui.Text(text))
	}
	show := func(d *ui.Document, pairs ...*ui.Node) *ui.Document {
		for i := 0; i < len(pairs); i += 2 {
			d.ShowPopover(pairs[i], pairs[i+1])
		}
		return d
	}
	cases := map[string]func() *ui.Document{
		"shown alone, in the middle": func() *ui.Document {
			c := pop("c", "", "", "a popover in the middle of the page")
			return show(styled(page, c), c, nil)
		},
		"anchored to the button that showed it": func() *ui.Document {
			b1, b2 := button("p1", "left: 100px; top: 80px"), button("p2", "left: 400px; top: 300px")
			p1, p2 := pop("p1", "", "under", "below its button"), pop("p2", "manual", "beside", "beside it")
			return show(styled(page, b1, p1, b2, p2), p1, b1, p2, b2)
		},
		"a hint beside an auto popover": func() *ui.Document {
			b1, p1 := button("p1", "left: 100px; top: 80px"), pop("p1", "", "under", "below its button")
			b3, hint := button("h", "left: 600px; top: 500px"), pop("h", "hint", "tip", "a hint")
			return show(styled(page, b1, p1, b3, hint), p1, b1, hint, b3)
		},
		"closed, it has no box": func() *ui.Document {
			return styled(page, button("p1", "left: 100px; top: 80px"), pop("p1", "", "under", "below its button"))
		},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d(), geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
