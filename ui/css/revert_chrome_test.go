//go:build chrome

package css_test

import (
	"testing"

	"github.com/lantervale/ember/ui"
)

func TestRevertMatchesChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { color: rgb(200, 0, 0); font-size: 14px; }
p, h1, ul { display: inline; margin: 0; padding: 0; color: rgb(0, 0, 200); font-weight: 300; }
.back { display: revert; margin: revert; padding-left: revert; color: revert; font-weight: revert; }
h1.back { font-size: 30px; font-size: revert; }
.short { margin: revert; }
.imp { display: revert !important; }
.twice { display: flex; }
.twice.back { display: revert; }
`))
	d.Body().Append(
		ui.P(ui.Text("inline")),
		ui.P(ui.Class("back"), ui.Text("a paragraph again")),
		ui.H1(ui.Class("back"), ui.Text("a heading again")),
		ui.Ul(ui.Class("back"), ui.Li(ui.Text("a list again"))),
		ui.P(ui.Class("short"), ui.Text("margins back through the shorthand")),
		ui.Div(ui.Class("imp"), ui.Style("display: inline"), ui.Text("important over inline")),
		ui.Span(ui.Class("twice", "back"), ui.Text("a span")),
		ui.Div(ui.Style("color: revert; display: revert"), ui.Text("inline revert")),
	)
	matchChrome(t, d)
}

func TestRevertLayerMatchesChrome(t *testing.T) {
	d := ui.NewDocument(ui.BrowserSheets(), ui.CSS(`
@layer base, theme;
@layer base { p { color: rgb(0, 120, 0); margin-top: 7px; } }
@layer theme { p { color: rgb(0, 0, 200); margin-top: 3px; } .up { color: revert-layer; margin-top: revert-layer; } }
.top { color: revert-layer; display: revert-layer; }
p.top { display: inline; }
@layer theme { .deep { padding-left: 9px; } }
.deep { padding-left: revert-layer; }
@layer base { .only { padding-left: revert-layer; } }
`))
	d.Body().Append(
		ui.P(ui.Text("the theme's")),
		ui.P(ui.Class("up"), ui.Text("the base's")),
		ui.P(ui.Class("top"), ui.Text("the theme's, then the browser's display")),
		ui.P(ui.Class("deep"), ui.Text("the theme's padding")),
		ui.Ul(ui.Class("only"), ui.Li(ui.Text("the browser's padding"))),
		ui.P(ui.Style("color: revert-layer"), ui.Text("an inline revert-layer")),
	)
	matchChrome(t, d)
}
