//go:build chrome

package ui_test

import (
	"fmt"
	"testing"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/ui"
	"github.com/lantervale/ember/ui/internal/chrome"
)

// TestChromeComputesDirection has Chrome compute direction and
// unicode-bidi from the dir attribute, bdo, bdi, dir="auto" read from
// the text, inheritance and declarations, and ember match each.
func TestChromeComputesDirection(t *testing.T) {
	doc := ui.NewDocument(ui.CSS(".r { direction: rtl } .l { direction: ltr } .e { unicode-bidi: embed } " +
		".o { unicode-bidi: bidi-override } .i { direction: inherit } .n { direction: initial }"))
	rtl := ui.Attr("dir", "rtl")
	doc.Body().Append(
		ui.Div(rtl, ui.P(ui.Span(ui.Text("a"))), ui.Div(ui.Attr("dir", "ltr")), ui.Bdi(ui.Text("abc")),
			ui.Span(ui.Class("i")), ui.Span(ui.Class("n"))),
		ui.Div(ui.Attr("dir", "auto"), ui.Text("שלום")),
		ui.Div(ui.Attr("dir", "auto"), ui.Text("1, hello")),
		ui.P(ui.Attr("dir", "RTL")),
		ui.Div(ui.Attr("dir", "sideways"), ui.Span()),
		ui.Bdo(rtl), ui.Bdo(), ui.Bdi(ui.Text("עברית")), ui.Bdi(),
		ui.Span(ui.Class("r"), ui.Span(ui.Class("e")), ui.Span(ui.Class("o"), ui.Class("l"))),
		ui.Textarea(ui.Attr("dir", "auto")), ui.Pre(ui.Attr("dir", "AUTO")),
		ui.Section(), ui.Article(), ui.Footer(), ui.Ul(ui.Li()),
		ui.Header(), ui.Nav(), ui.Main(), ui.Aside(), ui.H1(), ui.H6(), ui.Details(ui.Summary()), ui.Dialog(),
		ui.Hr(), ui.Ol(), ui.Menu(), ui.Dl(ui.Dt(), ui.Dd()), ui.Blockquote(), ui.Figure(ui.Figcaption()),
		ui.Form(), ui.Address(), ui.Hgroup(), ui.Search(), ui.Center(), ui.Fieldset(ui.Legend()),
		ui.Table(ui.Caption(), ui.Colgroup(ui.Col()), ui.Thead(ui.Tr(ui.Th())), ui.Tbody(ui.Tr(ui.Td())), ui.Tfoot()),
		ui.Button(), ui.Input(), ui.Select(), ui.Img(), ui.Label(), ui.A(), ui.Em(), ui.Progress(), ui.Meter(),
	)
	size := geom.V2(400, 300)
	doc.Layout(size)
	styles, err := chrome.Styles(doc, size, []string{"direction", "unicode-bidi"})
	if err != nil {
		t.Fatal(err)
	}
	nodes := elementsOf(doc)
	if len(nodes) != len(styles) {
		t.Fatalf("%d elements, Chrome styled %d", len(nodes), len(styles))
	}
	for i, n := range nodes {
		dir, bidi := ui.Bidi(n)
		if dir != styles[i]["direction"] || bidi != styles[i]["unicode-bidi"] {
			t.Errorf("element %d (%s): %s %s, Chrome %s %s", i, n.Tag(), dir, bidi, styles[i]["direction"],
				styles[i]["unicode-bidi"])
		}
	}
}

// rtlCases are pages laid out right to left, compared box by box with
// Chrome's.
//
//nolint:gochecknoglobals // a test's table
var rtlCases = map[string]func() []*ui.Node{
	"logical margins, paddings, borders and insets": func() []*ui.Node {
		box := func(dir, style string) *ui.Node {
			return ui.Div(ui.Attr("dir", dir), ui.Style("position: relative; width: 300px; height: 40px"),
				ui.Div(ui.Style("width: 100px; height: 10px; "+style)))
		}
		var out []*ui.Node
		for _, dir := range []string{"ltr", "rtl"} {
			out = append(out,
				box(dir, "margin-inline-start: 30px"),
				box(dir, "margin-inline: 10px 50px; padding-inline-end: 7px"),
				box(dir, "border-inline-start: 6px solid; padding-inline-start: 4px"),
				box(dir, "border-style: solid; border-inline-width: 2px 9px"),
				box(dir, "position: absolute; inset-inline-start: 12px"),
				box(dir, "position: absolute; inset-inline: 5px auto"),
				box(dir, "margin-left: 20px; margin-inline-start: 40px"),
				box(dir, "margin-inline-start: 40px; margin-left: 20px"),
				ui.Div(ui.Attr("dir", dir), ui.Style("width: 300px"),
					ui.Div(ui.Style("float: inline-start; width: 50px; height: 10px")),
					ui.Div(ui.Style("float: inline-end; width: 60px; height: 10px")),
					ui.Div(ui.Style("clear: inline-end; height: 5px"))),
			)
		}
		return out
	},
	"lines of a right-to-left paragraph": func() []*ui.Node {
		words := func() []ui.Arg {
			return []ui.Arg{ui.Span(ui.Text("Lead")), ui.Text(" words "), ui.Span(ui.Text("then")),
				ui.Text(" a "), ui.Span(ui.Style("padding: 0 10px 0 4px; border-left: 3px solid; margin-right: 6px"),
					ui.Text("padded")), ui.Text(" span that wraps "), ui.Span(ui.Text("twice over."))}
		}
		var out []*ui.Node
		for _, align := range []string{"start", "end", "left", "right", "center", "justify"} {
			out = append(out, ui.P(append(words(), ui.Attr("dir", "rtl"),
				ui.Style("width: 140px; text-align: "+align))...))
		}
		return append(out,
			ui.P(append(words(), ui.Attr("dir", "rtl"), ui.Style("width: 160px; text-indent: 20px"))...),
			ui.P(append(words(), ui.Style("width: 160px; direction: rtl; unicode-bidi: plaintext"))...),
			ui.Div(ui.Attr("dir", "rtl"), ui.Style("width: 150px"), ui.Span(ui.Text("one")), ui.Br(),
				ui.Span(ui.Text("two")), ui.Text(" "), ui.Span(ui.Text("three"))),
		)
	},
	"flex and grid from the right": func() []*ui.Node {
		item := func(style string) *ui.Node { return ui.Div(ui.Style("height: 10px; " + style)) }
		rtl := ui.Attr("dir", "rtl")
		return []*ui.Node{
			ui.Div(rtl, ui.Style("display: flex; width: 300px; gap: 5px; padding: 0 3px 0 11px; border-left: 4px solid"),
				item("width: 40px; margin-left: 7px"), item("width: 50px; margin-inline-start: 9px"), item("flex: 1")),
			ui.Div(rtl, ui.Style("display: flex; width: 300px; justify-content: flex-end"),
				item("width: 40px"), item("width: 50px")),
			ui.Div(rtl, ui.Style("display: flex; width: 300px; justify-content: space-between; flex-direction: row-reverse"),
				item("width: 40px"), item("width: 50px"), item("width: 60px")),
			ui.Div(rtl, ui.Style("display: flex; width: 200px; flex-wrap: wrap"),
				item("width: 80px"), item("width: 90px"), item("width: 70px; margin-left: auto")),
			ui.Div(rtl, ui.Style("display: flex; flex-direction: column; width: 200px; align-items: flex-start"),
				item("width: 80px"), item("width: 90px; align-self: flex-end"), item("width: 70px; align-self: center")),
			ui.Div(rtl, ui.Style("display: grid; width: 300px; grid-template-columns: 50px 1fr 70px; gap: 4px 6px"),
				item(""), item(""), item(""), item("grid-column: 1 / 3"), item("justify-self: start; width: 20px")),
			ui.Div(rtl, ui.Style("display: grid; width: 300px; grid-template-columns: repeat(3, 60px); "+
				"justify-content: start; padding-right: 13px"), item(""), item("justify-self: end; width: 10px")),
			ui.Div(rtl, ui.Style("position: relative; display: flex; width: 300px; height: 30px"),
				item("width: 30px"), item("position: absolute; width: 20px")),
		}
	},
	"tables and lists from the right": func() []*ui.Node {
		cell := func(w string, args ...ui.Arg) *ui.Node {
			return ui.Td(append(args, ui.Div(ui.Style("height: 10px; width: "+w)))...)
		}
		return []*ui.Node{
			ui.Table(ui.Attr("dir", "rtl"), ui.Style("border-spacing: 3px; padding-left: 9px; border: 1px solid"),
				ui.Caption(ui.Text("cap")),
				ui.Colgroup(ui.Col(), ui.Col(ui.Attr("span", "2"))),
				ui.Tbody(ui.Tr(cell("20px"), cell("30px"), cell("40px")),
					ui.Tr(cell("20px", ui.Attr("colspan", "2")), cell("10px"))),
				ui.Tr(cell("5px"), cell("6px"), cell("7px"))),
			ui.Table(ui.Attr("dir", "rtl"), ui.Style("width: 300px"),
				ui.Tr(ui.Td(ui.Text("one")), ui.Td(ui.Style("width: 100px"), ui.Text("two")))),
			ui.Ul(ui.Attr("dir", "rtl"), ui.Li(ui.Text("one")), ui.Li(ui.Text("two"))),
			ui.Ol(ui.Attr("dir", "rtl"), ui.Li(ui.Text("one")), ui.Dl(ui.Dd(ui.Text("two")))),
		}
	},
	"positioned boxes and scrollers": func() []*ui.Node {
		rtl := ui.Attr("dir", "rtl")
		return []*ui.Node{
			ui.Div(rtl, ui.Style("position: relative; width: 300px; height: 40px; padding-left: 5px"),
				ui.Div(ui.Style("height: 10px")),
				ui.Div(ui.Style("position: absolute; width: 20px; height: 10px")),
				ui.Span(ui.Text("ab "), ui.Span(ui.Style("position: absolute; width: 8px; height: 8px")))),
			ui.Div(rtl, ui.Style("overflow-y: scroll; width: 200px; height: 50px"),
				ui.Div(ui.Style("height: 80px")), ui.Div(ui.Style("width: 50px; height: 10px"))),
			ui.Div(rtl, ui.Style("overflow-y: scroll; width: 200px; height: 50px; display: flex"),
				ui.Div(ui.Style("width: 50px; height: 80px"))),
		}
	},
	"atoms and spans reordered": func() []*ui.Node {
		block := func(w string) *ui.Node {
			return ui.Span(ui.Style("display: inline-block; height: 10px; width: " + w))
		}
		return []*ui.Node{
			ui.Div(ui.Attr("dir", "rtl"), ui.Style("width: 300px"), block("20px"), ui.Text(" "), block("30px"),
				ui.Text(" "), block("40px")),
			ui.Div(ui.Style("width: 300px"), ui.Text("ab "),
				ui.Span(ui.Attr("dir", "rtl"), block("20px"), ui.Text(" "), block("30px")), ui.Text(" cd")),
			ui.Div(ui.Style("width: 300px"), ui.Bdo(ui.Attr("dir", "rtl"), ui.Span(ui.Text("abc")),
				ui.Span(ui.Text("def")), ui.Text(" "), ui.Span(ui.Style("padding-left: 5px"), ui.Text("gh")))),
			ui.Div(ui.Style("width: 300px"), ui.Span(ui.Text("one ")),
				ui.Span(ui.Style("direction: rtl; unicode-bidi: bidi-override"), ui.Span(ui.Text("two")),
					ui.Span(ui.Text("three"))), ui.Span(ui.Text(" four"))),
			ui.Div(ui.Attr("dir", "rtl"), ui.Style("width: 300px"), ui.Span(ui.Text("one ")),
				ui.Span(ui.Style("unicode-bidi: isolate; direction: ltr"), block("10px"), block("20px")),
				ui.Span(ui.Style("unicode-bidi: embed; direction: ltr"), block("12px"), block("22px"))),
			ui.Div(ui.Attr("dir", "rtl"), ui.Style("width: 300px"),
				ui.Span(ui.Style("padding-left: 10px; padding-right: 20px; border: 2px solid"), ui.Text("abc")),
				ui.Text(" "), ui.Span(ui.Style("direction: ltr; padding-left: 7px"), block("15px"))),
		}
	},
}

// TestChromeOrdersMixedTextAsChrome has Chrome and ember lay out
// Hebrew among Latin, numbers and brackets in a font both draw alike,
// each word in a span, and compares where each stands.
func TestChromeOrdersMixedTextAsChrome(t *testing.T) {
	font := hebrewFont(t)
	doc := ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
		ui.CSS(`@font-face { font-family: Hebrew; src: url(hebrew.ttf); } body { font-family: Hebrew; }`))
	words := func(ws ...string) []ui.Arg {
		var out []ui.Arg
		for i, w := range ws {
			if i > 0 {
				out = append(out, ui.Text(" "))
			}
			out = append(out, ui.Span(ui.Text(w)))
		}
		return out
	}
	p := func(dir string, ws ...string) *ui.Node {
		return ui.P(append(words(ws...), ui.Attr("dir", dir), ui.Style("width: 220px"))...)
	}
	doc.Body().Append(
		p("rtl", "שלום", "עולם"),
		p("ltr", "abc", "שלום", "עולם", "def"),
		p("rtl", "abc", "שלום", "123", "עולם", "def"),
		p("ltr", "שלום", "123", "45", "abc"),
		p("rtl", "(abc)", "שלום", "[עולם]", "1,5%"),
		p("auto", "שלום", "abc"),
		p("rtl", "אבג", "דהו", "זחט", "יכל", "מנס", "עפצ", "קרש", "abc", "def", "ghi"),
		ui.P(ui.Attr("dir", "rtl"), ui.Style("width: 220px"), ui.Text("שלום "),
			ui.Span(ui.Text("abc"), ui.Span(ui.Text("עולם")), ui.Text("def")), ui.Text(" xyz")),
		ui.P(ui.Style("width: 220px"), ui.Text("abc "),
			ui.Span(ui.Attr("dir", "rtl"), ui.Span(ui.Text("12")), ui.Text(" "), ui.Span(ui.Text("שלום"))),
			ui.Text(" def")),
	)
	size := geom.V2(800, 600)
	landed(t, doc, size)
	r := chrome.Match(t, doc, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

func TestChromeLaysOutRightToLeft(t *testing.T) {
	for name, nodes := range rtlCases {
		t.Run(name, func(t *testing.T) {
			doc := ui.NewDocument()
			doc.Body().Append(nodes()...)
			r := chrome.Match(t, doc, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// TestFirstLettersAndLinesRightToLeftMatchChrome has ::first-letter and
// ::first-line style paragraphs of direction rtl, of Latin and of Hebrew
// in a font both draw alike, a first letter floated to the start.
func TestFirstLettersAndLinesRightToLeftMatchChrome(t *testing.T) {
	font := hebrewFont(t)
	doc := ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }), ui.CSS(`
@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
body { font-family: Hebrew; margin: 8px; font-size: 16px; width: 300px; }
p { margin: 0 0 10px; }
.big::first-letter { font-size: 200%; }
.drop::first-letter { float: inline-start; font-size: 3em; line-height: 1; margin-inline-end: 6px; }
.line::first-line { font-size: 22px; letter-spacing: 1px; }
`))
	para := func(class, text string) *ui.Node {
		return ui.P(append([]ui.Arg{ui.Attr("dir", "rtl"), ui.Class(class)}, words(text)...)...)
	}
	latin := "Lines from the right, every word in a span of its own, wrapping over three lines or so."
	hebrew := "אבג דהו זחט יכל מנס עפצ קרש תאב גדה וזח טיכ למנ סעפ צקר"
	doc.Body().Append(
		para("big", latin), para("drop", latin+" "+latin), para("line", latin),
		para("big", hebrew), para("drop", hebrew+" "+hebrew), para("line", hebrew+" 123 abc"),
	)
	size := geom.V2(400, 900)
	landed(t, doc, size)
	r := chrome.Match(t, doc, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}

// TestChromeScrollsRightToLeft has scrollers of direction rtl start at
// their right end and scroll left by negative offsets, as Chrome's
// scrollLeft, each compared box by box at its start and scrolled.
func TestChromeScrollsRightToLeft(t *testing.T) {
	scrollers := func() (*ui.Document, []*ui.Node) {
		rtl := ui.Attr("dir", "rtl")
		box := func(style string) *ui.Node { return ui.Div(ui.Style("height: 20px; " + style)) }
		s := []*ui.Node{
			ui.Div(rtl, ui.Style("overflow-x: auto; width: 200px; height: 60px"), box("width: 350px"), box("width: 50px")),
			ui.Div(rtl, ui.Style("overflow: scroll; width: 200px; height: 60px; padding: 4px 10px 6px 7px; border: 3px solid"),
				box("width: 300px; margin-left: 15px"), box("height: 90px")),
			ui.Div(rtl, ui.Style("overflow-x: auto; width: 150px; white-space: nowrap; padding: 0 5px"),
				ui.Span(ui.Text("a line of text far too long for its box")), ui.Text(" and more")),
			ui.Div(rtl, ui.Style("overflow-x: auto; width: 200px; display: flex; gap: 10px"),
				box("flex: none; width: 120px"), box("flex: none; width: 140px")),
			ui.Div(rtl, ui.Style("overflow-x: auto; width: 200px; display: grid; grid-template-columns: 130px 130px"),
				box(""), box("")),
			ui.Div(rtl, ui.Style("overflow: auto; width: 200px; height: 50px; position: relative"),
				box("width: 100px"), box("position: absolute; top: 0; left: -120px; width: 30px")),
			ui.Div(rtl, ui.Style("overflow-x: auto; width: 200px"), box("width: 100px")),
		}
		d := ui.NewDocument()
		d.Body().Append(s...)
		return d, s
	}
	for _, by := range []float32{0, -40, -1000, 30} {
		t.Run(fmt.Sprint(by), func(t *testing.T) {
			d, s := scrollers()
			d.Layout(geom.V2(800, 600))
			for _, n := range s {
				d.SetScroll(n, geom.V2(by, 0))
			}
			r := chrome.Match(t, d, geom.V2(800, 600))
			t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
		})
	}
}

// TestChromeLaysOutControlsRightToLeft has text fields, a textarea,
// selects and buttons of direction rtl hold their text from the right,
// a select its arrow on its left, compared box by box with Chrome's in a
// font both draw alike.
func TestChromeLaysOutControlsRightToLeft(t *testing.T) {
	font := hebrewFont(t)
	doc := ui.NewDocument(ui.FetchFonts(func(_, _ string) ([]byte, error) { return font, nil }),
		ui.CSS(`@font-face { font-family: Hebrew; src: url(hebrew.ttf); }
body { font-family: Hebrew; } input, select, textarea, button { width: 160px; display: block; margin-bottom: 4px; }`))
	rtl := ui.Attr("dir", "rtl")
	doc.Body().Append(
		ui.Input(rtl, ui.Attr("value", "שלום abc")),
		ui.Input(rtl, ui.Attr("placeholder", "abc שלום")),
		ui.Input(rtl, ui.Attr("value", "a value far too long for the width of its field")),
		ui.Input(ui.Attr("value", "שלום עולם"), ui.Style("direction: rtl; text-align: left")),
		ui.Textarea(rtl, ui.Text("שלום\nabc def")),
		ui.Select(rtl, ui.Option(ui.Text("שלום"))),
		ui.Select(rtl, ui.Style("width: auto; display: inline-block"), ui.Option(ui.Text("abc")), ui.Option(ui.Text("a longer option"))),
		ui.Button(rtl, ui.Text("שלום abc")),
		ui.Button(rtl, ui.Style("width: auto; display: inline-block; padding-left: 20px"), ui.Text("abc")),
		ui.Input(rtl, ui.Attr("type", "button"), ui.Attr("value", "abc שלום"), ui.Style("width: auto; display: inline-block")),
		ui.Div(rtl, ui.Input(ui.Attr("type", "checkbox")), ui.Input(ui.Attr("type", "radio")),
			ui.Input(ui.Style("display: inline-block; width: 50px"))),
	)
	size := geom.V2(400, 500)
	landed(t, doc, size)
	r := chrome.Match(t, doc, size)
	t.Logf("%d boxes, max delta %.3f pt", r.Boxes, r.MaxDelta)
}
