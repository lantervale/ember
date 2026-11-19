//go:build chrome

package layout_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// textTags are the text elements of pages, as a JavaScript alternation.
const textTags = "ul|ol|li|menu|dl|dt|dd|h5|h6|blockquote|figure|figcaption|form|address|hgroup|search|center|" +
	"fieldset|legend|a|b|i|u|s|small|sub|sup|big|abbr|cite|mark|var|dfn|del|ins|q|br"

// ownDefaults drops from the page every rule of ember's user-agent sheet that names one of
// tags, so Chrome lays them out by its own, and returns each element's box.
func ownDefaults(tags string) string {
	return `
const tags = /(^|[\s,>+~(])(` + tags + `)(?![\w-])/;
for (const sheet of document.styleSheets) {
	for (const layer of sheet.cssRules) {
		if (!(layer instanceof CSSLayerBlockRule)) continue;
		for (let i = layer.cssRules.length - 1; i >= 0; i--) {
			if (tags.test(layer.cssRules[i].selectorText || "")) layer.deleteRule(i);
		}
	}
}
return [document.documentElement, document.body, ...document.body.querySelectorAll("*:not([data-ember-implied])")]
	.map(e => { const r = e.getBoundingClientRect(); return [r.x, r.y, r.width, r.height]; });`
}

// matchesChromesDefaults compares d's boxes with Chrome's where Chrome styles the text
// elements by its own user-agent sheet instead of ember's.
func matchesChromesDefaults(t *testing.T, d *ui.Document, size geom.Vec2) {
	t.Helper()
	matchesChromesOwn(t, d, size, textTags)
}

// matchesChromesOwn compares d's boxes with Chrome's where Chrome styles tags, a
// JavaScript alternation, by its own user-agent sheet instead of ember's.
func matchesChromesOwn(t *testing.T, d *ui.Document, size geom.Vec2, tags string) {
	t.Helper()
	out, err := chrome.Eval(d, size, ownDefaults(tags))
	require.NoError(t, err)
	var want [][4]float32
	require.NoError(t, json.Unmarshal(out, &want))
	d.Layout(size)
	var got []*ui.Node
	d.HTML().Walk(func(n *ui.Node) bool {
		if !n.IsText() {
			got = append(got, n)
		}
		return true
	})
	require.Len(t, got, len(want))
	for i, n := range got {
		g, w := n.Box(), geom.Rect{X: want[i][0], Y: want[i][1], W: want[i][2], H: want[i][3]}
		delta := math.Max(math.Max(math.Abs(float64(g.X-w.X)), math.Abs(float64(g.Y-w.Y))),
			math.Max(math.Abs(float64(g.X+g.W-w.X-w.W)), math.Abs(float64(g.Y+g.H-w.Y-w.H))))
		if delta > float64(chrome.Tolerance) {
			t.Errorf("%d <%s> %q: ember %v, Chrome %v", i, n.Tag(), n.Text(), g, w)
		}
	}
}

func TestTheTextElementsDefaultsMatchChromes(t *testing.T) {
	item := func(s string) *ui.Node { return ui.Li(ui.Text(s)) }
	cases := map[string]*ui.Document{
		"lists": styled("",
			ui.Ul(item("Rope"), item("Lantern oil, two flasks, and a wick that lasts the whole way down"),
				ui.Li(ui.Text("Food"), ui.Ul(item("Bread"), ui.Li(ui.Text("Cheese"), ui.Ul(item("Hard")))))),
			ui.Ol(item("Light the lantern"), item("Open the gate"), ui.Li(ui.P(ui.Text("Go down")))),
			ui.Ol(ui.Attr("start", "9"), item("Nine"), item("Ten")),
			ui.Menu(item("Cut"), item("Paste")),
			ui.Dl(ui.Dt(ui.Text("Lantern")), ui.Dd(ui.Text("Lights the way")), ui.Dt(ui.Text("Rope")),
				ui.Dd(ui.Text("Fifty feet"))),
		),
		"blocks": styled("",
			ui.H5(ui.Text("The fifth")), ui.H6(ui.Text("The sixth")),
			ui.Blockquote(ui.Text(sentence)),
			ui.Figure(ui.Div(ui.Style("width: 80px; height: 40px; background: #888")), ui.Figcaption(ui.Text("A grey box"))),
			ui.Address(ui.Text("The gatehouse")), ui.Hgroup(ui.H5(ui.Text("Title")), ui.P(ui.Text("Subtitle"))),
			ui.Form(ui.P(ui.Text("A form"))), ui.Search(ui.Text("A search")),
			ui.Center(ui.Text("Centred"), ui.P(ui.Text("and its paragraph"))),
		),
		"inline text": styled("",
			ui.P(ui.A(ui.Attr("href", "/x"), ui.Text("a link")), ui.Text(" "), ui.B(ui.Text("bold")), ui.Text(" "),
				ui.I(ui.Text("italic")), ui.Text(" "), ui.U(ui.Text("under")), ui.Text(" "), ui.S(ui.Text("struck")),
				ui.Text(" "), ui.Small(ui.Text("small")), ui.Text(" "), ui.Big(ui.Text("big")), ui.Text(" "),
				ui.Abbr(ui.Attr("title", "Hit points"), ui.Text("HP")), ui.Text(" "), ui.Cite(ui.Text("cite")),
				ui.Text(" "), ui.Mark(ui.Text("mark")), ui.Text(" "), ui.Var(ui.Text("x")), ui.Text(" "),
				ui.Dfn(ui.Text("dfn")), ui.Text(" "), ui.Del(ui.Text("del")), ui.Text(" "), ui.Ins(ui.Text("ins")),
				ui.Text(" "), ui.Time(ui.Text("noon")), ui.Text(" "), ui.Data(ui.Text("12")), ui.Text(" "),
				ui.Bdi(ui.Text("bdi")), ui.Text(" "), ui.Font(ui.Text("font"))),
			ui.P(ui.Text("One line"), ui.Br(), ui.Text("and the next"), ui.Br(), ui.Br(), ui.Text("after a gap")),
		),
		"a line break ending its block": styled("",
			ui.Div(ui.Br()), ui.Div(ui.Text("One line"), ui.Br()), ui.Div(ui.Text("One line"), ui.Br(), ui.Br()),
			ui.Div(ui.Br(), ui.Div(ui.Text("a block after the break"))),
			ui.Div(ui.B(ui.Text("in bold"), ui.Br())), ui.Div(ui.Text("before "), ui.I(ui.Br())),
			ui.Div(ui.Text("One line"), ui.Br(), ui.Span(ui.Style("padding: 0 4px"))),
		),
		"sub and sup": styled("", ui.P(ui.Text("H"), ui.Sub(ui.Text("2")), ui.Text("O and x"), ui.Sup(ui.Text("2")))),
		"a quotation": styled("", ui.P(ui.Text("They said "), ui.Q(ui.Text("mind the gap")), ui.Text("."))),
		"a fieldset":  styled("", ui.Fieldset(ui.Legend(ui.Text("Sound")), ui.P(ui.Text("Volume")))),
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) { matchesChromesDefaults(t, d, geom.V2(400, 600)) })
	}
}
