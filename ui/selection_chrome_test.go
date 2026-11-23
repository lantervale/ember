//go:build chrome

package ui_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
	"github.com/lantervale/ember/ui/internal/dom"
	"github.com/lantervale/ember/ui/internal/style"
)

const selectionSheet = `body { margin: 8px; user-select: text; font-size: 16px; line-height: 20px; width: 300px; }
p { margin: 0 0 10px; } h2 { margin: 0 0 6px; font-size: 16px; } ul { margin: 0 0 8px; } table { border-spacing: 2px; }
.up { text-transform: uppercase; } .cap { text-transform: capitalize; } .gone { display: none; }
.ghost { visibility: hidden; } .none { user-select: none; } .all { user-select: all; } .text { user-select: text; }
.box { display: inline-block; } .row { display: flex; gap: 8px; } li::before { content: "> "; }
.pre { white-space: pre; margin: 0; font: inherit; }`

// selectionPage is a page of the text a copy must write as Chrome does,
// and the text nodes its selections end in.
func selectionPage() (*ui.Document, map[string]*ui.Node) {
	d := ui.NewDocument(ui.CSS(selectionSheet))
	texts, wanted := map[string]*ui.Node{}, map[string]string{}
	named := func(name, s string) ui.Arg {
		wanted[s] = name
		return ui.Text(s)
	}
	d.Body().Append(
		ui.H2(named("title", "A title")),
		ui.P(named("long", "The quick brown fox jumps over the lazy dog and then runs on past the end of its line"),
			ui.Br(), named("after", "after the break")),
		ui.P(ui.Text("  spaces   between\n   words  "), ui.Span(ui.Class("up"), named("shout", "shout it out")),
			ui.Text(" "), ui.Span(ui.Class("cap"), ui.Text("hello world"))),
		ui.P(named("keep", "keep "), ui.Span(ui.Class("none"), ui.Text("skip")), ui.Text(" this "),
			ui.Span(ui.Class("gone"), ui.Text("gone")), ui.Span(ui.Class("ghost"), ui.Text("ghost")), named("end", "end")),
		ui.Div(ui.Class("none"), ui.Text("none of this "), ui.Span(ui.Class("text"), ui.Text("but this")),
			ui.Span(ui.Text(" nor this"))),
		ui.P(ui.Text("an "), ui.Code(ui.Class("all"), named("all", "install --all")), ui.Text(" line")),
		ui.Ul(ui.Li(ui.Text("first")), ui.Li(ui.Text("second"), ui.Ul(ui.Li(ui.Text("nested"))))),
		ui.Table(ui.Tbody(ui.Tr(ui.Th(ui.Text("h1")), ui.Th(ui.Text("h2"))),
			ui.Tr(ui.Td(ui.Text("a1")), ui.Td(ui.Text("b1"))))),
		ui.Div(ui.Text("x "), ui.Input(ui.Attr("value", "typed")), ui.Text(" y "), ui.Button(ui.Text("btn")),
			ui.Text(" z "), ui.Textarea(ui.Text("area")), ui.Text(" w")),
		ui.Div(ui.Text("in "), ui.Span(ui.Class("box"), ui.Text("a box")), ui.Text(" line"),
			ui.Img(ui.Attr("alt", "a picture"), ui.Style("width: 10px; height: 10px")), ui.Text("tail")),
		ui.Div(ui.Class("row"), ui.Div(ui.Text("flex one")), ui.Div(ui.Text("flex two"))),
		ui.Pre(ui.Class("pre"), ui.Text("pre  two\n  lines")),
		ui.Div(ui.Text("non breaking"), ui.Div(ui.Text("a block in a div")), named("last", "the last")),
	)
	d.Body().Walk(func(n *ui.Node) bool {
		if name, ok := wanted[n.Text()]; ok && n.IsText() {
			texts[name] = n
		}
		return true
	})
	d.Layout(geom.V2(400, 800))
	return d, texts
}

func chromeSelection(t *testing.T, d *ui.Document) string {
	t.Helper()
	out, err := chrome.Eval(d, geom.V2(400, 800), `return getSelection().toString()`)
	require.NoError(t, err)
	var s string
	require.NoError(t, json.Unmarshal(out, &s))
	return s
}

func TestTheCopiedTextIsChromesSelectionToString(t *testing.T) {
	d, texts := selectionPage()
	at := func(name string, off int) ui.Position { return ui.Position{Node: texts[name], Offset: off} }
	for name, ends := range map[string][2]ui.Position{
		"a word":                 {at("long", 4), at("long", 9)},
		"across a soft wrap":     {at("long", 10), at("long", 70)},
		"across a br":            {at("long", 80), at("after", 5)},
		"title into paragraph":   {at("title", 2), at("long", 3)},
		"from a paragraph's end": {at("after", 15), at("shout", 3)},
		"backwards":              {at("end", 2), at("keep", 1)},
		"through none":           {at("keep", 0), at("end", 3)},
		"into all":               {at("end", 1), at("all", 4)},
		"to the last":            {at("all", 0), at("last", 8)},
	} {
		d.Select(ends[0], ends[1])
		assert.Equal(t, chromeSelection(t, d), d.Selection().Text, name)
	}
	require.True(t, d.SelectAll())
	assert.Equal(t, chromeSelection(t, d), d.Selection().Text, "the whole page")
}

func TestUserSelectResolvesAsChromes(t *testing.T) {
	for _, opts := range [][]ui.DocumentOption{nil, {ui.BrowserSheets()}} {
		d := ui.NewDocument(append(opts, ui.CSS(`.none { user-select: none; } .all { user-select: all; }
.text { user-select: text; } .auto { user-select: auto; }`))...)
		d.Body().Append(
			ui.P(ui.Text("plain"), ui.Span(ui.Text("span"))),
			ui.Div(ui.Class("none"), ui.Span(ui.Text("in none")), ui.Span(ui.Class("text"), ui.Text("text in none")),
				ui.Span(ui.Class("auto"), ui.Text("auto in none")), ui.Span(ui.Class("all"), ui.B(ui.Text("all in none")))),
			ui.Div(ui.Class("all"), ui.Span(ui.Text("in all")), ui.Span(ui.Class("none"), ui.Text("none in all")),
				ui.Span(ui.Class("text"), ui.I(ui.Text("text in all")))),
			ui.Div(ui.Class("text"), ui.Span(ui.Class("auto"), ui.Text("auto in text")), ui.Button(ui.Text("button")),
				ui.A(ui.Attr("href", "#"), ui.Text("link"))),
		)
		d.Layout(geom.V2(400, 400))
		// Chrome's computed value keeps auto; what a selection takes is
		// what selectAllChildren then copies of each element.
		out, err := chrome.Eval(d, geom.V2(400, 400), `
			return [...document.body.querySelectorAll("*")].map(e => {
				getSelection().selectAllChildren(e);
				return getSelection().toString() !== "";
			})`)
		require.NoError(t, err)
		var want []bool
		require.NoError(t, json.Unmarshal(out, &want))
		var got []bool
		var names []string
		d.Body().Walk(func(n *ui.Node) bool {
			if !n.IsText() && n != d.Body() {
				d.SelectContents(n)
				got = append(got, d.Selection().Text != "")
				names = append(names, n.Tag().String()+"."+strings.Join(n.Classes(), "."))
				d.ClearSelection()
			}
			return true
		})
		require.Len(t, got, len(want))
		for i := range want {
			assert.Equal(t, want[i], got[i], "%s (%d), browser sheets %v", names[i], i, opts != nil)
		}
		assert.Equal(t, style.SelectAll, dom.Data(d.Body().Children()[2]).Style.UserSelect)
	}
}

func TestTheSelectionsRectanglesAreChromesRangeRects(t *testing.T) {
	d, texts := selectionPage()
	at := func(name string, off int) ui.Position { return ui.Position{Node: texts[name], Offset: off} }
	d.Select(at("keep", 0), at("keep", 5))
	space := d.SelectionRects()[0].W
	d.Select(at("keep", 0), at("keep", 4))
	space -= d.SelectionRects()[0].W
	require.Greater(t, space, float32(2))

	for name, ends := range map[string][2]ui.Position{
		"a word":             {at("long", 4), at("long", 9)},
		"three lines":        {at("long", 10), at("long", 80)},
		"across a br":        {at("long", 80), at("after", 5)},
		"title to paragraph": {at("title", 2), at("long", 3)},
		"cased, to the next": {at("shout", 2), at("keep", 4)},
	} {
		d.Select(ends[0], ends[1])
		out, err := chrome.Eval(d, geom.V2(400, 800), `
			return [...getSelection().getRangeAt(0).getClientRects()].map(r => [r.x, r.y, r.width, r.height])`)
		require.NoError(t, err)
		var all, want [][4]float32
		require.NoError(t, json.Unmarshal(out, &all))
		// Chrome gives an element wholly in the range a rectangle of its
		// own beside its text's, and a br an empty one.
		for _, r := range all {
			if r[2] > 0 && (len(want) == 0 || want[len(want)-1] != r) {
				want = append(want, r)
			}
		}
		got := d.SelectionRects()
		if !assert.Len(t, got, len(want), "%s: ember %v, Chrome %v", name, got, want) {
			continue
		}
		for i, w := range want {
			g := got[i]
			assert.InDelta(t, w[0], g.X, 0.51, "%s: rect %d starts where Chrome's does", name, i)
			assert.InDelta(t, w[1]+w[3]/2, g.Y+g.H/2, 1.01, "%s: rect %d on Chrome's line", name, i)
			assert.GreaterOrEqual(t, g.H, w[3]-0.01, "%s: rect %d as high as its line, Chrome's as its glyphs", name, i)
			if extra := g.W - w[2]; extra > 0.51 {
				assert.InDelta(t, space, extra, 0.51,
					"%s: rect %d is Chrome's, and a space more where the selection goes past its line's end", name, i)
			} else {
				assert.InDelta(t, w[2], g.W, 0.51, "%s: rect %d as wide as Chrome's", name, i)
			}
		}
	}
}

// TestARightToLeftSelectionsRectanglesAreChromes selects inside text
// shown right to left, Hebrew in a font both draw alike, and compares
// the rectangles with Chrome's range rects, line by line from the left.
func TestARightToLeftSelectionsRectanglesAreChromes(t *testing.T) {
	font := hebrewFont(t)
	d := ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
		ui.CSS(selectionSheet+`@font-face { font-family: Hebrew; src: url(hebrew.ttf); } body { font-family: Hebrew; }`))
	d.Body().Append(ui.P(ui.Attr("dir", "rtl"), ui.Text("אבג דהו זחט יכל מנס עפצ קרש תאב גדה וזח טיכ")),
		ui.P(ui.Attr("dir", "rtl"), ui.Text("שלום abc def עולם")),
		ui.P(ui.Text("abc "), ui.Span(ui.Attr("dir", "rtl"), ui.Text("שלום עולם")), ui.Text(" def")))
	size := geom.V2(400, 800)
	landed(t, d, size)
	var texts []*ui.Node
	d.Body().Walk(func(n *ui.Node) bool {
		if n.IsText() && strings.ContainsFunc(n.Text(), func(r rune) bool { return r >= 0x0590 }) {
			texts = append(texts, n)
		}
		return true
	})
	hebrew, mixed, third := texts[0], texts[1], texts[2]
	for name, ends := range map[string][2]ui.Position{
		"inside a word":     {{Node: hebrew, Offset: 1}, {Node: hebrew, Offset: 3}},
		"across lines":      {{Node: hebrew, Offset: 5}, {Node: hebrew, Offset: 30}},
		"hebrew into latin": {{Node: mixed, Offset: 2}, {Node: mixed, Offset: 7}},
		"latin":             {{Node: mixed, Offset: 6}, {Node: mixed, Offset: 11}},
		"an rtl span":       {{Node: third, Offset: 1}, {Node: third, Offset: 7}},
	} {
		d.Select(ends[0], ends[1])
		out, err := chrome.Eval(d, size, `
			return [...getSelection().getRangeAt(0).getClientRects()].map(r => [r.x, r.y, r.width, r.height])`)
		require.NoError(t, err)
		var all [][4]float32
		require.NoError(t, json.Unmarshal(out, &all))
		var want []geom.Rect
		for _, r := range all {
			if r[2] > 0 {
				want = append(want, geom.Rect{X: r[0], Y: r[1], W: r[2], H: r[3]})
			}
		}
		got := d.SelectionRects()
		byLine := func(rs []geom.Rect) []geom.Rect {
			out := slices.Clone(rs)
			slices.SortFunc(out, func(a, b geom.Rect) int {
				if d := a.Y + a.H/2 - b.Y - b.H/2; d < -1 || d > 1 {
					return int(d)
				}
				return int(a.X - b.X)
			})
			return mergedRects(out)
		}
		g, w := byLine(got), byLine(want)
		if !assert.Len(t, g, len(w), "%s: ember %v, Chrome %v", name, g, w) {
			continue
		}
		for i := range w {
			assert.InDelta(t, w[i].X, g[i].X, 0.51, "%s: rect %d starts where Chrome's does: %v %v", name, i, g, w)
			assert.InDelta(t, w[i].Y+w[i].H/2, g[i].Y+g[i].H/2, 1.01, "%s: rect %d on Chrome's line", name, i)
		}
	}
}

// mergedRects joins rectangles of a line that touch, as Chrome gives a
// rectangle per text box where ember gives one per fragment.
func mergedRects(rs []geom.Rect) []geom.Rect {
	var out []geom.Rect
	for _, r := range rs {
		if n := len(out); n > 0 && abs32(out[n-1].Y-r.Y) < 1 && abs32(out[n-1].X+out[n-1].W-r.X) < 0.6 {
			out[n-1].W = r.X + r.W - out[n-1].X
			continue
		}
		out = append(out, r)
	}
	return out
}

func abs32(v float32) float32 { return max(v, -v) }
