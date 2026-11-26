//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestSpansBidiSplitsMatchChrome has Chrome and ember lay out inline
// elements whose content bidi reorders around text outside them, and
// compares each part of each with Chrome's getClientRects.
func TestSpansBidiSplitsMatchChrome(t *testing.T) {
	d := page(
		ui.P(ui.Span(ui.Style("border: 2px solid; padding: 0 5px; margin: 0 3px"), ui.Text("AAA BBB\u202eIII HHH")),
			ui.Text("FFF EEE\u202cJJJ")),
		ui.P(ui.Style("width: 120px"), ui.Span(ui.Style("border-left: 4px solid; padding-right: 6px"),
			ui.Text("one two\u202ethree four five\u202c six"))),
		ui.P(ui.Attr("dir", "rtl"), ui.Span(ui.Style("border: 3px solid; margin-left: 4px"), ui.Text("abc "),
			ui.Span(ui.Attr("dir", "ltr"), ui.Text("def")))),
		ui.P(ui.Style("width: 90px"), ui.Text("ab "), ui.Span(ui.Attr("dir", "rtl"),
			ui.Style("border: 2px solid; padding: 0 3px"), ui.Text("abc def ghi"))),
	)
	size := geom.V2(800, 600)
	r := chrome.Match(t, d, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
	matchSpanFragments(t, d, size)
}
