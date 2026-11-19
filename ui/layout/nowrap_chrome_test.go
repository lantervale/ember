//go:build chrome

package layout_test

import (
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// A line breaks at a space of wrapping text before a span that does not wrap, beside a
// float as anywhere: the span goes to the next line, not the whole line below the float.
func TestALineBreaksBeforeASpanThatDoesNotWrap(t *testing.T) {
	const sheet = `body { width: 400px } .f { float: right; width: 200px; height: 100px }
.n { white-space: nowrap } .box { width: 200px } .no { white-space: nowrap; width: 120px } .y { white-space: normal }`
	nowrap := func(nodes ...ui.Arg) *ui.Node { return ui.Span(append([]ui.Arg{ui.Class("n")}, nodes...)...) }
	cases := map[string]*ui.Document{
		"beside a float": styled(sheet, ui.Div(ui.Class("f")),
			ui.P(ui.B(ui.Text("Le")), ui.Text(" Canada est un et "), nowrap(ui.Text("grand pays du nord")),
				ui.Text(" de l'Amérique"))),
		"links in the span": styled(sheet, ui.Div(ui.Class("f")),
			ui.P(ui.B(ui.Text("Le")), ui.Text(" Canada est un pays ou "),
				nowrap(ui.Text("/"), ui.A(ui.Text("kana")), ui.A(ui.Text("dakana")), ui.A(ui.Text("dakana")), ui.Text("/")),
				ui.Sup(ui.Text("6")), ui.Text(" et après"))),
		"in a narrow block": styled(sheet,
			ui.P(ui.Class("box"), ui.Span(ui.Text("Le Canada est un")), ui.Text(" "),
				nowrap(ui.Text("grand pays du nord")), ui.Text(" "), nowrap(ui.Text("de l'Amérique")))),
		"a wrapping span in text that does not": styled(sheet,
			ui.P(ui.Class("no"), ui.Text("ne coupe pas "), ui.Span(ui.Class("y"), ui.Text("mais ici oui, entre ces mots")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
