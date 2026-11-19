//go:build chrome

package css_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestSelectorsMatchWhatChromeMatches asks Chrome which elements of a
// page each selector matches: the sibling combinators, :is(), :where(),
// :enabled, the places counted by type or from the end, and the
// attribute matchers.
func TestSelectorsMatchWhatChromeMatches(t *testing.T) {
	d := ui.NewDocument()
	d.Body().Append(
		ui.H2(ui.ID("h"), ui.Text("title")),
		ui.P(ui.ID("p1"), ui.Class("lead"), ui.Text("one")),
		ui.P(ui.ID("p2"), ui.Text("two")),
		ui.Div(ui.ID("box"), ui.Attr("lang", "fr-CA"), ui.Attr("data-kind", "Note Wide"),
			ui.Span(ui.ID("s1"), ui.Class("vector-icon"), ui.Class("mw-ui-icon-menu-progressive"), ui.Text("a")),
			ui.Text(" text "),
			ui.Span(ui.ID("s2"), ui.Text("b")),
			ui.A(ui.ID("a"), ui.Attr("href", "https://example.org/a.PDF"), ui.Attr("rel", "nofollow noopener"), ui.Text("c"))),
		ui.Button(ui.ID("on"), ui.Text("on")),
		ui.Button(ui.ID("off"), ui.Attr("disabled", ""), ui.Text("off")),
		ui.Input(ui.ID("in")),
		ui.P(ui.ID("p3"), ui.Text("three")),
	)
	sels := []string{
		"h2 + p", "h2 ~ p", "p + p", "p ~ p", "h2 + p + p", "p ~ div > span", "span + span", "span ~ a", "span + a",
		".lead ~ button", "button + button", "button ~ *", "h2 + .lead ~ div span + span", "#p2 + #box > span",
		":is(h2, p)", "div :is(span, a)", ":where(.lead, #on)", ":is(blink, p)", "p:not(:is(.lead, #p3))",
		":is(h2 + p, span + span)", "body > :is(p ~ p)",
		":enabled", "button:enabled", ":not(:enabled)", "button:not(:enabled) + input",
		"p:first-of-type", "p:last-of-type", "h2:only-of-type", "span:only-of-type", "a:only-of-type", "a:only-child",
		"p:nth-of-type(2)", "p:nth-of-type(odd)", "p:nth-last-of-type(1)", "body > :nth-last-child(2)",
		"span:nth-last-child(2n+1)", "div > :nth-last-child(-n+2)",
		`[href^="https://"]`, `[href^="HTTPS://"]`, `[href$=".pdf"]`, `[href$=".pdf" i]`,
		`[href*="example"]`, `[class*='-progressive']`, `[class*=""]`, `[rel~="noopener"]`, `[rel~="open"]`,
		`[rel~="nofollow noopener"]`, `[lang|="fr"]`, `[lang|="f"]`, `[data-kind~="wide" i]`, `[data-kind^="Note"]`,
		`[DATA-KIND="note wide" i]`,
	}
	list, _ := json.Marshal(sels)
	raw, err := chrome.Eval(d, geom.V2(400, 300),
		`return `+string(list)+`.map(s => [...document.querySelectorAll(s)].map(e => e.id).filter(id => id));`)
	if err != nil {
		t.Fatal(err)
	}
	var chromes [][]string
	if err = json.Unmarshal(raw, &chromes); err != nil {
		t.Fatalf("%v in %s", err, raw)
	}
	d.Layout(geom.V2(400, 300))
	for i, src := range sels {
		parsed, unread := css.ParseSelectors(src)
		if unread != nil {
			t.Errorf("%s: %v", src, unread)
			continue
		}
		var got []string
		d.HTML().Walk(func(n *ui.Node) bool {
			if !n.IsText() && n.ID() != "" && parsed[0].Matches(n) {
				got = append(got, n.ID())
			}
			return true
		})
		if !slices.Equal(got, chromes[i]) {
			t.Errorf("%s: ui matches %v, Chrome %v", src, got, chromes[i])
		}
	}
}
