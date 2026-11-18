//go:build chrome

package layout_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

func TestDisplayTableOnAnyElementMatchesChrome(t *testing.T) {
	const sheet = `.l { float: left; } .r { float: right; }
.box { width: 60px; height: 50px; background: #888; }
.wide { width: 300px; margin: 0 0 8px; border: 2px solid #000; }
.fix::before, .fix::after { content: ''; display: table; }
.fix::after { clear: both; }
.table { display: table; background: #ccc; }
.inline { display: inline-table; background: #ccc; }`
	cases := map[string]*ui.Document{
		"a clearfix holds the floats": styled(sheet,
			ui.Div(ui.Class("wide", "fix"),
				ui.Div(ui.Class("l", "box")), ui.Div(ui.Class("r", "box"), ui.Style("height: 70px"))),
			ui.Div(ui.Class("wide", "fix"), ui.Div(ui.Class("l", "box")), ui.Text("Beside.")),
			ui.Div(ui.Class("wide"), ui.Div(ui.Class("l", "box")), ui.Div(ui.Style("display: flow-root; clear: both"))),
			ui.Div(ui.Class("wide"), ui.Text("After.")),
		),
		"a header and a menu of floats, each in a clearfix": styled(sheet+`
.menu li { float: left; list-style: none; line-height: 30px; padding: 0 10px; } .menu { margin: 0; padding: 0; }
.bar { background: #36f; } .banner { overflow: hidden; height: 40px; background: #333; }`,
			ui.Div(ui.Style("width: 400px"),
				ui.Div(ui.Class("fix"), ui.Div(ui.Class("l", "box")), ui.Div(ui.Class("r", "box"))),
				ui.Div(ui.Class("bar"), ui.Div(ui.Class("fix"),
					ui.Ul(ui.Class("menu"), ui.Li(ui.Text("One")), ui.Li(ui.Text("Two")), ui.Li(ui.Text("Three"))))),
				ui.Div(ui.Class("banner"))),
		),
		"shrinks to its content": styled(sheet,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("table"), ui.Text("Short.")),
				ui.Div(ui.Class("table"), ui.Text(sentence+" "+sentence)),
				ui.Div(ui.Class("table"), ui.Style("margin: 0 auto"), ui.Text("Centred.")),
				ui.Div(ui.Class("table"), ui.Style("width: 100%"), ui.Text("As wide as said.")),
				ui.Div(ui.Class("table"), ui.Style("width: 120px; padding: 5px"), ui.Text("Sized by its border.")),
				ui.Text("Text "), ui.Span(ui.Class("inline"), ui.Text("an inline table")), ui.Text(" and more.")),
		),
		"holds its floats and stands beside those before": styled(sheet,
			ui.Div(ui.Class("wide"),
				ui.Div(ui.Class("l", "box"), ui.Style("margin-right: 8px")),
				ui.Div(ui.Class("table"), ui.Style("width: 200px"), ui.Div(ui.Class("l", "box"), ui.Style("height: 40px")),
					ui.Text("Round its float.")),
				ui.Div(ui.Class("table"), ui.P(ui.Style("margin: 10px 0"), ui.Text("Its margins stay inside.")))),
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
