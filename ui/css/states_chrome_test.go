//go:build chrome

package css_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/assets"
	"github.com/lantervale/ember/ui/internal/dom"
)

// TestStatesReadAwayFromTheSubjectStyleWhatChromeStyles styles a page,
// then puts the pointer over a menu, its row and the row's item, a press
// on the menu and the focus in a field, and compares what the restyle of
// those states alone leaves with Chrome's styles of the page in them.
func TestStatesReadAwayFromTheSubjectStyleWhatChromeStyles(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
span, a, b, input { display: inline-block; }
.menu:hover .item { background-color: #f00; }
.menu:hover > .label { color: #0a0; }
.menu:hover { color: #00f; font-size: 20px; }
.menu:hover + p { padding-left: 4px; }
.menu:active ~ p span { margin-top: 2px; }
.label:hover ~ .row b { opacity: 0.5; }
.row:hover ~ a { border-top: 3px solid #123; }
:is(.menu:hover, .row:focus) > :first-child { padding-top: 1px; }
.item:not(.menu:hover .item) { margin-right: 3px; }
:not(:hover) > b { letter-spacing: 2px; }
:where(.menu:hover .row) + a { font-weight: 700; }
.menu:hover .row:hover .item:hover b { font-style: italic; }
.menu:not(:hover) .item { opacity: 0.25; }
input:focus + span { background-color: #ff0; }
input:focus-visible ~ .item { outline: 2px solid #f0f; }
p:hover input { opacity: 0.75; }
.menu:hover * { text-transform: uppercase; }
`))
	item := ui.Span(ui.Class("item"), ui.B(ui.Text("deep")))
	row := ui.Div(ui.Class("row"), item)
	menu := ui.Div(ui.Class("menu"), ui.Span(ui.Class("label"), ui.Text("label")), row, ui.A(ui.Text("link")))
	field := ui.Input(ui.Type("checkbox"))
	d.Body().Append(
		menu,
		ui.P(ui.Class("after"), ui.Span(ui.Class("item"), ui.B(ui.Text("in")))),
		ui.P(field, ui.Span(ui.Class("item"), ui.Text("tail")), ui.Span(ui.Class("item"), ui.Text("last"))),
		ui.Div(ui.Class("menu"), ui.Span(ui.Class("label"), ui.Text("other")), ui.Span(ui.Class("item"), ui.B(ui.Text("b")))),
	)
	d.Layout(geom.V2(800, 600))
	for _, n := range []*ui.Node{d.HTML(), d.Body(), menu, row, item} {
		assets.Pin(d, n, dom.Hover)
	}
	assets.Pin(d, menu, dom.Active)
	assets.Pin(d, field, dom.Focus|dom.FocusVisible)
	matchChrome(t, d)
}
