//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestSelfAlignmentSidesMatchChrome places blocks and absolutely positioned boxes by
// justify-self and align-self in containers of either direction and writing mode: start
// and end are the container's, self-start and self-end the item's own, left and right
// the line's.
func TestSelfAlignmentSidesMatchChrome(t *testing.T) {
	const page = `body { margin: 0 } .c { display: inline-block; position: relative; margin: 4px;
border: solid 4px; width: 40px; height: 40px }
.i { background: green } .i::before { width: 20px; height: 20px; content: ''; display: block }
.a { position: absolute; inset: 0 }`
	keywords := []string{"start", "center", "end", "self-start", "self-end", "left", "right", "safe end"}
	containers := []string{"direction: ltr", "direction: rtl", "writing-mode: vertical-rl",
		"writing-mode: vertical-lr; direction: rtl"}
	items := []string{"direction: ltr", "direction: rtl", "writing-mode: vertical-rl", "writing-mode: vertical-lr"}
	cases := map[string]string{"a block by justify-self": "", "an absolutely positioned box by justify-self": "a",
		"an absolutely positioned box by align-self": "a"}
	for name, class := range cases {
		property := "justify-self: "
		if name == "an absolutely positioned box by align-self" {
			property = "align-self: "
		}
		t.Run(name, func(t *testing.T) {
			d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(page))
			for _, c := range containers {
				for _, it := range items {
					if class == "" && it == "writing-mode: vertical-rl" {
						continue
					}
					for _, k := range keywords {
						if k == "left" || k == "right" {
							if property == "align-self: " {
								continue
							}
						}
						d.Body().Append(ui.Div(ui.Class("c"), ui.Style(c),
							ui.Div(ui.Class("i", class), ui.Style(it+"; "+property+k))))
					}
				}
			}
			r := chrome.Match(t, d, geom.V2(800, 1200))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
