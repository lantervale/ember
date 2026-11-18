//go:build chrome

package css_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/css"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

func TestChromeReadsThePseudoElementsOfTheTable(t *testing.T) {
	table := css.PseudoElements()
	var sels []string
	for _, pe := range table {
		sels = append(sels, pe.Selector)
	}
	// Names beside the table's, to see Chrome refuse what the table leaves out.
	sels = append(sels, "::cue-region", "::nope", "::interest-hint", "::-moz-selection")
	names, _ := json.Marshal(sels)
	var got map[string]bool
	ask(t, geom.V2(400, 300), fmt.Sprintf(
		`return Object.fromEntries(%s.map(s => [s, CSS.supports("selector(" + s + ")")]));`, names), &got)
	for _, pe := range table {
		if !got[pe.Selector] {
			t.Errorf("Chrome doesn't read %s, which the table lists", pe.Selector)
		}
		_, err := css.ParseSelectors("div" + pe.Selector)
		if (err != nil) != pe.Refused {
			t.Errorf("div%s: %v, refused in the table: %v", pe.Selector, err, pe.Refused)
		}
	}
	for _, s := range sels[len(table):] {
		if got[s] {
			t.Errorf("Chrome reads %s, which the table leaves out", s)
		}
		if _, err := css.ParseSelectors("div" + s); err == nil {
			t.Errorf("div%s: ui reads what Chrome doesn't", s)
		}
	}
}

// Where a pseudo-element may stand in a selector, as Chrome has it. The
// forms Chrome reads that ui refuses are the last ones: a ::marker of a
// generated box, and the few pseudo-classes of a pseudo-element.
func TestPseudoElementsStandWhereChromeTakesThem(t *testing.T) {
	sels := []string{"div::before", "div:before", "div:after", "div::BEFORE", "::before", "*::after",
		"div:hover::before", "ul li:nth-child(2)::before", "div > p::after", "li::marker", "div.a#b::before",
		"div::before:hover", "div::before:not(.x)", "div::before.x", "div::before span", "div::before > span",
		"div :: before", "div::after::before", "li::marker::before", "div::before:first-child", "div:marker",
		"div:placeholder", "div:selection", "div:backdrop", "div::before::before", "input::placeholder:hover",
		"div::marker:hover", "div::before:active",
		"div::before::marker", "div::after::marker", "div::selection:window-inactive"}
	const onlyChrome = 3
	names, _ := json.Marshal(sels)
	var got map[string]bool
	ask(t, geom.V2(400, 300), fmt.Sprintf(
		`return Object.fromEntries(%s.map(s => [s, CSS.supports("selector(" + s + ")")]));`, names), &got)
	for i, s := range sels {
		_, err := css.ParseSelectors(s)
		ours := err == nil
		want := got[s]
		if i >= len(sels)-onlyChrome {
			if !want {
				t.Errorf("%s: Chrome no longer reads it", s)
			}
			want = false
		}
		if ours != want {
			t.Errorf("%s: ui reads it: %v (%v), Chrome: %v", s, ours, err, got[s])
		}
	}
}

func TestPseudoElementStylesMatchChrome(t *testing.T) {
	d := ui.NewDocument(ui.CSS(`
body { color: #123456; font-size: 20px; }
.card { color: #a00; position: relative; letter-spacing: 1px; word-spacing: 3px; }
.card::before { content: "a"; font-size: 150%; font-weight: 700; padding-left: 0.5em; background-color: #0a0; }
.card::after { content: ""; position: absolute; display: inline; opacity: 0.5; margin-top: 2em; color: #00a; }
.card:first-child::before { font-style: italic; }
.row { display: flex; }
.row::before { content: "x"; }
.row::after { content: "y"; display: inline-flex; float: left; }
p, p::before { color: #0aa; content: "both"; }
p::after { content: none; color: #f00; }
ol { color: #333; }
li::marker { color: #b0b; font-size: 12px; font-weight: 700; letter-spacing: 2px; word-spacing: 0.5em;
  padding-left: 9px; background-color: #ff0; }
li.plain { font-style: italic; }
.over::before { content: "1"; color: #111; }
#over.over::before { color: #222; }
div.over::before { color: #333 !important; }
`))
	d.Body().Append(
		ui.Div(ui.Class("card"), ui.Text("one")), ui.Div(ui.Class("card"), ui.Text("two")),
		ui.Div(ui.Class("row"), ui.Text("row")), ui.P(ui.Text("p")),
		ui.Ol(ui.Li(ui.Text("styled")), ui.Li(ui.Class("plain"), ui.Text("too"))),
		ui.Div(ui.ID("over"), ui.Class("over"), ui.Text("over")),
	)
	props := []string{"color", "font-size", "font-weight", "font-style", "letter-spacing", "word-spacing", "display",
		"position", "padding-left", "margin-top", "opacity", "background-color"}
	markerProps := []string{"color", "font-size", "font-weight", "font-style", "letter-spacing", "word-spacing"}
	names, _ := json.Marshal(props)
	raw, err := chrome.Eval(d, geom.V2(800, 600), fmt.Sprintf(`
return Array.from(document.body.querySelectorAll("*")).map(e => Object.fromEntries(
	["::before", "::after", "::marker"].map(pe => {
		const s = getComputedStyle(e, pe);
		return [pe, Object.fromEntries(%s.concat(["content"]).map(p => [p, s.getPropertyValue(p)]))];
	})));`, names))
	if err != nil {
		t.Fatal(err)
	}
	var want []map[string]map[string]string
	if err = json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	d.Layout(geom.V2(800, 600))
	var els []*dom.Node
	d.Body().Walk(func(n *dom.Node) bool {
		if !n.IsText() && n != d.Body() {
			els = append(els, n)
		}
		return true
	})
	if len(els) != len(want) {
		t.Fatalf("ember has %d elements, Chrome %d", len(els), len(want))
	}
	checked := 0
	check := func(n *dom.Node, pe string, st *style.Computed, chromes map[string]string, props []string) {
		for _, p := range props {
			if ours := compared[p](st); !same(p, ours, chromes[p]) {
				t.Errorf("%s%s %s: ember %q, Chrome %q", where(n), pe, p, ours, chromes[p])
			}
			checked++
		}
	}
	for i, n := range els {
		data := dom.Data(n)
		for pe, g := range map[string]*dom.Node{"::before": data.Before, "::after": data.After} {
			content := want[i][pe]["content"]
			if (g != nil) != (content != "none" && content != "normal") {
				t.Errorf("%s%s: ember has a box: %v, Chrome's content is %s", where(n), pe, g != nil, content)
				continue
			}
			if g != nil {
				check(n, pe, dom.Data(g).Style, want[i][pe], props)
			}
		}
		if m := data.Marker; m != nil {
			check(n, "::marker", m, want[i]["::marker"], markerProps)
		} else if data.Style.ListItem {
			check(n, "::marker", data.Style, want[i]["::marker"], markerProps)
		}
	}
	if checked < 90 {
		t.Errorf("%d values compared", checked)
	}
	t.Logf("%d values compared", checked)
}
