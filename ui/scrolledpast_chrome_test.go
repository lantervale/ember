//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
)

// A fixed box and an absolutely positioned one whose containing block is above an element
// that scrolls stay where that block placed them as it scrolls, as in Chrome; one whose
// containing block is inside scrolls with it.
func TestBoxesPlacedFromAboveAScrollerMatchChrome(t *testing.T) {
	const sheet = `#wrap { position: relative; padding: 10px; }
#list { overflow: auto; width: 200px; height: 100px; }
#fixed { position: fixed; top: 20px; right: 20px; width: 30px; height: 30px; }
#abs { position: absolute; top: 5px; left: 220px; width: 30px; height: 30px; }
#static { position: absolute; width: 30px; height: 30px; }
#inside { position: relative; height: 40px; }
#inner { position: absolute; top: 5px; left: 5px; width: 20px; height: 10px; }
.row { height: 30px; }`
	for name, c := range map[string]struct{ page, list float32 }{
		"the list scrolled":          {0, 120},
		"the list and page scrolled": {200, 60},
	} {
		t.Run(name, func(t *testing.T) {
			d := ui.NewDocument(ui.ScrollingViewport(), ui.CSS(sheet))
			list := ui.Div(ui.ID("list"), ui.Div(ui.Class("row")), ui.Div(ui.ID("fixed")), ui.Div(ui.ID("abs")),
				ui.Div(ui.ID("static")), ui.Div(ui.ID("inside"), ui.Div(ui.ID("inner"))), ui.Div(ui.Style("height: 400px")))
			d.Body().Append(ui.Div(ui.ID("wrap"), list), ui.Div(ui.Style("height: 1200px")))
			dom.Data(d.HTML()).Scroll = geom.V2(0, c.page)
			dom.Data(list).Scroll = geom.V2(0, c.list)
			r := chrome.Match(t, d, geom.V2(640, 400))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
