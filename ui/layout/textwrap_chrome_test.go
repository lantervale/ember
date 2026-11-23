//go:build chrome

package layout_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

const (
	shortText  = "Lantern light on the old stone walls"
	middleText = "A lantern of brass swings over the gate, its light falling on the old stone walls of the keep."
	longText   = "The quick brown fox jumps over the lazy dog, then naps in the warm afternoon sun while the " +
		"farmer counts his sheep twice, loses count, and starts all over again from the very first one."
)

func TestTextWrapBalanceBreaksAsChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"short paragraphs": page(
			ui.H1(ui.Style("width: 300px; text-wrap: balance"), ui.Text("A heading that wraps onto two lines")),
			ui.P(ui.Style("width: 200px; text-wrap: balance"), ui.Text(shortText)),
			ui.P(ui.Style("width: 140px; text-wrap: balance"), ui.Text(middleText)),
			ui.P(ui.Style("width: 260px; text-wrap: balance"), ui.Text(middleText)),
			ui.P(ui.Style("width: 90px; text-wrap: balance"), ui.Text("one two three")),
		),
		"few breaks bisected": page(
			ui.P(ui.Style("width: 120px; text-wrap: balance"), ui.Text("Lanterns everywhere glowing")),
			ui.P(ui.Style("width: 100px; text-wrap: balance"), ui.Text("Supercalifragilistic expialidocious")),
		),
		"too long to balance": page(
			ui.P(ui.Style("width: 120px; text-wrap: balance"), ui.Text(longText)),
			ui.P(ui.Style("width: 220px; text-wrap: balance"), ui.Text(longText+" "+longText)),
		),
		"aligned": page(
			ui.P(ui.Style("width: 220px; text-wrap: balance; text-align: center"), ui.Text(middleText)),
			ui.P(ui.Style("width: 220px; text-wrap: balance; text-align: right"), ui.Text(middleText)),
			ui.P(ui.Style("width: 220px; text-wrap: balance; text-align: center"), ui.Text("Lanterns everywhere glowing at dusk")),
			ui.P(ui.Style("width: 220px; text-wrap: balance; text-align: justify"), ui.Text(middleText)),
		),
		"indented": page(
			ui.P(ui.Style("width: 200px; text-wrap: balance; text-indent: 30px"), ui.Text(middleText)),
		),
		"spans": page(
			ui.P(ui.Style("width: 200px; text-wrap: balance"), ui.Text("A lantern of "),
				ui.Span(ui.Style("font-weight: bold; padding: 0 6px; border: 1px solid"), ui.Text("polished brass")),
				ui.Text(" swings over the gate of the keep.")),
			ui.P(ui.Style("width: 180px; text-wrap: balance"), ui.Text("Big "),
				ui.Span(ui.Style("font-size: 24px"), ui.Text("and small words")), ui.Text(" share a line or two.")),
		),
		"forced breaks": page(
			ui.P(ui.Style("width: 200px; text-wrap: balance"), ui.Text(middleText), ui.Br(), ui.Text(middleText)),
			ui.P(ui.Style("width: 200px; text-wrap: balance; white-space: pre-line"),
				ui.Text("Short first line\n"+middleText)),
		),
		"inherited": page(
			ui.Div(ui.Style("text-wrap: balance; width: 240px"),
				ui.P(ui.Text(middleText)), ui.P(ui.Style("text-wrap: wrap"), ui.Text(middleText))),
		),
		"shrink to fit": page(
			ui.Div(ui.Style("display: inline-block; max-width: 250px; text-wrap: balance"), ui.Text(middleText)),
			ui.Div(ui.Style("display: flex; width: 300px"),
				ui.Div(ui.Style("text-wrap: balance"), ui.Text(middleText))),
		),
		"a sweep of widths": sweep("balance"),
		"letter-spacing": page(
			ui.P(ui.Style("width: 220px; text-wrap: balance; letter-spacing: 1px; word-spacing: 3px"), ui.Text(middleText)),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600))
			matchSpanFragments(t, d, geom.V2(800, 600))
		})
	}
}

func TestTextWrapPrettyBreaksAsChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"a short last line": page(
			ui.P(ui.Style("width: 200px; text-wrap: pretty"), ui.Text("The keep's lantern swings over the gate all night long")),
			ui.P(ui.Style("width: 150px; text-wrap: pretty"), ui.Text(middleText)),
			ui.P(ui.Style("width: 230px; text-wrap: pretty"), ui.Text(middleText)),
			ui.P(ui.Style("width: 170px; text-wrap: pretty"), ui.Text(sentence)),
		),
		"long paragraphs": page(
			ui.P(ui.Style("width: 160px; text-wrap: pretty"), ui.Text(longText)),
			ui.P(ui.Style("width: 210px; text-wrap: pretty"), ui.Text(longText)),
			ui.P(ui.Style("width: 250px; text-wrap: pretty"), ui.Text(longText+" "+longText)),
		),
		"forced breaks": page(
			ui.P(ui.Style("width: 150px; text-wrap: pretty"), ui.Text(middleText), ui.Br(), ui.Text(middleText)),
		),
		"justified and indented": page(
			ui.P(ui.Style("width: 150px; text-wrap: pretty; text-align: justify"), ui.Text(middleText)),
			ui.P(ui.Style("width: 150px; text-wrap: pretty; text-indent: 2em"), ui.Text(middleText)),
		),
		"spans": page(
			ui.P(ui.Style("width: 160px; text-wrap: pretty"), ui.Text("A lantern of "),
				ui.Span(ui.Style("font-weight: bold; padding: 0 6px"), ui.Text("polished brass")),
				ui.Text(" swings over the gate of the keep all night.")),
		),
		"a sweep of widths": sweep("pretty"),
		"a last word too long to move": page(
			ui.P(ui.Style("width: 120px; text-wrap: pretty"), ui.Text("Lanterns glow over walls tonight")),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600))
			matchSpanFragments(t, d, geom.V2(800, 600))
		})
	}
}

// sweep is a page of paragraphs of every length of text, each at widths
// from narrow to wide, under text-wrap-style wrap.
func sweep(wrap string) *ui.Document {
	texts := []string{shortText, sentence, middleText, longText, "Gate keep wall tower moat bridge yard hall",
		"An old, old lantern-maker's shop sells lanterns: tin, brass, iron and glass ones."}
	var ps []*ui.Node
	for i, text := range texts {
		for w := 70 + 13*i; w < 420; w += 47 {
			ps = append(ps, ui.P(ui.Style(fmt.Sprintf("width: %dpx; text-wrap: %s; margin: 0", w, wrap)), ui.Text(text)))
		}
	}
	return page(ps...)
}
