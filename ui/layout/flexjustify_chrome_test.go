//go:build chrome

package layout_test

import (
	"strings"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
)

// TestFlexJustifyLeftAndRightMatchChrome lays out justify-content: left and right, WPT
// css/css-flexbox/flexbox-justify-content-vert-*: the sides they name along a row in
// either direction and reversed, start along a column whatever its direction.
func TestFlexJustifyLeftAndRightMatchChrome(t *testing.T) {
	const sheet = `.f { display: flex; width: 120px; height: 60px; margin-bottom: 4px; background: #eee }
.f > div { width: 20px; height: 15px; background: #888 } .c { flex-direction: column }
.cr { flex-direction: column-reverse } .rr { flex-direction: row-reverse } .rtl { direction: rtl }
.v { writing-mode: vertical-rl } .vl { writing-mode: vertical-lr }`
	box := func(classes, justify string) *ui.Node {
		return ui.Div(ui.Class(strings.Fields(classes)...), ui.Style("justify-content: "+justify), ui.Div(), ui.Div())
	}
	var rows []*ui.Node
	for _, classes := range []string{"f", "f rtl", "f rr", "f rr rtl", "f c", "f c rtl", "f cr", "f v", "f v rtl",
		"f c v", "f c vl", "f cr v", "f c v rtl", "f vl rtl"} {
		for _, j := range []string{"left", "right", "safe right", "unsafe left"} {
			rows = append(rows, box(classes, j))
		}
	}
	matchPages(t, geom.Vec2{X: 800, Y: 2400}, map[string]*ui.Document{"left and right": styled(sheet, rows...)})
}
