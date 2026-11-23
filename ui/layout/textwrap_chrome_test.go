//go:build chrome

package layout_test

import (
	"fmt"
	"strings"
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
			ui.P(ui.Style("width: 220px; text-wrap: balance; text-align: center"),
				ui.Text("Lanterns everywhere glowing at dusk")),
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

const soft = "Lan\u00adtern\u00admakers il\u00adlu\u00admi\u00adnate ex\u00adtra\u00adordi\u00adnar\u00adily " +
	"lu\u00admi\u00adnous " +
	"ca\u00adthe\u00addrals."

func TestSoftHyphensBreakAsChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"manual": page(
			ui.P(ui.Style("width: 90px"), ui.Text(soft)),
			ui.P(ui.Style("width: 130px"), ui.Text(soft)),
			ui.P(ui.Style("width: 400px"), ui.Text(soft)),
		),
		"none":                    page(ui.P(ui.Style("width: 90px; hyphens: none"), ui.Text(soft))),
		"auto without a language": page(ui.P(ui.Style("width: 110px; hyphens: auto"), ui.Text(soft))),
		"hyphenate-character": page(
			ui.P(ui.Style("width: 100px; hyphenate-character: '='"), ui.Text(soft)),
			ui.P(ui.Style("width: 100px; hyphenate-character: '\\2013\\2013'"), ui.Text(soft)),
			ui.P(ui.Style("width: 100px; hyphenate-character: ''"), ui.Text(soft)),
		),
		"spans": page(
			ui.P(ui.Style("width: 80px"), ui.Text("Lan\u00ad"), ui.Span(ui.Style("font-weight: bold"),
				ui.Text("tern\u00admakers")), ui.Text(" il\u00adlu\u00admi\u00adnate")),
		),
		"aligned and balanced": page(
			ui.P(ui.Style("width: 120px; text-align: right"), ui.Text(soft)),
			ui.P(ui.Style("width: 120px; text-align: justify"), ui.Text(soft)),
			ui.P(ui.Style("width: 160px; text-wrap: balance"), ui.Text(soft)),
			ui.P(ui.Style("width: 120px; text-wrap: pretty"), ui.Text(soft)),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600))
			if name != "spans" {
				// Chrome gives a span's hyphen a rect of its own, an artefact of the
				// fragment it makes for it, not a box.
				matchSpanFragments(t, d, geom.V2(800, 600))
			}
		})
	}
}

func TestTextBoxTrimsAsChrome(t *testing.T) {
	box := "background: #eee; margin: 4px 0; "
	cases := map[string]*ui.Document{
		"edges": page(
			ui.P(ui.Style(box+"text-box: trim-both cap alphabetic"), ui.Text("Cap to baseline")),
			ui.P(ui.Style(box+"text-box: trim-both ex alphabetic"), ui.Text("Ex to baseline")),
			ui.P(ui.Style(box+"text-box: trim-both text"), ui.Text("Text to text")),
			ui.P(ui.Style(box+"text-box-trim: trim-both"), ui.Text("Auto")),
			ui.P(ui.Style(box+"text-box: trim-both cap text"), ui.Text("Cap to text")),
		),
		"one end": page(
			ui.P(ui.Style(box+"text-box: trim-start cap alphabetic"), ui.Text("Start only")),
			ui.P(ui.Style(box+"text-box: trim-end cap alphabetic"), ui.Text("End only")),
		),
		"line heights and lines": page(
			ui.P(ui.Style(box+"line-height: 2; width: 120px; text-box: trim-both cap alphabetic"), ui.Text(sentence)),
			ui.P(ui.Style(box+"line-height: 10px; text-box: trim-both cap alphabetic"), ui.Text("Tight")),
			ui.H1(ui.Style(box+"text-box: trim-both cap alphabetic"), ui.Text("A heading")),
			ui.P(ui.Style(box+"font-size: 30px; line-height: 1.5; text-box: trim-both ex alphabetic"),
				ui.Text("Big "), ui.Span(ui.Style("font-size: 12px"), ui.Text("small"))),
		),
		"padding and borders": page(
			ui.Div(ui.Style(box+"padding: 10px; border: 2px solid; text-box: trim-both cap alphabetic"),
				ui.Text("Padded")),
			ui.Button(ui.Style("text-box: trim-both cap alphabetic; padding: 8px 12px"), ui.Text("Button")),
		),
		"blocks inside": page(
			ui.Div(ui.Style(box+"text-box: trim-both cap alphabetic"),
				ui.P(ui.Style("margin: 0"), ui.Text("First child")), ui.P(ui.Style("margin: 0"), ui.Text("Last child"))),
			ui.Div(ui.Style(box+"text-box: trim-both cap alphabetic"),
				ui.Div(ui.Style("padding-top: 1px"), ui.Text("Padded child")), ui.Div(ui.Text("Last"))),
		),
		"inline boxes": page(
			ui.P(ui.Text("A "), ui.Span(ui.Style("background: #ccc; text-box: trim-both cap alphabetic"),
				ui.Text("trimmed span")), ui.Text(" in a line")),
		),
		"inherited edge, trim not": page(
			ui.Div(ui.Style("text-box-edge: cap alphabetic"),
				ui.P(ui.Style(box+"text-box-trim: trim-both"), ui.Text("Edge from the parent"))),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
			matchText(t, d, geom.V2(800, 600))
		})
	}
}

// spanned is text with each word in a span of its own, so the boxes show
// which line each word stands on.
func spanned(style, text string) *ui.Node {
	args := []ui.Arg{ui.Style(style)}
	for i, w := range strings.Fields(text) {
		if i > 0 {
			args = append(args, ui.Text(" "))
		}
		args = append(args, ui.Span(ui.Text(w)))
	}
	return ui.P(args...)
}

func TestVerticalLinesWrapAsChrome(t *testing.T) {
	cases := map[string]*ui.Document{
		"balance": page(
			spanned("writing-mode: vertical-rl; height: 200px; text-wrap: balance", shortText),
			spanned("writing-mode: vertical-lr; height: 260px; text-wrap: balance", middleText),
		),
		"pretty": page(
			spanned("writing-mode: vertical-rl; height: 200px; text-wrap: pretty",
				"The keep's lantern swings over the gate all night long"),
		),
		"soft hyphens": page(
			ui.P(ui.Style("writing-mode: vertical-rl; height: 90px"), ui.Text(soft)),
			ui.P(ui.Style("writing-mode: vertical-rl; height: 100px; hyphenate-character: '='"), ui.Text(soft)),
		),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}
