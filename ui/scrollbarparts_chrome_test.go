//go:build chrome

package ui_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// Bars their sheets style are as thick as their ::-webkit-scrollbar says,
// each way, none for display: none, and a stable gutter is kept whether
// the content overflows or not: the content beside them is laid out in
// what is left, as in Chrome.
func TestStyledScrollbarsTakeTheirRoomAsChromes(t *testing.T) {
	const sheet = `.s { width: 120px; height: 80px; overflow: scroll; margin: 4px; }
.s > div { width: 300px; height: 300px; }
.thick::-webkit-scrollbar { width: 22px; height: 9px; }
.thin::-webkit-scrollbar:vertical { width: 4px; }
.thin::-webkit-scrollbar:horizontal { height: 30px; }
.none::-webkit-scrollbar { display: none; }
.auto { overflow: auto; }
.auto > div { width: 50%; height: 20px; }
.stable { scrollbar-gutter: stable; }
.stable::-webkit-scrollbar { width: 11px; height: 11px; }
.hidden { overflow: hidden; }`
	d := ui.NewDocument(ui.CSS(sheet))
	for _, c := range [][]string{{"thick"}, {"thin"}, {"none"}, {"auto", "thick"}, {"auto", "stable"},
		{"hidden", "stable"}, {"auto", "thin"}} {
		d.Body().Append(ui.Div(ui.Class(append([]string{"s"}, c...)...), ui.Div(ui.Class("in"))))
	}
	r := chrome.Match(t, d, geom.V2(640, 600))
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
