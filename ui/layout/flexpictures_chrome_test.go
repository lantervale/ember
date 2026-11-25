//go:build chrome

package layout_test

import (
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestFlexPicturesKeepTheirCrossLimitsMatchChrome lays out pictures as flex items whose
// min or max cross size their proportions carry to their main size, WPT
// css/css-flexbox/image-as-flexitem-size-* and flex-aspect-ratio-img-column-006: a row
// and a column, with padding, borders and box-sizing.
func TestFlexPicturesKeepTheirCrossLimitsMatchChrome(t *testing.T) {
	fsys := fstest.MapFS{"sq.png": {Data: squarePNG(16)}, "tall.png": {Data: rectPNG(20, 50)},
		"page.css": {Data: []byte(`
.f { display: flex; border: 1px solid black; margin: 0 2px 2px 0; width: 40px; height: 40px;
  justify-content: flex-start; align-items: flex-start; float: left }
.c { flex-direction: column } .free { width: auto; height: auto }
.f > * { min-width: 0; min-height: 0 }
br { clear: both }`)}}
	pics := []string{"min-width: 34px", "min-height: 34px", "min-width: 34px; min-height: 30px",
		"max-width: 10px", "max-height: 10px", "max-width: 10px; max-height: 12px",
		"min-height: 34px; padding: 2px 3px", "min-height: 34px; border: 2px solid; box-sizing: border-box",
		"min-width: 34px; width: 20px"}
	d := ui.NewDocument(ui.Stylesheet(fsys, "page.css"), ui.Images(fsys))
	for _, box := range [][]string{{"f"}, {"f", "c"}, {"f", "free"}, {"f", "c", "free"}} {
		for _, pic := range []string{"sq.png", "tall.png"} {
			for _, style := range pics {
				d.Body().Append(ui.Div(ui.Class(box...), ui.Img(ui.Attr("src", pic), ui.Style(style))))
			}
			d.Body().Append(ui.Br())
		}
	}
	r := chrome.Match(t, d, geom.Vec2{X: 800, Y: 600})
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
