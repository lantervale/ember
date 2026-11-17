//go:build chrome

package layout_test

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/fonts"
)

func TestAnInstalledFontMatchesChrome(t *testing.T) {
	if _, err := os.Stat("/System/Library/Fonts/Supplemental/Georgia.ttf"); err != nil {
		t.Skip("Georgia is not installed")
	}
	const sheet = `body { font-family: georgia, sans-serif; margin: 8px; }
p { width: 300px; margin: 0 0 12px; }
.small { font-size: 12px; line-height: 1.5; } .big { font-size: 22px; }
.bold { font-weight: bold; } .italic { font-style: italic; }`
	cases := map[string][]*ui.Node{
		"a paragraph": {words("", sentence+" "+sentence)},
		"sizes":       {words("", sentence), ui.Div(ui.Class("small"), words("", sentence)), ui.Div(ui.Class("big"), words("", sentence))},
		"bold, italic and both": {
			ui.Div(ui.Class("bold"), words("", sentence)),
			ui.Div(ui.Class("italic"), words("", sentence)),
			ui.Div(ui.Class("bold", "italic"), words("", sentence)),
		},
	}
	for _, embedded := range []bool{false, true} {
		for name, nodes := range cases {
			if embedded {
				name += ", its file given to Chrome"
			}
			t.Run(name, func(t *testing.T) {
				d := ui.NewDocument(ui.SystemFonts(),
					ui.Stylesheet(fstest.MapFS{"page.css": {Data: []byte(sheet)}}, "page.css"))
				d.Body().Append(nodes...)
				fonts.Of(d).Await()
				if embedded {
					d.Layout(geom.V2(800, 600))
				}
				r := chrome.Match(t, d, geom.V2(800, 600))
				t.Logf("%d boxes, max delta %.3f", r.Boxes, r.MaxDelta)
				for _, w := range d.Warnings() {
					t.Error(w)
				}
			})
		}
	}
}
