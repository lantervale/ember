//go:build chrome

package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestLogicalPropertiesMatchChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"a boxed container centred by margin-inline in a stretched section": styled(`
.section { position: relative; width: 600px; left: -20px; background: #ccc; }
.container { display: flex; margin-inline: auto; max-width: 300px; }
.column { width: 100%; height: 40px; background: #888; }`,
			ui.Div(ui.Class("section"), ui.Div(ui.Class("container"), ui.Div(ui.Class("column")))),
		),
		"each side by its logical name": styled(`div { width: 300px; background: #ccc; }
p { background: #888; margin: 0; }
.a { margin-inline-start: 10px; margin-inline-end: 20px; margin-block-start: 5px; margin-block-end: 15px; }
.b { padding-inline: 10px 30px; padding-block: 4px; margin-block: 6px 2px; }
.c { inline-size: 120px; block-size: 30px; min-block-size: 40px; max-inline-size: 100px; }
.d { border-inline-start: 4px solid #000; border-block-end-width: 3px; border-block-end-style: solid;
  border-start-start-radius: 8px; border-end-end-radius: 6px; }
.e { position: relative; block-size: 50px; }
.e span { position: absolute; inset-inline-end: 10px; inset-block-start: 5px; inline-size: 20px; block-size: 20px; }
.f span { position: absolute; inset-inline: 10px 30px; inset-block: 5px; }`,
			ui.Div(ui.P(ui.Class("a"), ui.Text("a")), ui.P(ui.Class("b"), ui.Text("b")), ui.P(ui.Class("c"), ui.Text("c")),
				ui.P(ui.Class("d"), ui.Text("d")), ui.P(ui.Class("e"), ui.Span()), ui.P(ui.Class("e", "f"), ui.Span())),
		),
		"the last declared wins, logical or not": styled(`p { width: 200px; margin: 0; background: #888; }
.a { margin-left: 10px; margin-inline-start: 30px; } .b { margin-inline-start: 30px; margin-left: 10px; }
.c { margin-inline: 40px; margin-right: 0; width: auto; }`,
			ui.Div(ui.Style("width: 300px"), ui.P(ui.Class("a"), ui.Text("a")), ui.P(ui.Class("b"), ui.Text("b")),
				ui.P(ui.Class("c"), ui.Text("c"))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			assert.Empty(t, d.Warnings())
		})
	}
}
